package api

import (
	"encoding/json"
	"fmt"
	msgapp "mayfly-go/internal/msg/application"
	msgentity "mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/sys/api/form"
	"mayfly-go/internal/sys/api/vo"
	"mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/captcha"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/otp"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/cryptox"
	"mayfly-go/pkg/utils/jsonx"
	"mayfly-go/pkg/utils/netx"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/utils/timex"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	OtpStatusNone  = -1 // 未启用otp校验
	OtpStatusReg   = 1  // 用户otp secret已注册
	OtpStatusNoReg = 2  // 用户otp secret未注册
)

type Account struct {
	AccountApp  application.Account
	ResourceApp application.Resource
	RoleApp     application.Role
	MsgApp      msgapp.Msg
	ConfigApp   application.Config
}

/**   登录者个人相关操作   **/

// @router /accounts/login [post]
func (a *Account) Login(rc *req.Ctx) {
	loginForm := ginx.BindJsonAndValid(rc.GinCtx, new(form.LoginForm))

	accountLoginSecurity := a.ConfigApp.GetConfig(entity.ConfigKeyAccountLoginSecurity).ToAccountLoginSecurity()
	// 判断是否有开启登录验证码校验
	if accountLoginSecurity.UseCaptcha {
		// 校验验证码
		biz.IsTrue(captcha.Verify(loginForm.Cid, loginForm.Captcha), "验证码错误")
	}

	username := loginForm.Username

	clientIp := getIpAndRegion(rc)
	rc.ReqParam = fmt.Sprintf("username: %s | ip: %s", username, clientIp)

	originPwd, err := cryptox.DefaultRsaDecrypt(loginForm.Password, true)
	biz.ErrIsNilAppendErr(err, "解密密码错误: %s")

	account := &entity.Account{Username: username}
	err = a.AccountApp.GetAccount(account, "Id", "Name", "Username", "Password", "Status", "LastLoginTime", "LastLoginIp", "OtpSecret")

	failCountKey := fmt.Sprintf("account:login:failcount:%s", username)
	nowFailCount := cache.GetInt(failCountKey)
	loginFailCount := accountLoginSecurity.LoginFailCount
	loginFailMin := accountLoginSecurity.LoginFailMin
	biz.IsTrue(nowFailCount < loginFailCount, "登录失败超过%d次, 请%d分钟后再试", loginFailCount, loginFailMin)

	if err != nil || !cryptox.CheckPwdHash(originPwd, account.Password) {
		nowFailCount++
		cache.SetStr(failCountKey, strconv.Itoa(nowFailCount), time.Minute*time.Duration(loginFailMin))
		panic(biz.NewBizErr(fmt.Sprintf("用户名或密码错误【当前登录失败%d次】", nowFailCount)))
	}
	biz.IsTrue(account.IsEnable(), "该账号不可用")

	// 校验密码强度（新用户第一次登录密码与账号名一致）
	biz.IsTrueBy(CheckPasswordLever(originPwd), biz.NewBizErrCode(401, "您的密码安全等级较低，请修改后重新登录"))

	res := map[string]any{
		"name":          account.Name,
		"username":      username,
		"lastLoginTime": account.LastLoginTime,
		"lastLoginIp":   account.LastLoginIp,
	}

	// 默认为不校验otp
	otpStatus := OtpStatusNone
	// 访问系统使用的token
	accessToken := req.CreateToken(account.Id, username)
	// 若系统配置中设置开启otp双因素校验，则进行otp校验
	if accountLoginSecurity.UseOtp {
		otpInfo, otpurl, otpToken := useOtp(account, accountLoginSecurity.OtpIssuer, accessToken)
		otpStatus = otpInfo.OptStatus
		if otpurl != "" {
			res["otpUrl"] = otpurl
		}
		accessToken = otpToken
	} else {
		// 不进行otp二次校验则直接返回accessToken
		// 保存登录消息
		go saveLogin(a.AccountApp, a.MsgApp, account, clientIp)
	}

	// 赋值otp状态
	res["otp"] = otpStatus
	res["token"] = accessToken
	rc.ResData = res
}

func useOtp(account *entity.Account, otpIssuer, accessToken string) (*OtpVerifyInfo, string, string) {
	account.OtpSecretDecrypt()
	otpSecret := account.OtpSecret
	// 修改状态为已注册
	otpStatus := OtpStatusReg
	otpUrl := ""
	// 该token用于otp双因素校验
	token := stringx.Rand(32)
	// 未注册otp secret或重置了秘钥
	if otpSecret == "" || otpSecret == "-" {
		otpStatus = OtpStatusNoReg
		key, err := otp.NewTOTP(otp.GenerateOpts{
			AccountName: account.Username,
			Issuer:      otpIssuer,
		})
		biz.ErrIsNilAppendErr(err, "otp生成失败: %s")
		otpUrl = key.URL()
		otpSecret = key.Secret()
	}
	// 缓存otpInfo, 只有双因素校验通过才可返回真正的accessToken
	otpInfo := &OtpVerifyInfo{
		AccountId:   account.Id,
		Username:    account.Username,
		OptStatus:   otpStatus,
		OtpSecret:   otpSecret,
		AccessToken: accessToken,
	}
	cache.SetStr(fmt.Sprintf("otp:token:%s", token), jsonx.ToStr(otpInfo), time.Minute*time.Duration(3))
	return otpInfo, otpUrl, token
}

// 获取ip与归属地信息
func getIpAndRegion(rc *req.Ctx) string {
	clientIp := rc.GinCtx.ClientIP()
	return fmt.Sprintf("%s %s", clientIp, netx.Ip2Region(clientIp))
}

type OtpVerifyInfo struct {
	AccountId   uint64
	Username    string
	OptStatus   int
	AccessToken string
	OtpSecret   string
}

// OTP双因素校验
func (a *Account) OtpVerify(rc *req.Ctx) {
	otpVerify := new(form.OtpVerfiy)
	ginx.BindJsonAndValid(rc.GinCtx, otpVerify)

	tokenKey := fmt.Sprintf("otp:token:%s", otpVerify.OtpToken)
	otpInfoJson := cache.GetStr(tokenKey)
	biz.NotEmpty(otpInfoJson, "otpToken错误或失效, 请重新登陆获取")
	otpInfo := new(OtpVerifyInfo)
	json.Unmarshal([]byte(otpInfoJson), otpInfo)

	failCountKey := fmt.Sprintf("account:otp:failcount:%d", otpInfo.AccountId)
	failCount := cache.GetInt(failCountKey)
	biz.IsTrue(failCount < 5, "双因素校验失败超过5次, 请10分钟后再试")

	otpStatus := otpInfo.OptStatus
	accessToken := otpInfo.AccessToken
	accountId := otpInfo.AccountId
	otpSecret := otpInfo.OtpSecret

	if !otp.Validate(otpVerify.Code, otpSecret) {
		cache.SetStr(failCountKey, strconv.Itoa(failCount+1), time.Minute*time.Duration(10))
		panic(biz.NewBizErr("双因素认证授权码不正确"))
	}

	// 如果是未注册状态，则更新account表的otpSecret信息
	if otpStatus == OtpStatusNoReg {
		update := &entity.Account{OtpSecret: otpSecret}
		update.Id = accountId
		update.OtpSecretEncrypt()
		a.AccountApp.Update(update)
	}

	la := &entity.Account{Username: otpInfo.Username}
	la.Id = accountId
	go saveLogin(a.AccountApp, a.MsgApp, la, rc.GinCtx.ClientIP())

	cache.Del(tokenKey)
	rc.ResData = accessToken
}

// 获取当前登录用户的菜单与权限码
func (a *Account) GetPermissions(rc *req.Ctx) {
	account := rc.LoginAccount

	var resources vo.AccountResourceVOList
	// 获取账号菜单资源
	a.ResourceApp.GetAccountResources(account.Id, &resources)
	// 菜单树与权限code数组
	var menus vo.AccountResourceVOList
	var permissions []string
	for _, v := range resources {
		if v.Type == entity.ResourceTypeMenu {
			menus = append(menus, v)
		} else {
			permissions = append(permissions, *v.Code)
		}
	}
	// 保存该账号的权限codes
	req.SavePermissionCodes(account.Id, permissions)
	rc.ResData = map[string]any{
		"menus":       menus.ToTrees(0),
		"permissions": permissions,
	}
}

func (a *Account) ChangePassword(rc *req.Ctx) {
	form := new(form.AccountChangePasswordForm)
	ginx.BindJsonAndValid(rc.GinCtx, form)

	originOldPwd, err := cryptox.DefaultRsaDecrypt(form.OldPassword, true)
	biz.ErrIsNilAppendErr(err, "解密旧密码错误: %s")

	account := &entity.Account{Username: form.Username}
	err = a.AccountApp.GetAccount(account, "Id", "Username", "Password", "Status")
	biz.ErrIsNil(err, "旧密码错误")
	biz.IsTrue(cryptox.CheckPwdHash(originOldPwd, account.Password), "旧密码错误")
	biz.IsTrue(account.IsEnable(), "该账号不可用")

	originNewPwd, err := cryptox.DefaultRsaDecrypt(form.NewPassword, true)
	biz.ErrIsNilAppendErr(err, "解密新密码错误: %s")
	biz.IsTrue(CheckPasswordLever(originNewPwd), "密码强度必须8位以上且包含字⺟⼤⼩写+数字+特殊符号")

	updateAccount := new(entity.Account)
	updateAccount.Id = account.Id
	updateAccount.Password = cryptox.PwdHash(originNewPwd)
	a.AccountApp.Update(updateAccount)

	// 赋值loginAccount 主要用于记录操作日志，因为操作日志保存请求上下文没有该信息不保存日志
	rc.LoginAccount = &model.LoginAccount{Id: account.Id, Username: account.Username}
}

func CheckPasswordLever(ps string) bool {
	if len(ps) < 8 {
		return false
	}
	num := `[0-9]{1}`
	a_z := `[a-zA-Z]{1}`
	symbol := `[!@#~$%^&*()+|_.,]{1}`
	if b, err := regexp.MatchString(num, ps); !b || err != nil {
		return false
	}
	if b, err := regexp.MatchString(a_z, ps); !b || err != nil {
		return false
	}
	if b, err := regexp.MatchString(symbol, ps); !b || err != nil {
		return false
	}
	return true
}

// 保存更新账号登录信息
func saveLogin(accountApp application.Account, msgApp msgapp.Msg, account *entity.Account, ip string) {
	// 更新账号最后登录时间
	now := time.Now()
	updateAccount := &entity.Account{LastLoginTime: &now}
	updateAccount.Id = account.Id
	updateAccount.LastLoginIp = ip
	accountApp.Update(updateAccount)

	// 创建登录消息
	loginMsg := &msgentity.Msg{
		RecipientId: int64(account.Id),
		Msg:         fmt.Sprintf("于[%s]-[%s]登录", ip, timex.DefaultFormat(now)),
		Type:        1,
	}
	loginMsg.CreateTime = &now
	loginMsg.Creator = account.Username
	loginMsg.CreatorId = account.Id
	msgApp.Create(loginMsg)
}

// 获取个人账号信息
func (a *Account) AccountInfo(rc *req.Ctx) {
	ap := new(vo.AccountPersonVO)
	// 角色信息
	roles := new([]vo.AccountRoleVO)
	a.RoleApp.GetAccountRoles(rc.LoginAccount.Id, roles)

	ap.Roles = *roles
	rc.ResData = ap
}

// 更新个人账号信息
func (a *Account) UpdateAccount(rc *req.Ctx) {
	updateAccount := ginx.BindJsonAndCopyTo[*entity.Account](rc.GinCtx, new(form.AccountUpdateForm), new(entity.Account))
	// 账号id为登录者账号
	updateAccount.Id = rc.LoginAccount.Id

	if updateAccount.Password != "" {
		biz.IsTrue(CheckPasswordLever(updateAccount.Password), "密码强度必须8位以上且包含字⺟⼤⼩写+数字+特殊符号")
		updateAccount.Password = cryptox.PwdHash(updateAccount.Password)
	}
	a.AccountApp.Update(updateAccount)
}

/**    后台账号操作    **/

// @router /accounts [get]
func (a *Account) Accounts(rc *req.Ctx) {
	condition := &entity.Account{}
	condition.Username = rc.GinCtx.Query("username")
	rc.ResData = a.AccountApp.GetPageList(condition, ginx.GetPageParam(rc.GinCtx), new([]vo.AccountManageVO))
}

// @router /accounts
func (a *Account) SaveAccount(rc *req.Ctx) {
	form := &form.AccountCreateForm{}
	account := ginx.BindJsonAndCopyTo(rc.GinCtx, form, new(entity.Account))

	form.Password = "*****"
	rc.ReqParam = form
	account.SetBaseInfo(rc.LoginAccount)

	if account.Id == 0 {
		a.AccountApp.Create(account)
	} else {
		if account.Password != "" {
			biz.IsTrue(CheckPasswordLever(account.Password), "密码强度必须8位以上且包含字⺟⼤⼩写+数字+特殊符号")
			account.Password = cryptox.PwdHash(account.Password)
		}
		a.AccountApp.Update(account)
	}
}

func (a *Account) ChangeStatus(rc *req.Ctx) {
	g := rc.GinCtx

	account := &entity.Account{}
	account.Id = uint64(ginx.PathParamInt(g, "id"))
	account.Status = int8(ginx.PathParamInt(g, "status"))
	rc.ReqParam = fmt.Sprintf("accountId: %d, status: %d", account.Id, account.Status)
	a.AccountApp.Update(account)
}

func (a *Account) DeleteAccount(rc *req.Ctx) {
	idsStr := ginx.PathParam(rc.GinCtx, "id")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		a.AccountApp.Delete(uint64(value))
	}
}

// 获取账号角色id列表，用户回显角色分配
func (a *Account) AccountRoleIds(rc *req.Ctx) {
	rc.ResData = a.RoleApp.GetAccountRoleIds(uint64(ginx.PathParamInt(rc.GinCtx, "id")))
}

// 获取账号角色id列表，用户回显角色分配
func (a *Account) AccountRoles(rc *req.Ctx) {
	vos := new([]vo.AccountRoleVO)
	a.RoleApp.GetAccountRoles(uint64(ginx.PathParamInt(rc.GinCtx, "id")), vos)
	rc.ResData = vos
}

func (a *Account) AccountResources(rc *req.Ctx) {
	var resources vo.ResourceManageVOList
	// 获取账号菜单资源
	a.ResourceApp.GetAccountResources(uint64(ginx.PathParamInt(rc.GinCtx, "id")), &resources)
	rc.ResData = resources.ToTrees(0)
}

// 保存账号角色信息
func (a *Account) SaveRoles(rc *req.Ctx) {
	g := rc.GinCtx

	var form form.AccountRoleForm
	ginx.BindJsonAndValid(g, &form)
	aid := uint64(form.Id)
	rc.ReqParam = form

	// 将,拼接的字符串进行切割并转换
	newIds := collx.ArrayMap[string, uint64](strings.Split(form.RoleIds, ","), func(val string) uint64 {
		id, _ := strconv.Atoi(val)
		return uint64(id)
	})

	oIds := a.RoleApp.GetAccountRoleIds(uint64(form.Id))

	addIds, delIds, _ := collx.ArrayCompare(newIds, oIds, func(i1, i2 uint64) bool {
		return i1 == i2
	})

	createTime := time.Now()
	creator := rc.LoginAccount.Username
	creatorId := rc.LoginAccount.Id
	for _, v := range addIds {
		rr := &entity.AccountRole{AccountId: aid, RoleId: v, CreateTime: &createTime, CreatorId: creatorId, Creator: creator}
		a.RoleApp.SaveAccountRole(rr)
	}
	for _, v := range delIds {
		a.RoleApp.DeleteAccountRole(aid, v)
	}
}

// 重置otp秘钥
func (a *Account) ResetOtpSecret(rc *req.Ctx) {
	account := &entity.Account{OtpSecret: "-"}
	accountId := uint64(ginx.PathParamInt(rc.GinCtx, "id"))
	account.Id = accountId
	rc.ReqParam = fmt.Sprintf("accountId = %d", accountId)
	a.AccountApp.Update(account)
}

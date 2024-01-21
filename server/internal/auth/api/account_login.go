package api

import (
	"context"
	"encoding/json"
	"fmt"
	"mayfly-go/internal/auth/api/form"
	"mayfly-go/internal/auth/config"
	"mayfly-go/internal/common/utils"
	msgapp "mayfly-go/internal/msg/application"
	sysapp "mayfly-go/internal/sys/application"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/captcha"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/otp"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/cryptox"
	"mayfly-go/pkg/ws"
	"strconv"
	"time"
)

type AccountLogin struct {
	AccountApp sysapp.Account `inject:""`
	MsgApp     msgapp.Msg     `inject:""`
}

/**   用户账号密码登录   **/

// @router /auth/accounts/login [post]
func (a *AccountLogin) Login(rc *req.Ctx) {
	loginForm := ginx.BindJsonAndValid(rc.GinCtx, new(form.LoginForm))

	accountLoginSecurity := config.GetAccountLoginSecurity()
	// 判断是否有开启登录验证码校验
	if accountLoginSecurity.UseCaptcha {
		// 校验验证码
		biz.IsTrue(captcha.Verify(loginForm.Cid, loginForm.Captcha), "验证码错误")
	}

	username := loginForm.Username

	clientIp := getIpAndRegion(rc)
	rc.ReqParam = collx.Kvs("username", username, "ip", clientIp)

	originPwd, err := cryptox.DefaultRsaDecrypt(loginForm.Password, true)
	biz.ErrIsNilAppendErr(err, "解密密码错误: %s")

	account := &sysentity.Account{Username: username}
	err = a.AccountApp.GetBy(account, "Id", "Name", "Username", "Password", "Status", "LastLoginTime", "LastLoginIp", "OtpSecret")

	failCountKey := fmt.Sprintf("account:login:failcount:%s", username)
	nowFailCount := cache.GetInt(failCountKey)
	loginFailCount := accountLoginSecurity.LoginFailCount
	loginFailMin := accountLoginSecurity.LoginFailMin
	biz.IsTrue(nowFailCount < loginFailCount, "登录失败超过%d次, 请%d分钟后再试", loginFailCount, loginFailMin)

	if err != nil || !cryptox.CheckPwdHash(originPwd, account.Password) {
		nowFailCount++
		cache.SetStr(failCountKey, strconv.Itoa(nowFailCount), time.Minute*time.Duration(loginFailMin))
		panic(errorx.NewBiz(fmt.Sprintf("用户名或密码错误【当前登录失败%d次】", nowFailCount)))
	}

	// 校验密码强度（新用户第一次登录密码与账号名一致）
	biz.IsTrueBy(utils.CheckAccountPasswordLever(originPwd), errorx.NewBizCode(401, "您的密码安全等级较低，请修改后重新登录"))
	rc.ResData = LastLoginCheck(account, accountLoginSecurity, clientIp)
}

type OtpVerifyInfo struct {
	AccountId   uint64
	Username    string
	OptStatus   int
	AccessToken string
	OtpSecret   string
}

// OTP双因素校验
func (a *AccountLogin) OtpVerify(rc *req.Ctx) {
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
		panic(errorx.NewBiz("双因素认证授权码不正确"))
	}

	// 如果是未注册状态，则更新account表的otpSecret信息
	if otpStatus == OtpStatusNoReg {
		update := &sysentity.Account{OtpSecret: otpSecret}
		update.Id = accountId
		biz.ErrIsNil(update.OtpSecretEncrypt())
		biz.ErrIsNil(a.AccountApp.Update(context.Background(), update))
	}

	la := &sysentity.Account{Username: otpInfo.Username}
	la.Id = accountId
	go saveLogin(la, getIpAndRegion(rc))

	cache.Del(tokenKey)
	rc.ResData = accessToken
}

func (a *AccountLogin) Logout(rc *req.Ctx) {
	la := rc.GetLoginAccount()
	req.GetPermissionCodeRegistery().Remove(la.Id)
	ws.CloseClient(ws.UserId(la.Id))
}

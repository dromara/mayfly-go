package api

import (
	"fmt"
	"mayfly-go/internal/common/utils"
	msgapp "mayfly-go/internal/msg/application"
	"mayfly-go/internal/sys/api/form"
	"mayfly-go/internal/sys/api/vo"
	"mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/cryptox"
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
	biz.IsTrue(utils.CheckAccountPasswordLever(originNewPwd), "密码强度必须8位以上且包含字⺟⼤⼩写+数字+特殊符号")

	updateAccount := new(entity.Account)
	updateAccount.Id = account.Id
	updateAccount.Password = cryptox.PwdHash(originNewPwd)
	a.AccountApp.Update(updateAccount)

	// 赋值loginAccount 主要用于记录操作日志，因为操作日志保存请求上下文没有该信息不保存日志
	rc.LoginAccount = &model.LoginAccount{Id: account.Id, Username: account.Username}
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
		biz.IsTrue(utils.CheckAccountPasswordLever(updateAccount.Password), "密码强度必须8位以上且包含字⺟⼤⼩写+数字+特殊符号")
		updateAccount.Password = cryptox.PwdHash(updateAccount.Password)
	}

	oldAcc := a.AccountApp.GetById(updateAccount.Id)
	// 账号创建十分钟内允许修改用户名（兼容oauth2首次登录修改用户名），否则不允许修改
	if oldAcc.CreateTime.Add(10 * time.Minute).Before(time.Now()) {
		// 禁止更新用户名，防止误传被更新
		updateAccount.Username = ""
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
			biz.IsTrue(utils.CheckAccountPasswordLever(account.Password), "密码强度必须8位以上且包含字⺟⼤⼩写+数字+特殊符号")
			account.Password = cryptox.PwdHash(account.Password)
		}
		// 更新操作不允许修改用户名、防止误传更新
		account.Username = ""
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
	var form form.AccountRoleForm
	ginx.BindJsonAndValid(rc.GinCtx, &form)
	rc.ReqParam = form

	// 将,拼接的字符串进行切割并转换
	newIds := collx.ArrayMap[string, uint64](strings.Split(form.RoleIds, ","), func(val string) uint64 {
		id, _ := strconv.Atoi(val)
		return uint64(id)
	})

	a.RoleApp.SaveAccountRole(contextx.NewLoginAccount(rc.LoginAccount), form.Id, newIds)
}

// 重置otp秘钥
func (a *Account) ResetOtpSecret(rc *req.Ctx) {
	account := &entity.Account{OtpSecret: "-"}
	accountId := uint64(ginx.PathParamInt(rc.GinCtx, "id"))
	account.Id = accountId
	rc.ReqParam = fmt.Sprintf("accountId = %d", accountId)
	a.AccountApp.Update(account)
}

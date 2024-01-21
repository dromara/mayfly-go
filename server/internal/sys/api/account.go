package api

import (
	"mayfly-go/internal/common/utils"
	msgapp "mayfly-go/internal/msg/application"
	"mayfly-go/internal/sys/api/form"
	"mayfly-go/internal/sys/api/vo"
	"mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/consts"
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
	AccountApp  application.Account  `inject:""`
	ResourceApp application.Resource `inject:""`
	RoleApp     application.Role     `inject:""`
	MsgApp      msgapp.Msg           `inject:""`
	ConfigApp   application.Config   `inject:""`
}

// 获取当前登录用户的菜单与权限码
func (a *Account) GetPermissions(rc *req.Ctx) {
	account := rc.GetLoginAccount()

	var resources vo.AccountResourceVOList
	// 获取账号菜单资源
	biz.ErrIsNil(a.ResourceApp.GetAccountResources(account.Id, &resources))
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
	rc.ResData = collx.M{
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
	err = a.AccountApp.GetBy(account, "Id", "Username", "Password", "Status")
	biz.ErrIsNil(err, "旧密码错误")
	biz.IsTrue(cryptox.CheckPwdHash(originOldPwd, account.Password), "旧密码错误")
	biz.IsTrue(account.IsEnable(), "该账号不可用")

	originNewPwd, err := cryptox.DefaultRsaDecrypt(form.NewPassword, true)
	biz.ErrIsNilAppendErr(err, "解密新密码错误: %s")
	biz.IsTrue(utils.CheckAccountPasswordLever(originNewPwd), "密码强度必须8位以上且包含字⺟⼤⼩写+数字+特殊符号")

	updateAccount := new(entity.Account)
	updateAccount.Id = account.Id
	updateAccount.Password = cryptox.PwdHash(originNewPwd)
	biz.ErrIsNil(a.AccountApp.Update(rc.MetaCtx, updateAccount), "更新账号密码失败")

	// 赋值loginAccount 主要用于记录操作日志，因为操作日志保存请求上下文没有该信息不保存日志
	contextx.WithLoginAccount(rc.MetaCtx, &model.LoginAccount{
		Id:       account.Id,
		Username: account.Username,
	})
}

// 获取个人账号信息
func (a *Account) AccountInfo(rc *req.Ctx) {
	ap := new(vo.AccountPersonVO)
	// 角色信息
	ap.Roles = a.getAccountRoles(rc.GetLoginAccount().Id)
	rc.ResData = ap
}

// 更新个人账号信息
func (a *Account) UpdateAccount(rc *req.Ctx) {
	updateAccount := ginx.BindJsonAndCopyTo[*entity.Account](rc.GinCtx, new(form.AccountUpdateForm), new(entity.Account))
	// 账号id为登录者账号
	updateAccount.Id = rc.GetLoginAccount().Id

	if updateAccount.Password != "" {
		biz.IsTrue(utils.CheckAccountPasswordLever(updateAccount.Password), "密码强度必须8位以上且包含字⺟⼤⼩写+数字+特殊符号")
		updateAccount.Password = cryptox.PwdHash(updateAccount.Password)
	}

	oldAcc, err := a.AccountApp.GetById(new(entity.Account), updateAccount.Id)
	biz.ErrIsNil(err, "账号信息不存在")
	// 账号创建十分钟内允许修改用户名（兼容oauth2首次登录修改用户名），否则不允许修改
	if oldAcc.CreateTime.Add(10 * time.Minute).Before(time.Now()) {
		// 禁止更新用户名，防止误传被更新
		updateAccount.Username = ""
	}
	biz.ErrIsNil(a.AccountApp.Update(rc.MetaCtx, updateAccount))
}

/**    后台账号操作    **/

// @router /accounts [get]
func (a *Account) Accounts(rc *req.Ctx) {
	condition := &entity.Account{}
	condition.Username = rc.GinCtx.Query("username")
	res, err := a.AccountApp.GetPageList(condition, ginx.GetPageParam(rc.GinCtx), new([]vo.AccountManageVO))
	biz.ErrIsNil(err)
	rc.ResData = res
}

// @router /accounts
func (a *Account) SaveAccount(rc *req.Ctx) {
	form := &form.AccountCreateForm{}
	account := ginx.BindJsonAndCopyTo(rc.GinCtx, form, new(entity.Account))

	form.Password = "*****"
	rc.ReqParam = form

	if account.Id == 0 {
		biz.ErrIsNil(a.AccountApp.Create(rc.MetaCtx, account))
	} else {
		if account.Password != "" {
			biz.IsTrue(utils.CheckAccountPasswordLever(account.Password), "密码强度必须8位以上且包含字⺟⼤⼩写+数字+特殊符号")
			account.Password = cryptox.PwdHash(account.Password)
		}
		// 更新操作不允许修改用户名、防止误传更新
		account.Username = ""
		biz.ErrIsNil(a.AccountApp.Update(rc.MetaCtx, account))
	}
}

func (a *Account) ChangeStatus(rc *req.Ctx) {
	g := rc.GinCtx

	account := &entity.Account{}
	account.Id = uint64(ginx.PathParamInt(g, "id"))

	status := entity.AccountStatus(int8(ginx.PathParamInt(g, "status")))
	biz.ErrIsNil(entity.AccountStatusEnum.Valid(status))
	account.Status = status

	rc.ReqParam = collx.Kvs("accountId", account.Id, "status", account.Status)
	biz.ErrIsNil(a.AccountApp.Update(rc.MetaCtx, account))
}

func (a *Account) DeleteAccount(rc *req.Ctx) {
	idsStr := ginx.PathParam(rc.GinCtx, "id")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		biz.ErrIsNilAppendErr(a.AccountApp.Delete(rc.MetaCtx, uint64(value)), "删除失败：%s")
	}
}

// 获取账号角色信息列表
func (a *Account) AccountRoles(rc *req.Ctx) {
	rc.ResData = a.getAccountRoles(uint64(ginx.PathParamInt(rc.GinCtx, "id")))
}

func (a *Account) getAccountRoles(accountId uint64) []*vo.AccountRoleVO {
	vos := make([]*vo.AccountRoleVO, 0)

	accountRoles, err := a.RoleApp.GetAccountRoles(accountId)
	biz.ErrIsNil(err)

	if len(accountRoles) == 0 {
		return vos
	}

	// 获取角色信息进行组装
	roleIds := collx.ArrayMap[*entity.AccountRole, uint64](accountRoles, func(val *entity.AccountRole) uint64 {
		return val.RoleId
	})
	roles, err := a.RoleApp.ListByQuery(&entity.RoleQuery{Ids: roleIds})
	biz.ErrIsNil(err)
	roleId2Role := collx.ArrayToMap[*entity.Role, uint64](roles, func(val *entity.Role) uint64 {
		return val.Id
	})

	for _, ac := range accountRoles {
		role := roleId2Role[ac.RoleId]
		if role == nil {
			continue
		}
		vos = append(vos, &vo.AccountRoleVO{
			RoleId:     ac.RoleId,
			RoleName:   role.Name,
			Code:       role.Code,
			Status:     role.Status,
			CreateTime: ac.CreateTime, // 分配时间
			Creator:    ac.Creator,    // 分配者
		})
	}

	return vos
}

func (a *Account) AccountResources(rc *req.Ctx) {
	var resources vo.ResourceManageVOList
	// 获取账号菜单资源
	biz.ErrIsNil(a.ResourceApp.GetAccountResources(uint64(ginx.PathParamInt(rc.GinCtx, "id")), &resources))
	rc.ResData = resources.ToTrees(0)
}

// 关联账号角色
func (a *Account) RelateRole(rc *req.Ctx) {
	var form form.AccountRoleForm
	ginx.BindJsonAndValid(rc.GinCtx, &form)
	rc.ReqParam = form

	biz.ErrIsNil(a.RoleApp.RelateAccountRole(rc.MetaCtx, form.Id, form.RoleId, consts.AccountRoleRelateType(form.RelateType)))
}

// 重置otp秘钥
func (a *Account) ResetOtpSecret(rc *req.Ctx) {
	account := &entity.Account{OtpSecret: "-"}
	accountId := uint64(ginx.PathParamInt(rc.GinCtx, "id"))
	account.Id = accountId
	rc.ReqParam = collx.Kvs("accountId", accountId)
	biz.ErrIsNil(a.AccountApp.Update(rc.MetaCtx, account))
}

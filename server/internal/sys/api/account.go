package api

import (
	"mayfly-go/internal/pkg/utils"
	"mayfly-go/internal/sys/api/form"
	"mayfly-go/internal/sys/api/vo"
	"mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/consts"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/cryptox"
	"mayfly-go/pkg/utils/structx"
	"strings"
	"time"

	"github.com/may-fly/cast"
)

const (
	OtpStatusNone  = -1 // 未启用otp校验
	OtpStatusReg   = 1  // 用户otp secret已注册
	OtpStatusNoReg = 2  // 用户otp secret未注册
)

type Account struct {
	accountApp  application.Account  `inject:"T"`
	resourceApp application.Resource `inject:"T"`
	roleApp     application.Role     `inject:"T"`
}

func (a *Account) ReqConfs() *req.Confs {
	addAccountPermission := req.NewPermission("account:add")
	reqs := [...]*req.Conf{

		// 获取个人账号的权限资源信息
		req.NewGet("/permissions", a.GetPermissions),

		req.NewPost("/change-pwd", a.ChangePassword).DontNeedToken().Log(req.NewLogSaveI(imsg.LogChangePassword)),

		// 获取个人账号信息
		req.NewGet("/self", a.AccountInfo),

		// 更新个人账号信息
		req.NewPut("/self", a.UpdateAccount),

		/**   后台管理接口  **/

		// 获取所有用户列表
		req.NewGet("", a.Accounts),

		// 获取用户列表信息（只包含最基础信息）
		req.NewGet("/simple", a.SimpleAccounts),

		// 根据username获取账号基础信息
		req.NewGet("/detail", a.AccountDetail),

		req.NewPost("", a.SaveAccount).Log(req.NewLogSaveI(imsg.LogAccountCreate)).RequiredPermission(addAccountPermission),

		req.NewPut("change-status/:id/:status", a.ChangeStatus).Log(req.NewLogSaveI(imsg.LogAccountChangeStatus)).RequiredPermission(addAccountPermission),

		req.NewPut(":id/reset-otp", a.ResetOtpSecret).Log(req.NewLogSaveI(imsg.LogResetOtpSecret)).RequiredPermission(addAccountPermission),

		req.NewDelete(":id", a.DeleteAccount).Log(req.NewLogSaveI(imsg.LogAccountDelete)).RequiredPermissionCode("account:del"),

		// 关联用户角色
		req.NewPost("/roles", a.RelateRole).Log(req.NewLogSaveI(imsg.LogAssignUserRoles)).RequiredPermissionCode("account:saveRoles"),

		// 获取用户角色
		req.NewGet(":id/roles", a.AccountRoles),

		// 获取用户资源列表
		req.NewGet(":id/resources", a.AccountResources),
	}

	return req.NewConfs("sys/accounts", reqs[:]...)
}

// 获取当前登录用户的菜单与权限码
func (a *Account) GetPermissions(rc *req.Ctx) {
	account := rc.GetLoginAccount()

	var resources vo.AccountResourceVOList
	// 获取账号菜单资源
	biz.ErrIsNil(a.resourceApp.GetAccountResources(account.Id, &resources))
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
	ctx := rc.MetaCtx

	form := req.BindJsonAndValid[*form.AccountChangePasswordForm](rc)

	originOldPwd, err := utils.DefaultRsaDecrypt(form.OldPassword, true)
	biz.ErrIsNilAppendErr(err, "Wrong to decrypt old password: %s")

	account := &entity.Account{Username: form.Username}
	err = a.accountApp.GetByCond(model.NewModelCond(account).Columns("Id", "Username", "Password", "Status"))
	biz.ErrIsNilI(ctx, err, imsg.ErrOldPasswordWrong)
	biz.IsTrueI(ctx, cryptox.CheckPwdHash(originOldPwd, account.Password), imsg.ErrOldPasswordWrong)
	biz.IsTrue(account.IsEnable(), "This account is not available")

	originNewPwd, err := utils.DefaultRsaDecrypt(form.NewPassword, true)
	biz.ErrIsNilAppendErr(err, "Wrong to decrypt new password: %s")
	biz.IsTrueI(ctx, utils.CheckAccountPasswordLever(originNewPwd), imsg.ErrAccountPasswordNotFollowRule)

	updateAccount := new(entity.Account)
	updateAccount.Id = account.Id
	updateAccount.Password = cryptox.PwdHash(originNewPwd)
	biz.ErrIsNilAppendErr(a.accountApp.Update(ctx, updateAccount), "failed to update account password: %s")

	// 赋值loginAccount 主要用于记录操作日志，因为操作日志保存请求上下文没有该信息不保存日志
	contextx.WithLoginAccount(ctx, &model.LoginAccount{
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
	form, updateAccount := req.BindJsonAndCopyTo[*form.AccountUpdateForm, *entity.Account](rc)
	// 账号id为登录者账号
	updateAccount.Id = rc.GetLoginAccount().Id
	rc.ReqParam = form

	ctx := rc.MetaCtx
	if updateAccount.Password != "" {
		biz.IsTrueI(ctx, utils.CheckAccountPasswordLever(updateAccount.Password), imsg.ErrAccountPasswordNotFollowRule)
		updateAccount.Password = cryptox.PwdHash(updateAccount.Password)
	}

	oldAcc, err := a.accountApp.GetById(updateAccount.Id)
	biz.ErrIsNilAppendErr(err, "Account does not exist: %s")
	// 账号创建十分钟内允许修改用户名（兼容oauth2首次登录修改用户名），否则不允许修改
	if oldAcc.CreateTime.Add(10 * time.Minute).Before(time.Now()) {
		// 禁止更新用户名，防止误传被更新
		updateAccount.Username = ""
	}
	biz.ErrIsNil(a.accountApp.Update(ctx, updateAccount))
}

/**    后台账号操作    **/

// @router /accounts [get]
func (a *Account) Accounts(rc *req.Ctx) {
	condition := &entity.AccountQuery{}
	condition.Username = rc.Query("username")
	condition.Name = rc.Query("name")
	condition.PageParam = rc.GetPageParam()
	res, err := a.accountApp.GetPageList(condition)
	biz.ErrIsNil(err)
	rc.ResData = model.PageResultConv[*entity.Account, *vo.AccountManageVO](res)
}

func (a *Account) SimpleAccounts(rc *req.Ctx) {
	condition := &entity.AccountQuery{}
	condition.Username = rc.Query("username")
	condition.Name = rc.Query("name")
	idsStr := rc.Query("ids")
	if idsStr != "" {
		condition.Ids = collx.ArrayMap[string, uint64](strings.Split(idsStr, ","), func(val string) uint64 {
			return cast.ToUint64(val)
		})
	}
	condition.PageParam = rc.GetPageParam()
	res, err := a.accountApp.GetPageList(condition)
	biz.ErrIsNil(err)
	rc.ResData = model.PageResultConv[*entity.Account, *vo.SimpleAccountVO](res)
}

// 获取账号详情
func (a *Account) AccountDetail(rc *req.Ctx) {
	username := rc.Query("username")
	biz.NotEmpty(username, "username is required")
	account := &entity.Account{Username: username}
	err := a.accountApp.GetByCond(account)
	biz.ErrIsNilAppendErr(err, "Account does not exist: %s")
	accountvo := new(vo.SimpleAccountVO)
	structx.Copy(accountvo, account)

	accountvo.Roles = a.getAccountRoles(account.Id)
	rc.ResData = accountvo
}

// @router /accounts
func (a *Account) SaveAccount(rc *req.Ctx) {
	form, account := req.BindJsonAndCopyTo[*form.AccountCreateForm, *entity.Account](rc)

	form.Password = "*****"
	rc.ReqParam = form
	ctx := rc.MetaCtx

	if account.Id == 0 {
		biz.NotEmpty(account.Password, "password is required")
		biz.IsTrueI(ctx, utils.CheckAccountPasswordLever(account.Password), imsg.ErrAccountPasswordNotFollowRule)
		account.Password = cryptox.PwdHash(account.Password)
		biz.ErrIsNil(a.accountApp.Create(rc.MetaCtx, account))
	} else {
		if account.Password != "" {
			biz.IsTrueI(ctx, utils.CheckAccountPasswordLever(account.Password), imsg.ErrAccountPasswordNotFollowRule)
			account.Password = cryptox.PwdHash(account.Password)
		}
		// 更新操作不允许修改用户名、防止误传更新
		account.Username = ""
		biz.ErrIsNil(a.accountApp.Update(ctx, account))
	}
}

func (a *Account) ChangeStatus(rc *req.Ctx) {
	account := &entity.Account{}
	account.Id = uint64(rc.PathParamInt("id"))

	status := entity.AccountStatus(int8(rc.PathParamInt("status")))
	biz.ErrIsNil(entity.AccountStatusEnum.Valid(status))
	account.Status = status

	rc.ReqParam = collx.Kvs("accountId", account.Id, "status", account.Status)
	biz.ErrIsNil(a.accountApp.Update(rc.MetaCtx, account))
}

func (a *Account) DeleteAccount(rc *req.Ctx) {
	idsStr := rc.PathParam("id")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		biz.ErrIsNil(a.accountApp.Delete(rc.MetaCtx, cast.ToUint64(v)))
	}
}

// 获取账号角色信息列表
func (a *Account) AccountRoles(rc *req.Ctx) {
	rc.ResData = a.getAccountRoles(uint64(rc.PathParamInt("id")))
}

func (a *Account) getAccountRoles(accountId uint64) []*vo.AccountRoleVO {
	vos := make([]*vo.AccountRoleVO, 0)

	accountRoles, err := a.roleApp.GetAccountRoles(accountId)
	biz.ErrIsNil(err)

	if len(accountRoles) == 0 {
		return vos
	}

	// 获取角色信息进行组装
	roleIds := collx.ArrayMap[*entity.AccountRole, uint64](accountRoles, func(val *entity.AccountRole) uint64 {
		return val.RoleId
	})
	roles, err := a.roleApp.ListByQuery(&entity.RoleQuery{Ids: roleIds})
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
	biz.ErrIsNil(a.resourceApp.GetAccountResources(uint64(rc.PathParamInt("id")), &resources))
	rc.ResData = resources.ToTrees(0)
}

// 关联账号角色
func (a *Account) RelateRole(rc *req.Ctx) {
	form := req.BindJsonAndValid[*form.AccountRoleForm](rc)
	rc.ReqParam = form
	biz.ErrIsNil(a.roleApp.RelateAccountRole(rc.MetaCtx, form.Id, form.RoleId, consts.AccountRoleRelateType(form.RelateType)))
}

// 重置otp秘钥
func (a *Account) ResetOtpSecret(rc *req.Ctx) {
	account := &entity.Account{OtpSecret: "-"}
	accountId := uint64(rc.PathParamInt("id"))
	account.Id = accountId
	rc.ReqParam = collx.Kvs("accountId", accountId)
	biz.ErrIsNil(a.accountApp.Update(rc.MetaCtx, account))
}

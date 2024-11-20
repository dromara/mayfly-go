package router

import (
	"mayfly-go/internal/sys/api"
	"mayfly-go/internal/sys/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitAccountRouter(router *gin.RouterGroup) {
	account := router.Group("sys/accounts")
	a := new(api.Account)
	biz.ErrIsNil(ioc.Inject(a))

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

		// 根据账号id获取账号基础信息
		req.NewGet("/:id", a.AccountDetail),

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

	req.BatchSetGroup(account, reqs[:])
}

package router

import (
	"mayfly-go/internal/sys/api"
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

		req.NewPost("/change-pwd", a.ChangePassword).DontNeedToken().Log(req.NewLogSave("用户修改密码")),

		// 获取个人账号信息
		req.NewGet("/self", a.AccountInfo),

		// 更新个人账号信息
		req.NewPut("/self", a.UpdateAccount),

		/**   后台管理接口  **/

		// 获取所有用户列表
		req.NewGet("", a.Accounts),

		req.NewPost("", a.SaveAccount).Log(req.NewLogSave("保存账号信息")).RequiredPermission(addAccountPermission),

		req.NewPut("change-status/:id/:status", a.ChangeStatus).Log(req.NewLogSave("修改账号状态")).RequiredPermission(addAccountPermission),

		req.NewPut(":id/reset-otp", a.ResetOtpSecret).Log(req.NewLogSave("重置OTP密钥")).RequiredPermission(addAccountPermission),

		req.NewDelete(":id", a.DeleteAccount).Log(req.NewLogSave("删除账号")).RequiredPermissionCode("account:del"),

		// 关联用户角色
		req.NewPost("/roles", a.RelateRole).Log(req.NewLogSave("关联用户角色")).RequiredPermissionCode("account:saveRoles"),

		// 获取用户角色
		req.NewGet(":id/roles", a.AccountRoles),

		// 获取用户资源列表
		req.NewGet(":id/resources", a.AccountResources),
	}

	req.BatchSetGroup(account, reqs[:])
}

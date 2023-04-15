package router

import (
	"mayfly-go/internal/sys/api"
	"mayfly-go/internal/sys/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitAccountRouter(router *gin.RouterGroup) {
	account := router.Group("sys/accounts")
	a := &api.Account{
		AccountApp:  application.GetAccountApp(),
		ResourceApp: application.GetResourceApp(),
		RoleApp:     application.GetRoleApp(),
		MsgApp:      application.GetMsgApp(),
		ConfigApp:   application.GetConfigApp(),
	}
	{
		// 用户登录
		loginLog := req.NewLogInfo("用户登录").WithSave(true)
		account.POST("login", func(g *gin.Context) {
			req.NewCtxWithGin(g).
				DontNeedToken().
				WithLog(loginLog).
				Handle(a.Login)
		})

		// 获取个人账号的权限资源信息
		account.GET("/permissions", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(a.GetPermissions)
		})

		changePwdLog := req.NewLogInfo("用户修改密码").WithSave(true)
		account.POST("change-pwd", func(g *gin.Context) {
			req.NewCtxWithGin(g).
				DontNeedToken().
				WithLog(changePwdLog).
				Handle(a.ChangePassword)
		})

		// 获取个人账号信息
		account.GET("/self", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(a.AccountInfo)
		})

		// 更新个人账号信息
		account.PUT("/self", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(a.UpdateAccount)
		})

		account.GET("/msgs", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(a.GetMsgs)
		})

		/**   后台管理接口  **/

		// 获取所有用户列表
		account.GET("", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(a.Accounts)
		})

		createAccount := req.NewLogInfo("保存账号信息").WithSave(true)
		addAccountPermission := req.NewPermission("account:add")
		account.POST("", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithRequiredPermission(addAccountPermission).
				WithLog(createAccount).
				Handle(a.SaveAccount)
		})

		changeStatus := req.NewLogInfo("修改账号状态").WithSave(true)
		account.PUT("change-status/:id/:status", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(changeStatus).
				Handle(a.ChangeStatus)
		})

		delAccount := req.NewLogInfo("删除账号").WithSave(true)
		delAccountPermission := req.NewPermission("account:del")
		account.DELETE(":id", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithRequiredPermission(delAccountPermission).
				WithLog(delAccount).
				Handle(a.DeleteAccount)
		})

		// 获取所有用户角色id列表
		account.GET(":id/roleIds", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(a.AccountRoleIds)
		})

		// 保存用户角色
		saveAccountRole := req.NewLogInfo("保存用户角色").WithSave(true)
		sarPermission := req.NewPermission("account:saveRoles")
		account.POST("/roles", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(saveAccountRole).
				WithRequiredPermission(sarPermission).
				Handle(a.SaveRoles)
		})

		// 获取用户角色
		account.GET(":id/roles", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(a.AccountRoles)
		})

		// 获取用户资源列表
		account.GET(":id/resources", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(a.AccountResources)
		})
	}
}

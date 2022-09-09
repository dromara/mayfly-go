package router

import (
	"mayfly-go/internal/sys/api"
	"mayfly-go/internal/sys/application"
	"mayfly-go/pkg/ctx"

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
		loginLog := ctx.NewLogInfo("用户登录").WithSave(true)
		account.POST("login", func(g *gin.Context) {
			ctx.NewReqCtxWithGin(g).
				WithNeedToken(false).
				WithLog(loginLog).
				Handle(a.Login)
		})

		changePwdLog := ctx.NewLogInfo("用户修改密码").WithSave(true)
		account.POST("change-pwd", func(g *gin.Context) {
			ctx.NewReqCtxWithGin(g).
				WithNeedToken(false).
				WithLog(changePwdLog).
				Handle(a.ChangePassword)
		})

		// 获取个人账号信息
		account.GET("/self", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(a.AccountInfo)
		})

		// 更新个人账号信息
		account.PUT("/self", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(a.UpdateAccount)
		})

		account.GET("/msgs", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(a.GetMsgs)
		})

		/**   后台管理接口  **/

		// 获取所有用户列表
		account.GET("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(a.Accounts)
		})

		createAccount := ctx.NewLogInfo("创建账号").WithSave(true)
		addAccountPermission := ctx.NewPermission("account:add")
		account.POST("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithRequiredPermission(addAccountPermission).
				WithLog(createAccount).
				Handle(a.SaveAccount)
		})

		changeStatus := ctx.NewLogInfo("修改账号状态").WithSave(true)
		account.PUT("change-status/:id/:status", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(changeStatus).
				Handle(a.ChangeStatus)
		})

		delAccount := ctx.NewLogInfo("删除账号").WithSave(true)
		delAccountPermission := ctx.NewPermission("account:del")
		account.DELETE(":id", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithRequiredPermission(delAccountPermission).
				WithLog(delAccount).
				Handle(a.DeleteAccount)
		})

		// 获取所有用户角色id列表
		account.GET(":id/roleIds", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(a.AccountRoleIds)
		})

		// 保存用户角色
		saveAccountRole := ctx.NewLogInfo("保存用户角色").WithSave(true)
		sarPermission := ctx.NewPermission("account:saveRoles")
		account.POST("/roles", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(saveAccountRole).
				WithRequiredPermission(sarPermission).
				Handle(a.SaveRoles)
		})

		// 获取用户角色
		account.GET(":id/roles", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(a.AccountRoles)
		})

		// 获取用户资源列表
		account.GET(":id/resources", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(a.AccountResources)
		})
	}
}

package routers

import (
	"mayfly-go/base/ctx"
	"mayfly-go/server/sys/api"
	"mayfly-go/server/sys/application"

	"github.com/gin-gonic/gin"
)

func InitAccountRouter(router *gin.RouterGroup) {
	account := router.Group("sys/accounts")
	a := &api.Account{
		AccountApp:  application.AccountApp,
		ResourceApp: application.ResourceApp,
		RoleApp:     application.RoleApp,
		MsgApp:      application.MsgApp,
	}
	{
		// 用户登录
		account.POST("login", func(g *gin.Context) {
			rc := ctx.NewReqCtxWithGin(g).WithNeedToken(false).WithLog(ctx.NewLogInfo("用户登录"))
			rc.Handle(a.Login)
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

		createAccount := ctx.NewLogInfo("创建账号")
		addAccountPermission := ctx.NewPermission("account:add")
		account.POST("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithRequiredPermission(addAccountPermission).
				WithLog(createAccount).
				Handle(a.CreateAccount)
		})

		changeStatus := ctx.NewLogInfo("修改账号状态")
		account.PUT("change-status/:id/:status", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(changeStatus).
				Handle(a.ChangeStatus)
		})

		delAccount := ctx.NewLogInfo("删除账号")
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
		saveAccountRole := ctx.NewLogInfo("保存用户角色")
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

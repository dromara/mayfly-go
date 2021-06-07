package routers

import (
	"mayfly-go/base/ctx"
	"mayfly-go/server/sys/apis"
	"mayfly-go/server/sys/application"

	"github.com/gin-gonic/gin"
)

func InitAccountRouter(router *gin.RouterGroup) {
	account := router.Group("sys/accounts")
	a := &apis.Account{
		AccountApp:  application.Account,
		ResourceApp: application.Resource,
		RoleApp:     application.Role,
	}
	{
		// 用户登录
		account.POST("login", func(g *gin.Context) {
			rc := ctx.NewReqCtxWithGin(g).WithNeedToken(false).WithLog(ctx.NewLogInfo("用户登录"))
			rc.Handle(a.Login)
		})

		// 获取所有用户列表
		account.GET("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(a.Accounts)
		})

		// 获取所有用户角色id列表
		account.GET(":id/roleIds", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(a.AccountRoleIds)
		})

		saveAccountRole := ctx.NewLogInfo("保存用户角色")
		// 保存用户角色
		account.POST("/roles", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(saveAccountRole).Handle(a.SaveRoles)
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

package routers

import (
	"mayfly-go/base/ctx"
	"mayfly-go/devops/apis"
	"mayfly-go/devops/application"

	"github.com/gin-gonic/gin"
)

func InitAccountRouter(router *gin.RouterGroup) {
	account := router.Group("accounts")
	a := &apis.Account{AccountApp: application.Account}
	{
		// 用户登录
		account.POST("login", func(g *gin.Context) {
			rc := ctx.NewReqCtxWithGin(g).WithNeedToken(false).WithLog(ctx.NewLogInfo("用户登录"))
			rc.Handle(a.Login)
		})
		// 获取所有用户列表
		account.GET("", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c).WithLog(ctx.NewLogInfo("获取账号列表"))
			rc.Handle(a.Accounts)
		})
	}
}

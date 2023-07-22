package router

import "github.com/gin-gonic/gin"

func Init(router *gin.RouterGroup) {
	InitCaptchaRouter(router)
	InitAccountRouter(router) // 注册account路由
	InitResourceRouter(router)
	InitRoleRouter(router)
	InitSystemRouter(router)
	InitSyslogRouter(router)
	InitSysConfigRouter(router)
}

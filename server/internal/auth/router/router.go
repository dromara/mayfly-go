package router

import "github.com/gin-gonic/gin"

func Init(router *gin.RouterGroup) {
	InitCaptcha(router)
	InitAccount(router)
	InitOauth2(router)
	InitLdap(router)
}

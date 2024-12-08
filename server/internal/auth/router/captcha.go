package router

import (
	"mayfly-go/internal/auth/api"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitCaptcha(router *gin.RouterGroup) {
	captcha := router.Group("sys/captcha")
	{
		req.NewGet("", api.GenerateCaptcha).DontNeedToken().Group(captcha)
	}
}

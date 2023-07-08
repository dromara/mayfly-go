package router

import (
	"mayfly-go/internal/sys/api"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitCaptchaRouter(router *gin.RouterGroup) {
	captcha := router.Group("sys/captcha")
	{
		req.NewGet("", api.GenerateCaptcha).DontNeedToken().Group(captcha)
	}
}

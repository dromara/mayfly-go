package routers

import (
	"mayfly-go/base/ctx"
	"mayfly-go/server/sys/api"

	"github.com/gin-gonic/gin"
)

func InitCaptchaRouter(router *gin.RouterGroup) {
	captcha := router.Group("sys/captcha")
	{
		captcha.GET("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithNeedToken(false).Handle(api.GenerateCaptcha)
		})
	}
}

package router

import (
	"mayfly-go/internal/common/api"
	"mayfly-go/pkg/ctx"

	"github.com/gin-gonic/gin"
)

func InitCommonRouter(router *gin.RouterGroup) {
	common := router.Group("common")
	c := &api.Common{}
	{
		// 获取公钥
		common.GET("public-key", func(g *gin.Context) {
			ctx.NewReqCtxWithGin(g).
				WithNeedToken(false).
				Handle(c.RasPublicKey)
		})
	}
}

package router

import (
	"mayfly-go/internal/common/api"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitCommonRouter(router *gin.RouterGroup) {
	common := router.Group("common")
	c := &api.Common{}
	{
		// 获取公钥
		req.NewGet("public-key", c.RasPublicKey).DontNeedToken().Group(common)
	}
}

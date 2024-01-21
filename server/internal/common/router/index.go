package router

import (
	"mayfly-go/internal/common/api"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitIndexRouter(router *gin.RouterGroup) {
	index := router.Group("common/index")
	i := new(api.Index)
	biz.ErrIsNil(ioc.Inject(i))
	{
		// 首页基本信息统计
		req.NewGet("count", i.Count).Group(index)
	}
}

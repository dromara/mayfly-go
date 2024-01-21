package router

import (
	"mayfly-go/internal/sys/api"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"

	"github.com/gin-gonic/gin"
)

func InitSystemRouter(router *gin.RouterGroup) {
	sys := router.Group("sysmsg")
	s := new(api.System)
	biz.ErrIsNil(ioc.Inject(s))

	{
		sys.GET("", s.ConnectWs)
	}
}

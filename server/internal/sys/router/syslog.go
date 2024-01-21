package router

import (
	"mayfly-go/internal/sys/api"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitSyslogRouter(router *gin.RouterGroup) {
	sysG := router.Group("syslogs")
	s := new(api.Syslog)
	biz.ErrIsNil(ioc.Inject(s))

	req.NewGet("", s.Syslogs).Group(sysG)
}

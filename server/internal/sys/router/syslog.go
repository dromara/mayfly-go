package router

import (
	"mayfly-go/internal/sys/api"
	"mayfly-go/internal/sys/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitSyslogRouter(router *gin.RouterGroup) {
	s := &api.Syslog{
		SyslogApp: application.GetSyslogApp(),
	}
	sysG := router.Group("syslogs")

	req.NewGet("", s.Syslogs).Group(sysG)
}

package router

import (
	"github.com/gin-gonic/gin"
	"mayfly-go/internal/sys/api"
	"mayfly-go/internal/sys/application"
	"mayfly-go/pkg/req"
)

func InitSysAuthRouter(router *gin.RouterGroup) {
	r := &api.Auth{
		ConfigApp: application.GetConfigApp(),
	}
	rg := router.Group("sys/auth")

	baseP := req.NewPermission("system:auth:base")

	reqs := [...]*req.Conf{
		req.NewGet("", r.GetInfo).RequiredPermission(baseP),
		req.NewPut("/oauth2", r.SaveOAuth2).RequiredPermission(baseP),
	}

	req.BatchSetGroup(rg, reqs[:])
}

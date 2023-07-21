package router

import (
	"github.com/gin-gonic/gin"
	msgapp "mayfly-go/internal/msg/application"
	"mayfly-go/internal/sys/api"
	"mayfly-go/internal/sys/application"
	"mayfly-go/pkg/req"
)

func InitSysAuthRouter(router *gin.RouterGroup) {
	r := &api.Auth{
		ConfigApp:  application.GetConfigApp(),
		AuthApp:    application.GetAuthApp(),
		AccountApp: application.GetAccountApp(),
		MsgApp:     msgapp.GetMsgApp(),
	}
	rg := router.Group("sys/auth")

	baseP := req.NewPermission("system:auth:base")

	reqs := [...]*req.Conf{
		req.NewGet("", r.GetInfo).RequiredPermission(baseP),

		req.NewPut("/oauth2", r.SaveOAuth2).RequiredPermission(baseP),

		req.NewGet("/oauth2/login", r.OAuth2Login).DontNeedToken(),
		req.NewGet("/oauth2/callback", r.OAuth2Callback).NoRes().DontNeedToken(),
	}

	req.BatchSetGroup(rg, reqs[:])
}

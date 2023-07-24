package router

import (
	"mayfly-go/internal/auth/api"
	"mayfly-go/internal/auth/application"
	msgapp "mayfly-go/internal/msg/application"
	sysapp "mayfly-go/internal/sys/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func Init(router *gin.RouterGroup) {
	accountLogin := &api.AccountLogin{
		ConfigApp:  sysapp.GetConfigApp(),
		AccountApp: sysapp.GetAccountApp(),
		MsgApp:     msgapp.GetMsgApp(),
	}

	oauth2Login := &api.Oauth2Login{
		Oauth2App:  application.GetAuthApp(),
		ConfigApp:  sysapp.GetConfigApp(),
		AccountApp: sysapp.GetAccountApp(),
		MsgApp:     msgapp.GetMsgApp(),
	}

	rg := router.Group("/auth")

	reqs := [...]*req.Conf{

		// 用户账号密码登录
		req.NewPost("/accounts/login", accountLogin.Login).Log(req.NewLogSave("用户登录")).DontNeedToken(),

		// 用户退出登录
		req.NewPost("/accounts/logout", accountLogin.Logout),

		// 用户otp双因素校验
		req.NewPost("/accounts/otp-verify", accountLogin.OtpVerify).DontNeedToken(),

		/*--------oauth2登录相关----------*/

		// oauth2登录
		req.NewGet("/oauth2/login", oauth2Login.OAuth2Login).DontNeedToken(),

		req.NewGet("/oauth2/bind", oauth2Login.OAuth2Bind),

		// oauth2回调地址
		req.NewGet("/oauth2/callback", oauth2Login.OAuth2Callback).Log(req.NewLogSave("oauth2回调")).DontNeedToken(),

		req.NewGet("/oauth2/status", oauth2Login.Oauth2Status),

		req.NewGet("/oauth2/unbind", oauth2Login.Oauth2Unbind).Log(req.NewLogSave("oauth2解绑")),
	}

	req.BatchSetGroup(rg, reqs[:])
}

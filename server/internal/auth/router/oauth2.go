package router

import (
	"mayfly-go/internal/auth/api"
	"mayfly-go/internal/auth/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitOauth2(router *gin.RouterGroup) {
	oauth2Login := new(api.Oauth2Login)
	biz.ErrIsNil(ioc.Inject(oauth2Login))

	rg := router.Group("/auth/oauth2")

	reqs := [...]*req.Conf{

		req.NewGet("/config", oauth2Login.Oauth2Config).DontNeedToken(),

		// oauth2登录
		req.NewGet("/login", oauth2Login.OAuth2Login).DontNeedToken(),

		req.NewGet("/bind", oauth2Login.OAuth2Bind),

		// oauth2回调地址
		req.NewGet("/callback", oauth2Login.OAuth2Callback).Log(req.NewLogSaveI(imsg.LogOauth2Callback)).DontNeedToken(),

		req.NewGet("/status", oauth2Login.Oauth2Status),

		req.NewGet("/unbind", oauth2Login.Oauth2Unbind).Log(req.NewLogSaveI(imsg.LogOauth2Unbind)),
	}

	req.BatchSetGroup(rg, reqs[:])
}

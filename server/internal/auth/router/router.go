package router

import (
	"mayfly-go/internal/auth/api"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func Init(router *gin.RouterGroup) {
	accountLogin := new(api.AccountLogin)
	biz.ErrIsNil(ioc.Inject(accountLogin))

	ldapLogin := new(api.LdapLogin)
	biz.ErrIsNil(ioc.Inject(ldapLogin))

	oauth2Login := new(api.Oauth2Login)
	biz.ErrIsNil(ioc.Inject(oauth2Login))

	rg := router.Group("/auth")

	reqs := [...]*req.Conf{

		// 用户账号密码登录
		req.NewPost("/accounts/login", accountLogin.Login).Log(req.NewLogSave("用户登录")).DontNeedToken(),

		// 用户退出登录
		req.NewPost("/accounts/logout", accountLogin.Logout),

		// 用户otp双因素校验
		req.NewPost("/accounts/otp-verify", accountLogin.OtpVerify).DontNeedToken(),

		/*--------oauth2登录相关----------*/

		req.NewGet("/oauth2-config", oauth2Login.Oauth2Config).DontNeedToken(),

		// oauth2登录
		req.NewGet("/oauth2/login", oauth2Login.OAuth2Login).DontNeedToken(),

		req.NewGet("/oauth2/bind", oauth2Login.OAuth2Bind),

		// oauth2回调地址
		req.NewGet("/oauth2/callback", oauth2Login.OAuth2Callback).Log(req.NewLogSave("oauth2回调")).DontNeedToken(),

		req.NewGet("/oauth2/status", oauth2Login.Oauth2Status),

		req.NewGet("/oauth2/unbind", oauth2Login.Oauth2Unbind).Log(req.NewLogSave("oauth2解绑")),

		// LDAP 登录
		req.NewGet("/ldap/enabled", ldapLogin.GetLdapEnabled).DontNeedToken(),
		req.NewPost("/ldap/login", ldapLogin.Login).Log(req.NewLogSave("LDAP 登录")).DontNeedToken(),
	}

	req.BatchSetGroup(rg, reqs[:])
}

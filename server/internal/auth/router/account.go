package router

import (
	"mayfly-go/internal/auth/api"
	"mayfly-go/internal/auth/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitAccount(router *gin.RouterGroup) {
	accountLogin := new(api.AccountLogin)
	biz.ErrIsNil(ioc.Inject(accountLogin))

	ldapLogin := new(api.LdapLogin)
	biz.ErrIsNil(ioc.Inject(ldapLogin))

	rg := router.Group("/auth/accounts")

	reqs := [...]*req.Conf{

		// 用户账号密码登录
		req.NewPost("/login", accountLogin.Login).Log(req.NewLogSaveI(imsg.LogAccountLogin)).DontNeedToken(),

		req.NewGet("/refreshToken", accountLogin.RefreshToken).DontNeedToken(),

		// 用户退出登录
		req.NewPost("/logout", accountLogin.Logout),

		// 用户otp双因素校验
		req.NewPost("/otp-verify", accountLogin.OtpVerify).DontNeedToken(),
	}

	req.BatchSetGroup(rg, reqs[:])
}

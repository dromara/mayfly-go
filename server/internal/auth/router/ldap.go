package router

import (
	"mayfly-go/internal/auth/api"
	"mayfly-go/internal/auth/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitLdap(router *gin.RouterGroup) {
	ldapLogin := new(api.LdapLogin)
	biz.ErrIsNil(ioc.Inject(ldapLogin))

	rg := router.Group("/auth/ldap")

	reqs := [...]*req.Conf{
		req.NewGet("/enabled", ldapLogin.GetLdapEnabled).DontNeedToken(),
		req.NewPost("/login", ldapLogin.Login).Log(req.NewLogSaveI(imsg.LogLdapLogin)).DontNeedToken(),
	}

	req.BatchSetGroup(rg, reqs[:])
}

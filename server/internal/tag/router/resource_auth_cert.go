package router

import (
	"mayfly-go/internal/tag/api"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitResourceAuthCertRouter(router *gin.RouterGroup) {
	m := new(api.ResourceAuthCert)
	biz.ErrIsNil(ioc.Inject(m))

	resourceAuthCert := router.Group("/auth-certs")
	{
		reqs := [...]*req.Conf{
			req.NewGet("", m.ListByQuery),
		}

		req.BatchSetGroup(resourceAuthCert, reqs[:])
	}
}

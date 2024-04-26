package router

import (
	"mayfly-go/internal/tag/api"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitResourceOpLogRouter(router *gin.RouterGroup) {
	m := new(api.ResourceOpLog)
	biz.ErrIsNil(ioc.Inject(m))

	resourceOpLog := router.Group("/resource-op-logs")
	{
		reqs := [...]*req.Conf{
			req.NewGet("/account", m.PageAccountOpLog),
		}

		req.BatchSetGroup(resourceOpLog, reqs[:])
	}
}

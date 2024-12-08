package router

import (
	"mayfly-go/internal/flow/api"
	"mayfly-go/internal/flow/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitProcdefouter(router *gin.RouterGroup) {
	p := new(api.Procdef)
	biz.ErrIsNil(ioc.Inject(p))

	reqGroup := router.Group("/flow/procdefs")
	{
		reqs := [...]*req.Conf{
			req.NewGet("", p.GetProcdefPage),

			req.NewGet("/:resourceType/:resourceCode", p.GetProcdef),

			req.NewPost("", p.Save).Log(req.NewLogSaveI(imsg.LogProcdefSave)).RequiredPermissionCode("flow:procdef:save"),

			req.NewDelete(":id", p.Delete).Log(req.NewLogSaveI(imsg.LogProcdefDelete)).RequiredPermissionCode("flow:procdef:del"),
		}

		req.BatchSetGroup(reqGroup, reqs[:])
	}
}

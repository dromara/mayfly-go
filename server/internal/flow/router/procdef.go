package router

import (
	"mayfly-go/internal/flow/api"
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

			req.NewGet("/:key", p.GetProcdef),

			req.NewPost("", p.Save).Log(req.NewLogSave("流程定义-保存")).RequiredPermissionCode("flow:procdef:save"),

			req.NewDelete(":id", p.Delete).Log(req.NewLogSave("流程定义-删除")).RequiredPermissionCode("flow:procdef:del"),
		}

		req.BatchSetGroup(reqGroup, reqs[:])
	}
}

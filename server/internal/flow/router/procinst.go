package router

import (
	"mayfly-go/internal/flow/api"
	"mayfly-go/internal/flow/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitProcinstRouter(router *gin.RouterGroup) {
	p := new(api.Procinst)
	biz.ErrIsNil(ioc.Inject(p))

	reqGroup := router.Group("/flow/procinsts")
	{
		reqs := [...]*req.Conf{
			req.NewGet("", p.GetProcinstPage),

			req.NewGet("/:id", p.GetProcinstDetail),

			req.NewPost("/start", p.ProcinstStart).Log(req.NewLogSaveI(imsg.LogProcinstStart)),

			req.NewPost("/:id/cancel", p.ProcinstCancel).Log(req.NewLogSaveI(imsg.LogProcinstCancel)),

			req.NewGet("/tasks", p.GetTasks),

			req.NewPost("/tasks/complete", p.CompleteTask).Log(req.NewLogSaveI(imsg.LogCompleteTask)),

			req.NewPost("/tasks/reject", p.RejectTask).Log(req.NewLogSaveI(imsg.LogRejectTask)),

			req.NewPost("/tasks/back", p.BackTask).Log(req.NewLogSaveI(imsg.LogBackTask)),
		}

		req.BatchSetGroup(reqGroup, reqs[:])
	}
}

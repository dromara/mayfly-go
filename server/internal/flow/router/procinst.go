package router

import (
	"mayfly-go/internal/flow/api"
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

			req.NewPost("/:id/cancel", p.ProcinstCancel).Log(req.NewLogSave("流程-取消")),

			req.NewGet("/tasks", p.GetTasks),

			req.NewPost("/tasks/complete", p.CompleteTask).Log(req.NewLogSave("流程-任务完成")),

			req.NewPost("/tasks/reject", p.RejectTask).Log(req.NewLogSave("流程-任务拒绝")),

			req.NewPost("/tasks/back", p.BackTask).Log(req.NewLogSave("流程-任务驳回")),
		}

		req.BatchSetGroup(reqGroup, reqs[:])
	}
}

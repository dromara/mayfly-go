package router

import (
	"mayfly-go/internal/machine/api"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitMachineCronJobRouter(router *gin.RouterGroup) {
	cronjobs := router.Group("machine-cronjobs")

	cj := new(api.MachineCronJob)
	biz.ErrIsNil(ioc.Inject(cj))

	reqs := [...]*req.Conf{
		// 获取机器任务列表
		req.NewGet("", cj.MachineCronJobs),

		req.NewGet("/machine-ids", cj.GetRelateMachineIds),

		req.NewGet("/cronjob-ids", cj.GetRelateCronJobIds),

		req.NewPost("", cj.Save).Log(req.NewLogSave("保存机器计划任务")),

		req.NewDelete(":ids", cj.Delete).Log(req.NewLogSave("删除机器计划任务")),

		req.NewPost("/run/:key", cj.RunCronJob).Log(req.NewLogSave("手动执行计划任务")),

		req.NewGet("/execs", cj.CronJobExecs),
	}

	req.BatchSetGroup(cronjobs, reqs[:])
}

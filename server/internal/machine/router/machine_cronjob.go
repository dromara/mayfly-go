package router

import (
	"mayfly-go/internal/machine/api"
	"mayfly-go/internal/machine/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitMachineCronJobRouter(router *gin.RouterGroup) {
	cronjobs := router.Group("machine-cronjobs")
	cj := &api.MachineCronJob{
		MachineCronJobApp: application.GetMachineCronJobApp(),
	}

	reqs := [...]*req.Conf{
		// 获取机器任务列表
		req.NewGet("", cj.MachineCronJobs),

		req.NewGet("/machine-ids", cj.GetRelateMachineIds),

		req.NewGet("/cronjob-ids", cj.GetRelateCronJobIds),

		req.NewPost("", cj.Save).Log(req.NewLogSave("保存机器计划任务")),

		req.NewDelete(":ids", cj.Delete).Log(req.NewLogSave("删除机器计划任务")),

		req.NewGet("/execs", cj.CronJobExecs),
	}

	req.BatchSetGroup(cronjobs, reqs[:])
}

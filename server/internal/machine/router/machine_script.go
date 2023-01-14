package router

import (
	"mayfly-go/internal/machine/api"
	"mayfly-go/internal/machine/application"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitMachineScriptRouter(router *gin.RouterGroup) {
	machines := router.Group("machines")
	{
		ms := &api.MachineScript{
			MachineScriptApp: application.GetMachineScriptApp(),
			MachineApp:       application.GetMachineApp(),
			TagApp:           tagapp.GetTagTreeApp(),
		}

		// 获取指定机器脚本列表
		machines.GET(":machineId/scripts", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(ms.MachineScripts)
		})

		saveMachienScriptLog := req.NewLogInfo("机器-保存脚本").WithSave(true)
		smsP := req.NewPermission("machine:script:save")
		// 保存脚本
		machines.POST(":machineId/scripts", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(saveMachienScriptLog).
				WithRequiredPermission(smsP).
				Handle(ms.SaveMachineScript)
		})

		deleteLog := req.NewLogInfo("机器-删除脚本").WithSave(true)
		dP := req.NewPermission("machine:script:del")
		// 保存脚本
		machines.DELETE(":machineId/scripts/:scriptId", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(deleteLog).
				WithRequiredPermission(dP).
				Handle(ms.DeleteMachineScript)
		})

		runLog := req.NewLogInfo("机器-执行脚本").WithSave(true)
		rP := req.NewPermission("machine:script:run")
		// 运行脚本
		machines.GET(":machineId/scripts/:scriptId/run", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(runLog).
				WithRequiredPermission(rP).
				Handle(ms.RunMachineScript)
		})
	}
}

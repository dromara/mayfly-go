package routers

import (
	"mayfly-go/base/ctx"
	"mayfly-go/server/devops/api"
	"mayfly-go/server/devops/application"

	"github.com/gin-gonic/gin"
)

func InitMachineScriptRouter(router *gin.RouterGroup) {
	machines := router.Group("machines")
	{
		ms := &api.MachineScript{
			MachineScriptApp: application.MachineScriptApp,
			MachineApp:       application.MachineApp}

		// 获取指定机器脚本列表
		machines.GET(":machineId/scripts", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(ms.MachineScripts)
		})

		saveMachienScriptLog := ctx.NewLogInfo("保存脚本")
		smsP := ctx.NewPermission("machine:script:save")
		// 保存脚本
		machines.POST(":machineId/scripts", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(saveMachienScriptLog).
				WithRequiredPermission(smsP).
				Handle(ms.SaveMachineScript)
		})

		deleteLog := ctx.NewLogInfo("删除脚本")
		dP := ctx.NewPermission("machine:script:del")
		// 保存脚本
		machines.DELETE(":machineId/scripts/:scriptId", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(deleteLog).
				WithRequiredPermission(dP).
				Handle(ms.DeleteMachineScript)
		})

		runLog := ctx.NewLogInfo("执行机器脚本")
		rP := ctx.NewPermission("machine:script:run")
		// 运行脚本
		machines.GET(":machineId/scripts/:scriptId/run", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(runLog).
				WithRequiredPermission(rP).
				Handle(ms.RunMachineScript)
		})
	}
}

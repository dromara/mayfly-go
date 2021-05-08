package routers

import (
	"mayfly-go/base/ctx"
	"mayfly-go/devops/apis"
	"mayfly-go/devops/application"

	"github.com/gin-gonic/gin"
)

func InitMachineScriptRouter(router *gin.RouterGroup) {
	machines := router.Group("machines")
	{
		ms := &apis.MachineScript{
			MachineScriptApp: application.MachineScript,
			MachineApp:       application.Machine}

		// 获取指定机器脚本列表
		machines.GET(":machineId/scripts", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithNeedToken(false).Handle(ms.MachineScripts)
		})

		saveMachienScriptLog := ctx.NewLogInfo("保存脚本")
		// 保存脚本
		machines.POST(":machineId/scripts", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c).WithLog(saveMachienScriptLog)
			rc.Handle(ms.SaveMachineScript)
		})

		deleteLog := ctx.NewLogInfo("删除脚本")
		// 保存脚本
		machines.DELETE(":machineId/scripts/:scriptId", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c).WithLog(deleteLog)
			rc.Handle(ms.DeleteMachineScript)
		})

		runLog := ctx.NewLogInfo("执行机器脚本")
		// 运行脚本
		machines.GET(":machineId/scripts/:scriptId/run", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c).WithLog(runLog)
			rc.Handle(ms.RunMachineScript)
		})
	}
}

package router

import (
	"mayfly-go/base/ctx"
	"mayfly-go/server/devops/api"
	"mayfly-go/server/devops/application"

	"github.com/gin-gonic/gin"
)

func InitMachineRouter(router *gin.RouterGroup) {
	m := &api.Machine{MachineApp: application.MachineApp}
	machines := router.Group("machines")
	{
		machines.GET("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(m.Machines)
		})

		saveMachine := ctx.NewLogInfo("保存机器信息")
		machines.POST("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(saveMachine).
				Handle(m.SaveMachine)
		})

		delMachine := ctx.NewLogInfo("删除机器")
		machines.DELETE(":machineId", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(delMachine).
				Handle(m.DeleteMachine)
		})

		closeCli := ctx.NewLogInfo("关闭机器客户端")
		machines.DELETE(":machineId/close-cli", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(closeCli).Handle(m.CloseCli)
		})

		machines.GET(":machineId/terminal", m.WsSSH)
	}
}

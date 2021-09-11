package routers

import (
	"mayfly-go/base/ctx"
	"mayfly-go/server/devops/api"
	"mayfly-go/server/devops/application"

	"github.com/gin-gonic/gin"
)

func InitMachineRouter(router *gin.RouterGroup) {
	m := &api.Machine{MachineApp: application.MachineApp}
	db := router.Group("machines")
	{
		db.GET("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(m.Machines)
		})

		saveMachine := ctx.NewLogInfo("保存机器信息")
		db.POST("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(saveMachine).
				Handle(m.SaveMachine)
		})

		delMachine := ctx.NewLogInfo("删除机器")
		db.DELETE(":machineId", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(delMachine).
				Handle(m.DeleteMachine)
		})

		closeCli := ctx.NewLogInfo("关闭机器客户端")
		db.DELETE(":machineId/close-cli", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(closeCli).Handle(m.CloseCli)
		})

		db.GET(":machineId/terminal", m.WsSSH)
	}
}

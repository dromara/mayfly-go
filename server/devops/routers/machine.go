package routers

import (
	"mayfly-go/base/ctx"
	"mayfly-go/server/devops/apis"
	"mayfly-go/server/devops/application"

	"github.com/gin-gonic/gin"
)

func InitMachineRouter(router *gin.RouterGroup) {
	m := &apis.Machine{MachineApp: application.MachineApp}
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
		db.DELETE("/delete/:id", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(delMachine).
				Handle(m.DeleteMachine)
		})

		db.GET(":machineId/top", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(m.Top)
		})

		db.GET(":machineId/terminal", m.WsSSH)
	}
}

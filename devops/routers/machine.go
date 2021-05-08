package routers

import (
	"mayfly-go/base/ctx"
	"mayfly-go/devops/apis"
	"mayfly-go/devops/application"

	"github.com/gin-gonic/gin"
)

func InitMachineRouter(router *gin.RouterGroup) {
	m := &apis.Machine{MachineApp: application.Machine}
	db := router.Group("machines")
	{
		db.GET("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(m.Machines)
		})

		db.POST("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(m.SaveMachine)
		})

		db.GET(":machineId/top", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(m.Top)
		})

		db.GET(":machineId/terminal", m.WsSSH)
	}
}

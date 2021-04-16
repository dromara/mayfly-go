package routers

import (
	"mayfly-go/base/ctx"
	"mayfly-go/devops/controllers"

	"github.com/gin-gonic/gin"
)

func InitMachineRouter(router *gin.RouterGroup) {
	// web.Router("/api/machines", machine, "get:Machines")
	// web.Router("/api/machines/?:machineId/run", machine, "get:Run")
	// web.Router("/api/machines/?:machineId/top", machine, "get:Top")
	// web.Router("/api/machines/?:machineId/sysinfo", machine, "get:SysInfo")
	// web.Router("/api/machines/?:machineId/process", machine, "get:GetProcessByName")
	// web.Router("/api/machines/?:machineId/terminal", machine, "get:WsSSH")
	db := router.Group("machines")
	{
		db.GET("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(controllers.Machines)
		})

		db.GET(":machineId/run", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c).WithLog(ctx.NewLogInfo("执行机器命令"))
			rc.Handle(controllers.Run)
		})

		db.GET(":machineId/top", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(controllers.Top)
		})

		db.GET(":machineId/sysinfo", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(controllers.SysInfo)
		})

		db.GET(":machineId/process", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(controllers.GetProcessByName)
		})

		db.GET(":machineId/terminal", controllers.WsSSH)
	}
}

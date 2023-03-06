package router

import (
	"mayfly-go/internal/machine/api"
	"mayfly-go/internal/machine/application"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitMachineRouter(router *gin.RouterGroup) {
	m := &api.Machine{
		MachineApp: application.GetMachineApp(),
		TagApp:     tagapp.GetTagTreeApp(),
	}

	machines := router.Group("machines")
	{
		machines.GET("", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(m.Machines)
		})

		machines.GET(":machineId/stats", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(m.MachineStats)
		})

		machines.GET(":machineId/process", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(m.GetProcess)
		})

		// 终止进程
		killProcessL := req.NewLogInfo("终止进程").WithSave(true)
		killProcessP := req.NewPermission("machine:killprocess")
		machines.DELETE(":machineId/process", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(killProcessL).
				WithRequiredPermission(killProcessP).
				Handle(m.KillProcess)
		})

		saveMachine := req.NewLogInfo("保存机器信息").WithSave(true)
		saveMachineP := req.NewPermission("machine:update")
		machines.POST("", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(saveMachine).
				WithRequiredPermission(saveMachineP).
				Handle(m.SaveMachine)
		})

		machines.POST("test-conn", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				Handle(m.TestConn)
		})

		changeStatus := req.NewLogInfo("调整机器状态").WithSave(true)
		machines.PUT(":machineId/:status", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(changeStatus).
				Handle(m.ChangeStatus)
		})

		delMachine := req.NewLogInfo("删除机器").WithSave(true)
		machines.DELETE(":machineId", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(delMachine).
				Handle(m.DeleteMachine)
		})

		closeCli := req.NewLogInfo("关闭机器客户端").WithSave(true)
		machines.DELETE(":machineId/close-cli", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(closeCli).Handle(m.CloseCli)
		})

		machines.GET(":machineId/terminal", m.WsSSH)

		// 获取机器终端回放记录的相应文件夹名或文件名,目前具有保存机器信息的权限标识才有权限查看终端回放
		machines.GET("rec/names", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithRequiredPermission(saveMachineP).
				Handle(m.MachineRecDirNames)
		})
	}
}

package router

import (
	"mayfly-go/internal/machine/api"
	"mayfly-go/internal/machine/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitMachineRouter(router *gin.RouterGroup) {
	m := new(api.Machine)
	biz.ErrIsNil(ioc.Inject(m))

	dashbord := new(api.Dashbord)
	biz.ErrIsNil(ioc.Inject(dashbord))

	machines := router.Group("machines")
	{
		saveMachineP := req.NewPermission("machine:update")

		reqs := [...]*req.Conf{
			req.NewGet("dashbord", dashbord.Dashbord),

			req.NewGet("", m.Machines),

			req.NewGet("/simple", m.SimpleMachieInfo),

			req.NewGet(":machineId/stats", m.MachineStats),

			req.NewGet(":machineId/process", m.GetProcess),

			req.NewGet(":machineId/users", m.GetUsers),

			req.NewGet(":machineId/groups", m.GetGroups),

			req.NewPost("", m.SaveMachine).Log(req.NewLogSaveI(imsg.LogMachineSave)).RequiredPermission(saveMachineP),

			req.NewDelete(":machineId", m.DeleteMachine).Log(req.NewLogSaveI(imsg.LogMachineSave)),

			req.NewPost("test-conn", m.TestConn),

			req.NewPut(":machineId/:status", m.ChangeStatus).Log(req.NewLogSaveI(imsg.LogMachineChangeStatus)).RequiredPermission(saveMachineP),

			req.NewDelete(":machineId/process", m.KillProcess).Log(req.NewLogSaveI(imsg.LogMachineKillProcess)).RequiredPermissionCode("machine:killprocess"),

			// 获取机器终端回放记录列表,目前具有保存机器信息的权限标识才有权限查看终端回放
			req.NewGet(":machineId/term-recs", m.MachineTermOpRecords).RequiredPermission(saveMachineP),
		}

		req.BatchSetGroup(machines, reqs[:])

		// 终端连接
		machines.GET("terminal/:ac", m.WsSSH)

		// 终端连接
		machines.GET("rdp/:ac", m.WsGuacamole)
	}
}

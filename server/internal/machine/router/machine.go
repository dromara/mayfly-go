package router

import (
	"mayfly-go/internal/machine/api"
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

			req.NewGet(":machineId/stats", m.MachineStats),

			req.NewGet(":machineId/process", m.GetProcess),

			req.NewDelete(":machineId/process", m.KillProcess).Log(req.NewLogSave("终止进程")).RequiredPermissionCode("machine:killprocess"),

			req.NewPost("", m.SaveMachine).Log(req.NewLogSave("保存机器信息")).RequiredPermission(saveMachineP),

			req.NewPost("test-conn", m.TestConn),

			req.NewPut(":machineId/:status", m.ChangeStatus).Log(req.NewLogSave("调整机器状态")).RequiredPermission(saveMachineP),

			req.NewDelete(":machineId", m.DeleteMachine).Log(req.NewLogSave("删除机器")),

			req.NewDelete(":machineId/close-cli", m.CloseCli).Log(req.NewLogSave("关闭机器客户端")).RequiredPermissionCode("machine:close-cli"),

			// 获取机器终端回放记录列表,目前具有保存机器信息的权限标识才有权限查看终端回放
			req.NewGet(":machineId/term-recs", m.MachineTermOpRecords).RequiredPermission(saveMachineP),

			// 获取机器终端回放记录
			req.NewGet(":machineId/term-recs/:recId", m.MachineTermOpRecord).RequiredPermission(saveMachineP),
		}

		req.BatchSetGroup(machines, reqs[:])

		// 终端连接
		machines.GET(":machineId/terminal", m.WsSSH)
	}
}

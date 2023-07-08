package router

import (
	"mayfly-go/internal/machine/api"
	"mayfly-go/internal/machine/application"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitMachineScriptRouter(router *gin.RouterGroup) {
	machines := router.Group("machines")
	ms := &api.MachineScript{
		MachineScriptApp: application.GetMachineScriptApp(),
		MachineApp:       application.GetMachineApp(),
		TagApp:           tagapp.GetTagTreeApp(),
	}

	reqs := [...]*req.Conf{
		// 获取指定机器脚本列表
		req.NewGet(":machineId/scripts", ms.MachineScripts),

		req.NewPost(":machineId/scripts", ms.SaveMachineScript).Log(req.NewLogSave("机器-保存脚本")).RequiredPermissionCode("machine:script:save"),

		req.NewDelete(":machineId/scripts/:scriptId", ms.DeleteMachineScript).Log(req.NewLogSave("机器-删除脚本")).RequiredPermissionCode("machine:script:del"),

		req.NewGet(":machineId/scripts/:scriptId/run", ms.RunMachineScript).Log(req.NewLogSave("机器-执行脚本")).RequiredPermissionCode("machine:script:run"),
	}

	req.BatchSetGroup(machines, reqs[:])

}

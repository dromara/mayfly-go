package router

import (
	"mayfly-go/internal/machine/api"
	"mayfly-go/internal/machine/application"
	msgapp "mayfly-go/internal/msg/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitMachineFileRouter(router *gin.RouterGroup) {
	machineFile := router.Group("machines")

	mf := &api.MachineFile{
		MachineFileApp: application.GetMachineFileApp(),
		MsgApp:         msgapp.GetMsgApp(),
	}

	reqs := [...]*req.Conf{
		// 获取指定机器文件列表
		req.NewGet(":machineId/files", mf.MachineFiles),

		req.NewPost(":machineId/files", mf.SaveMachineFiles).Log(req.NewLogSave("机器-新增文件配置")).RequiredPermissionCode("machine:file:add"),

		req.NewDelete(":machineId/files/:fileId", mf.DeleteFile).Log(req.NewLogSave("机器-删除文件配置")).RequiredPermissionCode("machine:file:del"),

		req.NewGet(":machineId/files/:fileId/read", mf.ReadFileContent).Log(req.NewLogSave("机器-获取文件内容")),

		req.NewGet(":machineId/files/:fileId/read-dir", mf.GetDirEntry).Log(req.NewLogSave("机器-获取目录")),

		req.NewGet(":machineId/files/:fileId/dir-size", mf.GetDirSize),

		req.NewGet(":machineId/files/:fileId/file-stat", mf.GetFileStat),

		req.NewPost(":machineId/files/:fileId/write", mf.WriteFileContent).Log(req.NewLogSave("机器-修改文件内容")).RequiredPermissionCode("machine:file:write"),

		req.NewPost(":machineId/files/:fileId/create-file", mf.CreateFile).Log(req.NewLogSave("机器-创建文件or目录")),

		req.NewPost(":machineId/files/:fileId/upload", mf.UploadFile).Log(req.NewLogSave("机器-文件上传")).RequiredPermissionCode("machine:file:upload"),

		req.NewDelete(":machineId/files/:fileId/remove", mf.RemoveFile).Log(req.NewLogSave("机器-删除文件or文件夹")).RequiredPermissionCode("machine:file:rm"),
	}

	req.BatchSetGroup(machineFile, reqs[:])
}

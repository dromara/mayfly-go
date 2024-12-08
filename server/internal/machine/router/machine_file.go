package router

import (
	"mayfly-go/internal/machine/api"
	"mayfly-go/internal/machine/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitMachineFileRouter(router *gin.RouterGroup) {
	machineFile := router.Group("machines")

	mf := new(api.MachineFile)
	biz.ErrIsNil(ioc.Inject(mf))

	reqs := [...]*req.Conf{
		// 获取指定机器文件列表
		req.NewGet(":machineId/files", mf.MachineFiles),

		req.NewPost(":machineId/files", mf.SaveMachineFiles).Log(req.NewLogSaveI(imsg.LogMachineFileConfSave)).RequiredPermissionCode("machine:file:add"),

		req.NewDelete(":machineId/files/:fileId", mf.DeleteFile).Log(req.NewLogSaveI(imsg.LogMachineFileConfDelete)).RequiredPermissionCode("machine:file:del"),

		req.NewGet(":machineId/files/:fileId/read", mf.ReadFileContent).Log(req.NewLogSaveI(imsg.LogMachineFileRead)),

		req.NewGet(":machineId/files/:fileId/download", mf.DownloadFile).NoRes().Log(req.NewLogSaveI(imsg.LogMachineFileDownload)),

		req.NewGet(":machineId/files/:fileId/read-dir", mf.GetDirEntry),

		req.NewGet(":machineId/files/:fileId/dir-size", mf.GetDirSize),

		req.NewGet(":machineId/files/:fileId/file-stat", mf.GetFileStat),

		req.NewPost(":machineId/files/:fileId/write", mf.WriteFileContent).Log(req.NewLogSaveI(imsg.LogMachineFileModify)).RequiredPermissionCode("machine:file:write"),

		req.NewPost(":machineId/files/:fileId/create-file", mf.CreateFile).Log(req.NewLogSaveI(imsg.LogMachineFileCreate)),

		req.NewPost(":machineId/files/:fileId/upload", mf.UploadFile).Log(req.NewLogSaveI(imsg.LogMachineFileUpload)).RequiredPermissionCode("machine:file:upload"),

		req.NewPost(":machineId/files/:fileId/upload-folder", mf.UploadFolder).Log(req.NewLogSaveI(imsg.LogMachineFileUploadFolder)).RequiredPermissionCode("machine:file:upload"),

		req.NewPost(":machineId/files/:fileId/remove", mf.RemoveFile).Log(req.NewLogSaveI(imsg.LogMachineFileDelete)).RequiredPermissionCode("machine:file:rm"),

		req.NewPost(":machineId/files/:fileId/cp", mf.CopyFile).Log(req.NewLogSaveI(imsg.LogMachineFileCopy)).RequiredPermissionCode("machine:file:rm"),

		req.NewPost(":machineId/files/:fileId/mv", mf.MvFile).Log(req.NewLogSaveI(imsg.LogMachineFileMove)).RequiredPermissionCode("machine:file:rm"),

		req.NewPost(":machineId/files/:fileId/rename", mf.Rename).Log(req.NewLogSaveI(imsg.LogMachineFileRename)).RequiredPermissionCode("machine:file:write"),
	}

	req.BatchSetGroup(machineFile, reqs[:])
}

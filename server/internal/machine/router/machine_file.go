package router

import (
	"mayfly-go/internal/machine/api"
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

		req.NewPost(":machineId/files", mf.SaveMachineFiles).Log(req.NewLogSave("机器-新增文件配置")).RequiredPermissionCode("machine:file:add"),

		req.NewDelete(":machineId/files/:fileId", mf.DeleteFile).Log(req.NewLogSave("机器-删除文件配置")).RequiredPermissionCode("machine:file:del"),

		req.NewGet(":machineId/files/:fileId/read", mf.ReadFileContent).Log(req.NewLogSave("机器-读取文件内容")),

		req.NewGet(":machineId/files/:fileId/download", mf.DownloadFile).NoRes().Log(req.NewLogSave("机器-文件下载")),

		req.NewGet(":machineId/files/:fileId/read-dir", mf.GetDirEntry),

		req.NewGet(":machineId/files/:fileId/dir-size", mf.GetDirSize),

		req.NewGet(":machineId/files/:fileId/file-stat", mf.GetFileStat),

		req.NewPost(":machineId/files/:fileId/write", mf.WriteFileContent).Log(req.NewLogSave("机器-修改文件内容")).RequiredPermissionCode("machine:file:write"),

		req.NewPost(":machineId/files/:fileId/create-file", mf.CreateFile).Log(req.NewLogSave("机器-创建文件or目录")),

		req.NewPost(":machineId/files/:fileId/upload", mf.UploadFile).Log(req.NewLogSave("机器-文件上传")).RequiredPermissionCode("machine:file:upload"),

		req.NewPost(":machineId/files/:fileId/upload-folder", mf.UploadFolder).Log(req.NewLogSave("机器-文件夹上传")).RequiredPermissionCode("machine:file:upload"),

		req.NewPost(":machineId/files/:fileId/remove", mf.RemoveFile).Log(req.NewLogSave("机器-删除文件or文件夹")).RequiredPermissionCode("machine:file:rm"),

		req.NewPost(":machineId/files/:fileId/cp", mf.CopyFile).Log(req.NewLogSave("机器-拷贝文件")).RequiredPermissionCode("machine:file:rm"),

		req.NewPost(":machineId/files/:fileId/mv", mf.MvFile).Log(req.NewLogSave("机器-移动文件")).RequiredPermissionCode("machine:file:rm"),

		req.NewPost(":machineId/files/:fileId/rename", mf.Rename).Log(req.NewLogSave("机器-文件重命名")).RequiredPermissionCode("machine:file:write"),
	}

	req.BatchSetGroup(machineFile, reqs[:])
}

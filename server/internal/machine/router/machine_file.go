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
	{
		mf := &api.MachineFile{
			MachineFileApp: application.GetMachineFileApp(),
			MsgApp:         msgapp.GetMsgApp(),
		}

		// 获取指定机器文件列表
		machineFile.GET(":machineId/files", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(mf.MachineFiles)
		})

		// 新增修改机器文件
		addFileConf := req.NewLogInfo("机器-新增文件配置").WithSave(true)
		afcP := req.NewPermission("machine:file:add")
		machineFile.POST(":machineId/files", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(addFileConf).
				WithRequiredPermission(afcP).
				Handle(mf.SaveMachineFiles)
		})

		// 删除机器文件
		delFileConf := req.NewLogInfo("机器-删除文件配置").WithSave(true)
		dfcP := req.NewPermission("machine:file:del")
		machineFile.DELETE(":machineId/files/:fileId", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(delFileConf).
				WithRequiredPermission(dfcP).
				Handle(mf.DeleteFile)
		})

		getContent := req.NewLogInfo("机器-获取文件内容").WithSave(true)
		machineFile.GET(":machineId/files/:fileId/read", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(getContent).
				Handle(mf.ReadFileContent)
		})

		getDir := req.NewLogInfo("机器-获取目录")
		machineFile.GET(":machineId/files/:fileId/read-dir", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(getDir).
				Handle(mf.GetDirEntry)
		})

		machineFile.GET(":machineId/files/:fileId/dir-size", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(mf.GetDirSize)
		})

		machineFile.GET(":machineId/files/:fileId/file-stat", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(mf.GetFileStat)
		})

		writeFile := req.NewLogInfo("机器-修改文件内容").WithSave(true)
		wfP := req.NewPermission("machine:file:write")
		machineFile.POST(":machineId/files/:fileId/write", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(writeFile).
				WithRequiredPermission(wfP).
				Handle(mf.WriteFileContent)
		})

		createFile := req.NewLogInfo("机器-创建文件or目录").WithSave(true)
		machineFile.POST(":machineId/files/:fileId/create-file", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(createFile).
				WithRequiredPermission(wfP).
				Handle(mf.CreateFile)
		})

		uploadFile := req.NewLogInfo("机器-文件上传").WithSave(true)
		ufP := req.NewPermission("machine:file:upload")
		machineFile.POST(":machineId/files/:fileId/upload", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(uploadFile).
				WithRequiredPermission(ufP).
				Handle(mf.UploadFile)
		})

		removeFile := req.NewLogInfo("机器-删除文件or文件夹").WithSave(true)
		rfP := req.NewPermission("machine:file:rm")
		machineFile.DELETE(":machineId/files/:fileId/remove", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(removeFile).
				WithRequiredPermission(rfP).
				Handle(mf.RemoveFile)
		})
	}
}

package router

import (
	"mayfly-go/internal/machine/api"
	"mayfly-go/internal/machine/application"
	sysApplication "mayfly-go/internal/sys/application"
	"mayfly-go/pkg/ctx"

	"github.com/gin-gonic/gin"
)

func InitMachineFileRouter(router *gin.RouterGroup) {
	machineFile := router.Group("machines")
	{
		mf := &api.MachineFile{
			MachineFileApp: application.GetMachineFileApp(),
			MachineApp:     application.GetMachineApp(),
			MsgApp:         sysApplication.GetMsgApp(),
		}

		// 获取指定机器文件列表
		machineFile.GET(":machineId/files", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(mf.MachineFiles)
		})

		// 新增修改机器文件
		addFileConf := ctx.NewLogInfo("新增机器文件配置").WithSave(true)
		afcP := ctx.NewPermission("machine:file:add")
		machineFile.POST(":machineId/files", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(addFileConf).
				WithRequiredPermission(afcP).
				Handle(mf.SaveMachineFiles)
		})

		// 删除机器文件
		delFileConf := ctx.NewLogInfo("删除机器文件配置").WithSave(true)
		dfcP := ctx.NewPermission("machine:file:del")
		machineFile.DELETE(":machineId/files/:fileId", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(delFileConf).
				WithRequiredPermission(dfcP).
				Handle(mf.DeleteFile)
		})

		getContent := ctx.NewLogInfo("读取机器文件内容").WithSave(true)
		machineFile.GET(":machineId/files/:fileId/read", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(getContent).
				Handle(mf.ReadFileContent)
		})

		getDir := ctx.NewLogInfo("读取机器目录")
		machineFile.GET(":machineId/files/:fileId/read-dir", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(getDir).
				Handle(mf.GetDirEntry)
		})

		writeFile := ctx.NewLogInfo("写入or下载文件内容").WithSave(true)
		wfP := ctx.NewPermission("machine:file:write")
		machineFile.POST(":machineId/files/:fileId/write", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(writeFile).
				WithRequiredPermission(wfP).
				Handle(mf.WriteFileContent)
		})

		createFile := ctx.NewLogInfo("创建机器文件or目录").WithSave(true)
		machineFile.POST(":machineId/files/:fileId/create-file", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(createFile).
				WithRequiredPermission(wfP).
				Handle(mf.CreateFile)
		})

		uploadFile := ctx.NewLogInfo("文件上传").WithSave(true)
		ufP := ctx.NewPermission("machine:file:upload")
		machineFile.POST(":machineId/files/:fileId/upload", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(uploadFile).
				WithRequiredPermission(ufP).
				Handle(mf.UploadFile)
		})

		removeFile := ctx.NewLogInfo("删除文件or文件夹").WithSave(true)
		rfP := ctx.NewPermission("machine:file:rm")
		machineFile.DELETE(":machineId/files/:fileId/remove", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(removeFile).
				WithRequiredPermission(rfP).
				Handle(mf.RemoveFile)
		})
	}
}

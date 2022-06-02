package router

import (
	"mayfly-go/internal/devops/api"
	"mayfly-go/internal/devops/application"
	sysApplication "mayfly-go/internal/sys/application"
	"mayfly-go/pkg/ctx"

	"github.com/gin-gonic/gin"
)

func InitMachineFileRouter(router *gin.RouterGroup) {
	machineFile := router.Group("machines")
	{
		mf := &api.MachineFile{
			MachineFileApp: application.MachineFileApp,
			MachineApp:     application.MachineApp,
			MsgApp:         sysApplication.MsgApp,
		}

		// 获取指定机器文件列表
		machineFile.GET(":machineId/files", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(mf.MachineFiles)
		})

		// 新增修改机器文件
		addFileConf := ctx.NewLogInfo("新增机器文件配置")
		afcP := ctx.NewPermission("machine:file:add")
		machineFile.POST(":machineId/files", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(addFileConf).
				WithRequiredPermission(afcP).
				Handle(mf.SaveMachineFiles)
		})

		// 删除机器文件
		delFileConf := ctx.NewLogInfo("删除机器文件配置")
		dfcP := ctx.NewPermission("machine:file:del")
		machineFile.DELETE(":machineId/files/:fileId", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(delFileConf).
				WithRequiredPermission(dfcP).
				Handle(mf.DeleteFile)
		})

		getContent := ctx.NewLogInfo("读取机器文件内容")
		machineFile.GET(":machineId/files/:fileId/read", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c).WithLog(getContent)
			rc.Handle(mf.ReadFileContent)
		})

		getDir := ctx.NewLogInfo("读取机器目录")
		machineFile.GET(":machineId/files/:fileId/read-dir", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c).WithLog(getDir)
			rc.Handle(mf.GetDirEntry)
		})

		writeFile := ctx.NewLogInfo("写入or下载文件内容")
		wfP := ctx.NewPermission("machine:file:write")
		machineFile.POST(":machineId/files/:fileId/write", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(writeFile).
				WithRequiredPermission(wfP).
				Handle(mf.WriteFileContent)
		})

		uploadFile := ctx.NewLogInfo("文件上传")
		ufP := ctx.NewPermission("machine:file:upload")
		machineFile.POST(":machineId/files/:fileId/upload", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(uploadFile).
				WithRequiredPermission(ufP).
				Handle(mf.UploadFile)
		})

		removeFile := ctx.NewLogInfo("删除文件or文件夹")
		rfP := ctx.NewPermission("machine:file:rm")
		machineFile.DELETE(":machineId/files/:fileId/remove", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(removeFile).
				WithRequiredPermission(rfP).
				Handle(mf.RemoveFile)
		})
	}
}

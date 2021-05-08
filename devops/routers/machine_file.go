package routers

import (
	"mayfly-go/base/ctx"
	"mayfly-go/devops/apis"
	"mayfly-go/devops/application"

	"github.com/gin-gonic/gin"
)

func InitMachineFileRouter(router *gin.RouterGroup) {
	machineFile := router.Group("machines")
	{
		mf := &apis.MachineFile{
			MachineFileApp: application.MachineFile,
			MachineApp:     application.Machine}

		// 获取指定机器文件列表
		machineFile.GET(":machineId/files", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(mf.MachineFiles)
		})

		// 新增修改机器文件
		machineFile.POST(":machineId/files", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(mf.SaveMachineFiles)
		})

		// 删除机器文件
		machineFile.DELETE(":machineId/files/:fileId", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(mf.DeleteFile)
		})

		getContent := ctx.NewLogInfo("读取机器文件内容")
		machineFile.GET(":machineId/files/:fileId/read", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c).WithNeedToken(false).WithLog(getContent)
			rc.Handle(mf.ReadFileContent)
		})

		getDir := ctx.NewLogInfo("读取机器目录")
		machineFile.GET(":machineId/files/:fileId/read-dir", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c).WithLog(getDir)
			rc.Handle(mf.GetDirEntry)
		})

		writeFile := ctx.NewLogInfo("写入or下载文件内容")
		machineFile.POST(":machineId/files/:fileId/write", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c).WithLog(writeFile)
			rc.Handle(mf.WriteFileContent)
		})

		uploadFile := ctx.NewLogInfo("文件上传")
		machineFile.POST(":machineId/files/:fileId/upload", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c).WithLog(uploadFile)
			rc.Handle(mf.UploadFile)
		})

		removeFile := ctx.NewLogInfo("删除文件or文件夹")
		machineFile.DELETE(":machineId/files/:fileId/remove", func(c *gin.Context) {
			rc := ctx.NewReqCtxWithGin(c).WithLog(removeFile)
			rc.Handle(mf.RemoveFile)
		})
	}
}

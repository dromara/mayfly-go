package router

import (
	"mayfly-go/internal/db/api"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitDbTransferRouter(router *gin.RouterGroup) {
	instances := router.Group("/dbTransfer")

	d := new(api.DbTransferTask)
	biz.ErrIsNil(ioc.Inject(d))

	reqs := [...]*req.Conf{
		// 获取任务列表 /datasync
		req.NewGet("", d.Tasks),

		// 保存任务 /datasync/save
		req.NewPost("save", d.SaveTask).Log(req.NewLogSaveI(imsg.LogDtsSave)).RequiredPermissionCode("db:transfer:save"),

		// 删除任务 /datasync/:taskId/del
		req.NewDelete(":taskId/del", d.DeleteTask).Log(req.NewLogSaveI(imsg.LogDtsDelete)).RequiredPermissionCode("db:transfer:del"),

		// 启停用任务 /datasync/status
		req.NewPost(":taskId/status", d.ChangeStatus).Log(req.NewLogSaveI(imsg.LogDtsChangeStatus)).RequiredPermissionCode("db:transfer:status"),

		// 立即执行任务 /datasync/run
		req.NewPost(":taskId/run", d.Run).Log(req.NewLogI(imsg.LogDtsRun)).RequiredPermissionCode("db:transfer:run"),

		// 停止正在执行中的任务
		req.NewPost(":taskId/stop", d.Stop).Log(req.NewLogSaveI(imsg.LogDtsStop)).RequiredPermissionCode("db:transfer:run"),

		// 导出文件管理-列表
		req.NewGet("/files/:taskId", d.Files),

		// 导出文件管理-删除
		req.NewPost("/files/del/:fileId", d.FileDel).Log(req.NewLogSaveI(imsg.LogDtsDeleteFile)).RequiredPermissionCode("db:transfer:files:del"),

		req.NewPost("/files/run", d.FileRun).Log(req.NewLogSaveI(imsg.LogDtsRunSqlFile)).RequiredPermissionCode("db:transfer:files:run"),
	}

	req.BatchSetGroup(instances, reqs[:])
}

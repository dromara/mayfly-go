package router

import (
	"mayfly-go/internal/db/api"
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
		req.NewPost("save", d.SaveTask).Log(req.NewLogSave("dts-保存数据迁移任务信息")).RequiredPermissionCode("db:transfer:save"),

		// 删除任务 /datasync/:taskId/del
		req.NewDelete(":taskId/del", d.DeleteTask).Log(req.NewLogSave("dts-删除数据迁移任务信息")).RequiredPermissionCode("db:transfer:del"),

		// 启停用任务 /datasync/status
		req.NewPost(":taskId/status", d.ChangeStatus).Log(req.NewLogSave("dts-启停任务")).RequiredPermissionCode("db:transfer:status"),

		// 立即执行任务 /datasync/run
		req.NewPost(":taskId/run", d.Run).Log(req.NewLog("dts-执行数据迁移任务")).RequiredPermissionCode("db:transfer:run"),

		// 停止正在执行中的任务
		req.NewPost(":taskId/stop", d.Stop).Log(req.NewLogSave("dts-终止数据迁移任务")).RequiredPermissionCode("db:transfer:run"),

		// 导出文件管理-列表
		req.NewGet("/files/:taskId", d.Files),

		// 导出文件管理-删除
		req.NewPost("/files/del/:fileId", d.FileDel).Log(req.NewLogSave("dts-删除迁移文件")).RequiredPermissionCode("db:transfer:files:del"),

		req.NewPost("/files/run", d.FileRun).Log(req.NewLogSave("dts-执行sql文件")).RequiredPermissionCode("db:transfer:files:run"),
	}

	req.BatchSetGroup(instances, reqs[:])
}

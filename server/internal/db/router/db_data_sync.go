package router

import (
	"mayfly-go/internal/db/api"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitDbDataSyncRouter(router *gin.RouterGroup) {
	instances := router.Group("/datasync/tasks")

	d := new(api.DataSyncTask)
	biz.ErrIsNil(ioc.Inject(d))

	reqs := [...]*req.Conf{
		// 获取任务列表 /datasync
		req.NewGet("", d.Tasks),

		req.NewGet(":taskId/logs", d.Logs).RequiredPermissionCode("db:sync:log"),

		// 保存任务 /datasync/save
		req.NewPost("save", d.SaveTask).Log(req.NewLogSave("datasync-保存数据同步任务信息")).RequiredPermissionCode("db:sync:save"),

		// 获取单个详情 /datasync/:taskId
		req.NewGet(":taskId", d.GetTask),

		// 删除任务 /datasync/:taskId/del
		req.NewDelete(":taskId/del", d.DeleteTask).Log(req.NewLogSave("datasync-删除数据同步任务信息")).RequiredPermissionCode("db:sync:del"),

		// 启停用任务 /datasync/status
		req.NewPost(":taskId/status", d.ChangeStatus).Log(req.NewLogSave("datasync-启停任务")).RequiredPermissionCode("db:sync:status"),

		// 立即执行任务 /datasync/run
		req.NewPost(":taskId/run", d.Run),

		// 停止正在执行中的任务
		req.NewPost(":taskId/stop", d.Stop),
	}

	req.BatchSetGroup(instances, reqs[:])
}

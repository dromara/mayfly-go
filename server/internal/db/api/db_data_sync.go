package api

import (
	"encoding/base64"
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/stringx"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type DataSyncTask struct {
	DataSyncTaskApp application.DataSyncTask `inject:"DbDataSyncTaskApp"`
}

func (d *DataSyncTask) Tasks(rc *req.Ctx) {
	queryCond, page := ginx.BindQueryAndPage[*entity.DataSyncTaskQuery](rc.GinCtx, new(entity.DataSyncTaskQuery))
	res, err := d.DataSyncTaskApp.GetPageList(queryCond, page, new([]vo.DataSyncTaskListVO))
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (d *DataSyncTask) Logs(rc *req.Ctx) {
	queryCond, page := ginx.BindQueryAndPage[*entity.DataSyncLogQuery](rc.GinCtx, new(entity.DataSyncLogQuery))
	res, err := d.DataSyncTaskApp.GetTaskLogList(queryCond, page, new([]vo.DataSyncLogListVO))
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (d *DataSyncTask) SaveTask(rc *req.Ctx) {
	form := &form.DataSyncTaskForm{}
	task := ginx.BindJsonAndCopyTo[*entity.DataSyncTask](rc.GinCtx, form, new(entity.DataSyncTask))

	// 解码base64 sql
	sqlBytes, err := base64.StdEncoding.DecodeString(task.DataSql)
	biz.ErrIsNilAppendErr(err, "sql解码失败: %s")
	sql := stringx.TrimSpaceAndBr(string(sqlBytes))
	task.DataSql = sql
	form.DataSql = sql

	rc.ReqParam = form
	biz.ErrIsNil(d.DataSyncTaskApp.Save(rc.MetaCtx, task))
}

func (d *DataSyncTask) DeleteTask(rc *req.Ctx) {
	taskId := ginx.PathParam(rc.GinCtx, "taskId")
	rc.ReqParam = taskId
	ids := strings.Split(taskId, ",")

	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		biz.ErrIsNil(d.DataSyncTaskApp.Delete(rc.MetaCtx, uint64(value)))
	}
}

func (d *DataSyncTask) ChangeStatus(rc *req.Ctx) {
	form := &form.DataSyncTaskStatusForm{}
	task := ginx.BindJsonAndCopyTo[*entity.DataSyncTask](rc.GinCtx, form, new(entity.DataSyncTask))
	_ = d.DataSyncTaskApp.UpdateById(rc.MetaCtx, task)

	if task.Status == entity.DataSyncTaskStatusEnable {
		task, err := d.DataSyncTaskApp.GetById(new(entity.DataSyncTask), task.Id)
		biz.ErrIsNil(err, "该任务不存在")
		d.DataSyncTaskApp.AddCronJob(task)
	} else {
		d.DataSyncTaskApp.RemoveCronJobById(task.Id)
	}
	// 记录请求日志
	rc.ReqParam = form
}

func (d *DataSyncTask) Run(rc *req.Ctx) {
	taskId := getTaskId(rc.GinCtx)
	rc.ReqParam = taskId
	_ = d.DataSyncTaskApp.RunCronJob(taskId)
}

func (d *DataSyncTask) Stop(rc *req.Ctx) {
	taskId := getTaskId(rc.GinCtx)
	rc.ReqParam = taskId

	task := new(entity.DataSyncTask)
	task.Id = taskId
	task.RunningState = entity.DataSyncTaskRunStateStop
	_ = d.DataSyncTaskApp.UpdateById(rc.MetaCtx, task)
}

func (d *DataSyncTask) GetTask(rc *req.Ctx) {
	taskId := getTaskId(rc.GinCtx)
	dbEntity, _ := d.DataSyncTaskApp.GetById(new(entity.DataSyncTask), taskId)
	rc.ResData = dbEntity
}

func getTaskId(g *gin.Context) uint64 {
	instanceId, _ := strconv.Atoi(g.Param("taskId"))
	biz.IsTrue(instanceId > 0, "instanceId 错误")
	return uint64(instanceId)
}

package api

import (
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/cryptox"
	"mayfly-go/pkg/utils/stringx"
	"strings"

	"github.com/may-fly/cast"
)

type DataSyncTask struct {
	DataSyncTaskApp application.DataSyncTask `inject:"DbDataSyncTaskApp"`
}

func (d *DataSyncTask) Tasks(rc *req.Ctx) {
	queryCond, page := req.BindQueryAndPage[*entity.DataSyncTaskQuery](rc, new(entity.DataSyncTaskQuery))
	res, err := d.DataSyncTaskApp.GetPageList(queryCond, page, new([]vo.DataSyncTaskListVO))
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (d *DataSyncTask) Logs(rc *req.Ctx) {
	queryCond, page := req.BindQueryAndPage[*entity.DataSyncLogQuery](rc, new(entity.DataSyncLogQuery))
	res, err := d.DataSyncTaskApp.GetTaskLogList(queryCond, page, new([]vo.DataSyncLogListVO))
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (d *DataSyncTask) SaveTask(rc *req.Ctx) {
	form := &form.DataSyncTaskForm{}
	task := req.BindJsonAndCopyTo[*entity.DataSyncTask](rc, form, new(entity.DataSyncTask))

	// 解码base64 sql
	sqlStr, err := cryptox.AesDecryptByLa(task.DataSql, rc.GetLoginAccount())
	biz.ErrIsNilAppendErr(err, "sql decoding failure: %s")
	sql := stringx.TrimSpaceAndBr(sqlStr)
	task.DataSql = sql
	form.DataSql = sql

	rc.ReqParam = form
	biz.ErrIsNil(d.DataSyncTaskApp.Save(rc.MetaCtx, task))
}

func (d *DataSyncTask) DeleteTask(rc *req.Ctx) {
	taskId := rc.PathParam("taskId")
	rc.ReqParam = taskId
	ids := strings.Split(taskId, ",")

	for _, v := range ids {
		biz.ErrIsNil(d.DataSyncTaskApp.Delete(rc.MetaCtx, cast.ToUint64(v)))
	}
}

func (d *DataSyncTask) ChangeStatus(rc *req.Ctx) {
	form := &form.DataSyncTaskStatusForm{}
	task := req.BindJsonAndCopyTo[*entity.DataSyncTask](rc, form, new(entity.DataSyncTask))
	_ = d.DataSyncTaskApp.UpdateById(rc.MetaCtx, task)

	if task.Status == entity.DataSyncTaskStatusEnable {
		task, err := d.DataSyncTaskApp.GetById(task.Id)
		biz.ErrIsNil(err, "task not found")
		d.DataSyncTaskApp.AddCronJob(rc.MetaCtx, task)
	} else {
		d.DataSyncTaskApp.RemoveCronJobById(task.Id)
	}
	// 记录请求日志
	rc.ReqParam = form
}

func (d *DataSyncTask) Run(rc *req.Ctx) {
	taskId := d.getTaskId(rc)
	rc.ReqParam = taskId
	_ = d.DataSyncTaskApp.RunCronJob(rc.MetaCtx, taskId)
}

func (d *DataSyncTask) Stop(rc *req.Ctx) {
	taskId := d.getTaskId(rc)
	rc.ReqParam = taskId

	task := new(entity.DataSyncTask)
	task.Id = taskId
	task.RunningState = entity.DataSyncTaskRunStateStop
	_ = d.DataSyncTaskApp.UpdateById(rc.MetaCtx, task)
}

func (d *DataSyncTask) GetTask(rc *req.Ctx) {
	taskId := d.getTaskId(rc)
	dbEntity, _ := d.DataSyncTaskApp.GetById(taskId)
	rc.ResData = dbEntity
}

func (d *DataSyncTask) getTaskId(rc *req.Ctx) uint64 {
	instanceId := rc.PathParamInt("taskId")
	biz.IsTrue(instanceId > 0, "instanceId error")
	return uint64(instanceId)
}

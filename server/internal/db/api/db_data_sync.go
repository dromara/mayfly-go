package api

import (
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/internal/pkg/utils"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/stringx"
	"strings"

	"github.com/may-fly/cast"
)

type DataSyncTask struct {
	dataSyncTaskApp application.DataSyncTask `inject:"T"`
}

func (d *DataSyncTask) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 获取任务列表 /datasync
		req.NewGet("", d.Tasks),

		req.NewGet(":taskId/logs", d.Logs).RequiredPermissionCode("db:sync:log"),

		// 保存任务 /datasync/save
		req.NewPost("save", d.SaveTask).Log(req.NewLogSaveI(imsg.LogDataSyncSave)).RequiredPermissionCode("db:sync:save"),

		// 获取单个详情 /datasync/:taskId
		req.NewGet(":taskId", d.GetTask),

		// 删除任务 /datasync/:taskId/del
		req.NewDelete(":taskId/del", d.DeleteTask).Log(req.NewLogSaveI(imsg.LogDataSyncDelete)).RequiredPermissionCode("db:sync:del"),

		// 启停用任务 /datasync/status
		req.NewPost(":taskId/status", d.ChangeStatus).Log(req.NewLogSaveI(imsg.LogDataSyncChangeStatus)).RequiredPermissionCode("db:sync:status"),

		// 立即执行任务 /datasync/run
		req.NewPost(":taskId/run", d.Run),

		// 停止正在执行中的任务
		req.NewPost(":taskId/stop", d.Stop),
	}

	return req.NewConfs("/datasync/tasks", reqs[:]...)
}

func (d *DataSyncTask) Tasks(rc *req.Ctx) {
	queryCond := req.BindQuery[*entity.DataSyncTaskQuery](rc)
	res, err := d.dataSyncTaskApp.GetPageList(queryCond)
	biz.ErrIsNil(err)
	rc.ResData = model.PageResultConv[*entity.DataSyncTask, *vo.DataSyncTaskListVO](res)
}

func (d *DataSyncTask) Logs(rc *req.Ctx) {
	queryCond := req.BindQuery[*entity.DataSyncLogQuery](rc)
	res, err := d.dataSyncTaskApp.GetTaskLogList(queryCond)
	biz.ErrIsNil(err)
	rc.ResData = model.PageResultConv[*entity.DataSyncLog, *vo.DataSyncLogListVO](res)
}

func (d *DataSyncTask) SaveTask(rc *req.Ctx) {
	form, task := req.BindJsonAndCopyTo[*form.DataSyncTaskForm, *entity.DataSyncTask](rc)

	// 解码base64 sql
	sqlStr, err := utils.AesDecryptByLa(task.DataSql, rc.GetLoginAccount())
	biz.ErrIsNilAppendErr(err, "sql decoding failure: %s")
	sql := stringx.TrimSpaceAndBr(sqlStr)
	task.DataSql = sql
	form.DataSql = sql

	rc.ReqParam = form
	biz.ErrIsNil(d.dataSyncTaskApp.Save(rc.MetaCtx, task))
}

func (d *DataSyncTask) DeleteTask(rc *req.Ctx) {
	taskId := rc.PathParam("taskId")
	rc.ReqParam = taskId
	ids := strings.Split(taskId, ",")

	for _, v := range ids {
		biz.ErrIsNil(d.dataSyncTaskApp.Delete(rc.MetaCtx, cast.ToUint64(v)))
	}
}

func (d *DataSyncTask) ChangeStatus(rc *req.Ctx) {
	form, task := req.BindJsonAndCopyTo[*form.DataSyncTaskStatusForm, *entity.DataSyncTask](rc)
	_ = d.dataSyncTaskApp.UpdateById(rc.MetaCtx, task)

	if task.Status == entity.DataSyncTaskStatusEnable {
		task, err := d.dataSyncTaskApp.GetById(task.Id)
		biz.ErrIsNil(err, "task not found")
		d.dataSyncTaskApp.AddCronJob(rc.MetaCtx, task)
	} else {
		d.dataSyncTaskApp.RemoveCronJobById(task.Id)
	}
	// 记录请求日志
	rc.ReqParam = form
}

func (d *DataSyncTask) Run(rc *req.Ctx) {
	taskId := d.getTaskId(rc)
	rc.ReqParam = taskId
	_ = d.dataSyncTaskApp.RunCronJob(rc.MetaCtx, taskId)
}

func (d *DataSyncTask) Stop(rc *req.Ctx) {
	taskId := d.getTaskId(rc)
	rc.ReqParam = taskId
	biz.ErrIsNil(d.dataSyncTaskApp.StopTask(rc.MetaCtx, taskId))
}

func (d *DataSyncTask) GetTask(rc *req.Ctx) {
	taskId := d.getTaskId(rc)
	dbEntity, _ := d.dataSyncTaskApp.GetById(taskId)
	rc.ResData = dbEntity
}

func (d *DataSyncTask) getTaskId(rc *req.Ctx) uint64 {
	instanceId := rc.PathParamInt("taskId")
	biz.IsTrue(instanceId > 0, "instanceId error")
	return uint64(instanceId)
}

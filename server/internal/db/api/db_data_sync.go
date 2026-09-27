package api

import (
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/application/sync"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/internal/pkg/utils"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"strings"

	"github.com/spf13/cast"
)

type DataSyncTask struct {
	dataSyncTaskApp sync.DataSyncTask `inject:"T"`

	// 列表需要按库id回查数据库类型（同步任务表不像迁移任务表那样冗余存类型），故依赖库应用
	dbApp application.Db `inject:"T"`
}

func (d *DataSyncTask) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 获取任务列表 /datasync
		req.NewGet("", d.Tasks),

		req.NewGet(":taskId/logs", d.Logs).RequiredPermissionCode("db:sync:log"),

		// 单条执行日志的运行日志内容（列表接口不返回大文本，按需获取）
		req.NewGet("logs/:logId/run", d.LogRun).RequiredPermissionCode("db:sync:log"),

		// 保存任务 /datasync/save
		req.NewPost("save", d.SaveTask).Log(req.NewLogSaveI(imsg.LogDataSyncSave)).RequiredPermissionCode("db:sync:save"),

		// 获取单个详情 /datasync/:taskId
		req.NewGet(":taskId", d.GetTask),

		// 删除任务 /datasync/:taskId/del
		req.NewDelete(":taskId/del", d.DeleteTask).Log(req.NewLogSaveI(imsg.LogDataSyncDelete)).RequiredPermissionCode("db:sync:del"),

		// 启停用任务 /datasync/status
		req.NewPost(":taskId/status", d.ChangeStatus).Log(req.NewLogSaveI(imsg.LogDataSyncChangeStatus)).RequiredPermissionCode("db:sync:status"),

		// 立即执行任务 /datasync/run
		req.NewPost(":taskId/run", d.Run).RequiredPermissionCode("db:sync:run"),

		// 停止正在执行中的任务
		req.NewPost(":taskId/stop", d.Stop).RequiredPermissionCode("db:sync:stop"),
	}

	return req.NewConfs("/datasync/tasks", reqs[:]...)
}

func (d *DataSyncTask) Tasks(rc *req.Ctx) {
	queryCond := rc.BindQuery[entity.DataSyncTaskQuery]()
	res, err := d.dataSyncTaskApp.GetPageList(queryCond)
	biz.ErrIsNil(err)
	resVo := model.PageResultConv[*entity.DataSyncTask, *vo.DataSyncTaskListVO](res)

	// 方言图标需要库类型：本页内去重后两次批量主键查询取回，不逐行查也不开库连接
	dbIds := make([]uint64, 0, len(resVo.List)*2)
	for _, item := range resVo.List {
		dbIds = append(dbIds, uint64(item.SrcDbId), uint64(item.TargetDbId))
	}
	dbTypes, err := d.dbApp.GetDbTypesByDbIds(collx.ArrayDeduplicate(dbIds))
	biz.ErrIsNilAppendErr(err, "failed to obtain the database types: %s")
	for _, item := range resVo.List {
		item.SrcDbType = dbTypes[uint64(item.SrcDbId)]
		item.TargetDbType = dbTypes[uint64(item.TargetDbId)]
	}

	rc.ResData = resVo
}

// Logs 同步任务执行日志列表：不返回运行日志内容，避免日志较多时单次响应体过大，运行日志由 LogRun 按日志 id 单条获取
func (d *DataSyncTask) Logs(rc *req.Ctx) {
	// 任务 id 以路径参数为准，覆盖 query 中可能传入的同名参数
	queryCond := rc.BindQuery[entity.DataSyncLogQuery]()
	queryCond.TaskId = cast.ToUint64(rc.PathParam("taskId"))

	res, err := d.dataSyncTaskApp.GetLogPageList(queryCond)
	biz.ErrIsNil(err)
	rc.ResData = model.PageResultConv[*entity.DataSyncLog, *vo.DataSyncLogListVO](res)
}

// LogRun 获取单条执行日志的运行日志内容（执行中的日志取实时缓存，保证实时视图看到最新内容）
func (d *DataSyncTask) LogRun(rc *req.Ctx) {
	log, err := d.dataSyncTaskApp.GetLogWithRunLog(cast.ToUint64(rc.PathParam("logId")))
	biz.ErrIsNil(err)
	rc.ResData = &vo.DataSyncLogRunVO{Id: log.Id, Status: log.Status, RunLog: log.RunLog}
}

func (d *DataSyncTask) SaveTask(rc *req.Ctx) {
	form, task := rc.BindJsonAndCopyTo[form.DataSyncTaskForm, entity.DataSyncTask]()

	// 解码base64 sql
	sqlStr, err := utils.AesDecryptByLa(task.DataSQL, rc.GetLoginAccount())
	biz.ErrIsNilAppendErr(err, "sql decoding failure: %s")
	sql := stringx.TrimSpaceAndBr(sqlStr)
	task.DataSQL = sql
	form.DataSQL = sql

	// form 上的 cursor 边界与批间 sleep 存于 entity Extra（非查询维度，不占列），
	// BindJsonAndCopyTo 无同名目标字段可反射，需显式桥接；setter 会在 0/Auto 时清旧 key。
	task.SetCursorInclusivity(form.CursorInclusivity)
	task.SetSleepBetweenBatchesMs(form.SleepBetweenBatchesMs)
	// P1 逃生阀：写入同名 Extra key，Save 里的 validateIncrementalFieldIndex 读到则跳过探测
	task.SetSkipIndexValidation(form.SkipIndexValidation)

	rc.ReqParam = form
	biz.ErrIsNil(d.dataSyncTaskApp.Save(rc.MetaCtx, task))
}

func (d *DataSyncTask) DeleteTask(rc *req.Ctx) {
	taskId := rc.PathParam("taskId")
	rc.ReqParam = taskId

	for _, v := range strings.Split(taskId, ",") {
		biz.ErrIsNil(d.dataSyncTaskApp.Delete(rc.MetaCtx, cast.ToUint64(v)))
	}
}

func (d *DataSyncTask) ChangeStatus(rc *req.Ctx) {
	form := rc.BindJson[form.DataSyncTaskStatusForm]()
	rc.ReqParam = form

	task, err := d.dataSyncTaskApp.GetById(form.Id)
	biz.ErrIsNil(err)
	task.Status = entity.DataSyncTaskStatus(form.Status)
	biz.ErrIsNil(d.dataSyncTaskApp.Save(rc.MetaCtx, task))
}

func (d *DataSyncTask) Run(rc *req.Ctx) {
	taskId := d.getTaskId(rc)
	rc.ReqParam = taskId
	biz.ErrIsNil(d.dataSyncTaskApp.Run(rc.MetaCtx, taskId))
}

func (d *DataSyncTask) Stop(rc *req.Ctx) {
	taskId := d.getTaskId(rc)
	rc.ReqParam = taskId
	biz.ErrIsNil(d.dataSyncTaskApp.StopTask(rc.MetaCtx, taskId))
}

func (d *DataSyncTask) GetTask(rc *req.Ctx) {
	taskId := d.getTaskId(rc)
	dbEntity, err := d.dataSyncTaskApp.GetById(taskId)
	biz.ErrIsNil(err)
	rc.ResData = dbEntity
}

func (d *DataSyncTask) getTaskId(rc *req.Ctx) uint64 {
	taskId := cast.ToUint64(rc.PathParam("taskId"))
	biz.IsTrue(taskId > 0, "taskId error")
	return taskId
}

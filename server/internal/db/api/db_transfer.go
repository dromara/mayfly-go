package api

import (
	"context"
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/imsg"
	fileapp "mayfly-go/internal/file/application"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"strings"

	"github.com/may-fly/cast"
)

type DbTransferTask struct {
	dbTransferTask application.DbTransferTask `inject:"T"`
	dbTransferFile application.DbTransferFile `inject:"T"`
	dbApp          application.Db             `inject:"T"`
	tagApp         tagapp.TagTree             `inject:"T"`
	dbSqlExecApp   application.DbSqlExec      `inject:"T"`
	fileApp        fileapp.File               `inject:"T"`
}

func (d *DbTransferTask) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 获取任务列表
		req.NewGet("", d.Tasks),

		// 保存任务
		req.NewPost("save", d.SaveTask).Log(req.NewLogSaveI(imsg.LogDtsSave)).RequiredPermissionCode("db:transfer:save"),

		// 删除任务
		req.NewDelete(":taskId/del", d.DeleteTask).Log(req.NewLogSaveI(imsg.LogDtsDelete)).RequiredPermissionCode("db:transfer:del"),

		// 启停用任务
		req.NewPost(":taskId/status", d.ChangeStatus).Log(req.NewLogSaveI(imsg.LogDtsChangeStatus)).RequiredPermissionCode("db:transfer:status"),

		// 立即执行任务
		req.NewPost(":taskId/run", d.Run).Log(req.NewLogI(imsg.LogDtsRun)).RequiredPermissionCode("db:transfer:run"),

		// 停止正在执行中的任务
		req.NewPost(":taskId/stop", d.Stop).Log(req.NewLogSaveI(imsg.LogDtsStop)).RequiredPermissionCode("db:transfer:run"),

		// 导出文件管理-列表
		req.NewGet("/files/:taskId", d.Files),

		// 导出文件管理-删除
		req.NewPost("/files/del/:fileId", d.FileDel).Log(req.NewLogSaveI(imsg.LogDtsDeleteFile)).RequiredPermissionCode("db:transfer:files:del"),

		req.NewPost("/files/run", d.FileRun).Log(req.NewLogSaveI(imsg.LogDtsRunSqlFile)).RequiredPermissionCode("db:transfer:files:run"),
	}

	return req.NewConfs("/dbTransfer", reqs[:]...)
}

func (d *DbTransferTask) Tasks(rc *req.Ctx) {
	queryCond := req.BindQuery[*entity.DbTransferTaskQuery](rc)

	res, err := d.dbTransferTask.GetPageList(queryCond)
	biz.ErrIsNil(err)
	resVo := model.PageResultConv[*entity.DbTransferTask, *vo.DbTransferTaskListVO](res)

	for _, item := range resVo.List {
		item.RunningState = entity.DbTransferTaskRunStateSuccess
		if d.dbTransferTask.IsRunning(item.Id) {
			item.RunningState = entity.DbTransferTaskRunStateRunning
		}
	}

	rc.ResData = resVo
}

func (d *DbTransferTask) SaveTask(rc *req.Ctx) {
	reqForm, task := req.BindJsonAndCopyTo[*form.DbTransferTaskForm, *entity.DbTransferTask](rc)

	rc.ReqParam = reqForm
	biz.ErrIsNil(d.dbTransferTask.Save(rc.MetaCtx, task))
}

func (d *DbTransferTask) DeleteTask(rc *req.Ctx) {
	taskId := rc.PathParam("taskId")
	rc.ReqParam = taskId
	ids := strings.Split(taskId, ",")

	uids := collx.ArrayMap[string, uint64](ids, func(val string) uint64 {
		return cast.ToUint64(val)
	})

	biz.ErrIsNil(d.dbTransferTask.DeleteById(rc.MetaCtx, uids...))
}

func (d *DbTransferTask) ChangeStatus(rc *req.Ctx) {
	form, task := req.BindJsonAndCopyTo[*form.DbTransferTaskStatusForm, *entity.DbTransferTask](rc)
	_ = d.dbTransferTask.UpdateById(rc.MetaCtx, task)

	task, err := d.dbTransferTask.GetById(task.Id)
	biz.ErrIsNil(err, "task not found")
	d.dbTransferTask.AddCronJob(rc.MetaCtx, task)

	// 记录请求日志
	rc.ReqParam = form
}

func (d *DbTransferTask) Run(rc *req.Ctx) {
	taskId := uint64(rc.PathParamInt("taskId"))
	logId, _ := d.dbTransferTask.CreateLog(rc.MetaCtx, taskId)
	go d.dbTransferTask.Run(rc.MetaCtx, taskId, logId)
	rc.ResData = logId
}

func (d *DbTransferTask) Stop(rc *req.Ctx) {
	biz.ErrIsNil(d.dbTransferTask.Stop(rc.MetaCtx, uint64(rc.PathParamInt("taskId"))))
}

func (d *DbTransferTask) Files(rc *req.Ctx) {
	queryCond := req.BindQuery[*entity.DbTransferFileQuery](rc)

	res, err := d.dbTransferFile.GetPageList(queryCond)
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (d *DbTransferTask) FileDel(rc *req.Ctx) {
	fileId := rc.PathParam("fileId")
	rc.ReqParam = fileId // 记录操作日志
	ids := strings.Split(fileId, ",")

	uIds := make([]uint64, len(ids))
	for _, v := range ids {
		uIds = append(uIds, cast.ToUint64(v))
	}
	biz.ErrIsNil(d.dbTransferFile.Delete(rc.MetaCtx, uIds...))
}

func (d *DbTransferTask) FileRun(rc *req.Ctx) {
	fm := req.BindJsonAndValid[*form.DbTransferFileRunForm](rc)

	rc.ReqParam = fm

	tFile, err := d.dbTransferFile.GetById(fm.Id)
	biz.IsTrue(tFile != nil && err == nil, "file not found")

	targetDbConn, err := d.dbApp.GetDbConn(rc.MetaCtx, fm.TargetDbId, fm.TargetDbName)
	biz.ErrIsNilAppendErr(err, "failed to connect to the target database: %s")

	biz.ErrIsNilAppendErr(d.tagApp.CanAccess(rc.GetLoginAccount().Id, targetDbConn.Info.CodePath...), "%s")

	filename, reader, err := d.fileApp.GetReader(context.TODO(), tFile.FileKey)
	biz.ErrIsNil(err)
	go func() {
		biz.ErrIsNil(d.dbSqlExecApp.ExecReader(rc.MetaCtx, &dto.SqlReaderExec{
			Reader:   reader,
			Filename: filename,
			DbConn:   targetDbConn,
			ClientId: fm.ClientId,
		}))
	}()
}

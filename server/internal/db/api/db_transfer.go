package api

import (
	"context"
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/application/transfer"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/imsg"
	fileapp "mayfly-go/internal/file/application"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"strings"

	"github.com/spf13/cast"
)

type DbTransferTask struct {
	dbTransferTaskApp transfer.DbTransferTask `inject:"T"`
	dbTransferFileApp transfer.DbTransferFile `inject:"T"`
	dbApp             application.Db          `inject:"T"`
	tagApp            tagapp.TagTreeChecker   `inject:"T"`
	dbSqlExecApp      application.DbSqlExec   `inject:"T"`
	fileApp           fileapp.File            `inject:"T"`
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

		// 数据校验：校验任务源库与目标库数据一致性，异步执行并返回日志id
		req.NewPost(":taskId/verify", d.Verify).Log(req.NewLogI(imsg.LogDtsVerify)).RequiredPermissionCode("db:transfer:run"),

		// 导出文件管理-列表
		req.NewGet("/files/:taskId", d.Files),

		// 导出文件管理-删除
		req.NewPost("/files/del/:fileId", d.FileDel).Log(req.NewLogSaveI(imsg.LogDtsDeleteFile)).RequiredPermissionCode("db:transfer:files:del"),

		req.NewPost("/files/run", d.FileRun).Log(req.NewLogSaveI(imsg.LogDtsRunSqlFile)).RequiredPermissionCode("db:transfer:files:run"),
	}

	return req.NewConfs("/dbTransfer", reqs[:]...)
}

func (d *DbTransferTask) Tasks(rc *req.Ctx) {
	queryCond := rc.BindQuery[entity.DbTransferTaskQuery]()

	res, err := d.dbTransferTaskApp.GetPageList(queryCond)
	biz.ErrIsNil(err)
	resVo := model.PageResultConv[*entity.DbTransferTask, *vo.DbTransferTaskListVO](res)

	for _, item := range resVo.List {
		item.RunningState = entity.DbTransferTaskRunStateSuccess
		if d.dbTransferTaskApp.IsRunning(item.Id) {
			item.RunningState = entity.DbTransferTaskRunStateRunning
		}
	}

	rc.ResData = resVo
}

func (d *DbTransferTask) SaveTask(rc *req.Ctx) {
	reqForm, task := rc.BindJsonAndCopyTo[form.DbTransferTaskForm, entity.DbTransferTask]()

	rc.ReqParam = reqForm
	biz.ErrIsNil(d.dbTransferTaskApp.Save(rc.MetaCtx, task))
}

func (d *DbTransferTask) DeleteTask(rc *req.Ctx) {
	taskId := rc.PathParam("taskId")
	rc.ReqParam = taskId

	for _, id := range strings.Split(taskId, ",") {
		biz.ErrIsNil(d.dbTransferTaskApp.Delete(rc.MetaCtx, cast.ToUint64(id)))
	}
}

func (d *DbTransferTask) ChangeStatus(rc *req.Ctx) {
	form := rc.BindJson[form.DbTransferTaskStatusForm]()
	rc.ReqParam = form

	task, err := d.dbTransferTaskApp.GetById(form.Id)
	biz.ErrIsNil(err)
	task.Status = form.Status
	biz.ErrIsNil(d.dbTransferTaskApp.Save(rc.MetaCtx, task))
}

func (d *DbTransferTask) Run(rc *req.Ctx) {
	taskId := uint64(rc.PathParamInt("taskId"))
	rc.ReqParam = taskId

	logId, err := d.dbTransferTaskApp.Run(rc.MetaCtx, taskId)
	biz.ErrIsNil(err)
	rc.ResData = logId
}

func (d *DbTransferTask) Stop(rc *req.Ctx) {
	biz.ErrIsNil(d.dbTransferTaskApp.Stop(rc.MetaCtx, uint64(rc.PathParamInt("taskId"))))
}

// Verify 数据校验：异步执行，返回日志id（日志Resp中可查询校验报告）
func (d *DbTransferTask) Verify(rc *req.Ctx) {
	taskId := uint64(rc.PathParamInt("taskId"))
	rc.ReqParam = taskId

	logId, err := d.dbTransferTaskApp.Verify(rc.MetaCtx, taskId)
	biz.ErrIsNil(err)
	rc.ResData = logId
}

func (d *DbTransferTask) Files(rc *req.Ctx) {
	queryCond := rc.BindQuery[entity.DbTransferFileQuery]()

	res, err := d.dbTransferFileApp.GetPageList(queryCond)
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (d *DbTransferTask) FileDel(rc *req.Ctx) {
	fileId := rc.PathParam("fileId")
	rc.ReqParam = fileId // 记录操作日志

	biz.ErrIsNil(d.dbTransferFileApp.Delete(rc.MetaCtx, parseFileIds(fileId)...))
}

// parseFileIds 解析逗号分隔的文件id串，过滤空串与非法值。
// 不能make(len(ids))预填零值，否则前面多出len(ids)个0会导致误删id为0的记录
func parseFileIds(fileId string) []uint64 {
	ids := strings.Split(fileId, ",")
	uIds := make([]uint64, 0, len(ids))
	for _, v := range ids {
		// cast对string不做TrimSpace，" 5"会解析失败转0，需显式清理
		if id := cast.ToUint64(strings.TrimSpace(v)); id > 0 {
			uIds = append(uIds, id)
		}
	}
	return uIds
}

func (d *DbTransferTask) FileRun(rc *req.Ctx) {
	fm := rc.BindJson[form.DbTransferFileRunForm]()

	rc.ReqParam = fm

	tFile, err := d.dbTransferFileApp.GetById(fm.Id)
	biz.IsTrue(tFile != nil && err == nil, "file not found")

	targetDbConn, err := d.dbApp.GetDbConn(rc.MetaCtx, fm.TargetDbId, fm.TargetDbName)
	biz.ErrIsNilAppendErr(err, "failed to connect to the target database: %s")

	biz.ErrIsNilAppendErr(d.tagApp.CanAccess(rc.GetLoginAccount().Id, targetDbConn.Info.CodePath...), "%s")

	filename, reader, err := d.fileApp.GetReader(context.TODO(), tFile.FileKey)
	biz.ErrIsNil(err)

	gox.GoCtx(rc.MetaCtx, func(ctx context.Context) {
		biz.ErrIsNil(d.dbSqlExecApp.ExecReader(ctx, &dto.SqlReaderExec{
			Reader:   reader,
			Filename: filename,
			DbConn:   targetDbConn,
			ClientId: fm.ClientId,
		}))
	})
}

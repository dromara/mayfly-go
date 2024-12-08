package api

import (
	"context"
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/domain/entity"
	fileapp "mayfly-go/internal/file/application"
	msgapp "mayfly-go/internal/msg/application"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"strings"

	"github.com/may-fly/cast"
)

type DbTransferTask struct {
	DbTransferTask application.DbTransferTask `inject:"DbTransferTaskApp"`
	DbTransferFile application.DbTransferFile `inject:"DbTransferFileApp"`
	DbApp          application.Db             `inject:""`
	TagApp         tagapp.TagTree             `inject:"TagTreeApp"`
	MsgApp         msgapp.Msg                 `inject:""`
	DbSqlExecApp   application.DbSqlExec      `inject:""`
	FileApp        fileapp.File               `inject:""`
}

func (d *DbTransferTask) Tasks(rc *req.Ctx) {
	queryCond, page := req.BindQueryAndPage[*entity.DbTransferTaskQuery](rc, new(entity.DbTransferTaskQuery))
	res, err := d.DbTransferTask.GetPageList(queryCond, page, new([]vo.DbTransferTaskListVO))
	biz.ErrIsNil(err)

	if res.List != nil {
		list := res.List.(*[]vo.DbTransferTaskListVO)
		for _, item := range *list {
			item.RunningState = entity.DbTransferTaskRunStateSuccess
			if d.DbTransferTask.IsRunning(item.Id) {
				item.RunningState = entity.DbTransferTaskRunStateRunning
			}
		}
	}

	rc.ResData = res
}

func (d *DbTransferTask) SaveTask(rc *req.Ctx) {
	reqForm := &form.DbTransferTaskForm{}
	task := req.BindJsonAndCopyTo[*entity.DbTransferTask](rc, reqForm, new(entity.DbTransferTask))

	rc.ReqParam = reqForm
	biz.ErrIsNil(d.DbTransferTask.Save(rc.MetaCtx, task))
}

func (d *DbTransferTask) DeleteTask(rc *req.Ctx) {
	taskId := rc.PathParam("taskId")
	rc.ReqParam = taskId
	ids := strings.Split(taskId, ",")

	uids := collx.ArrayMap[string, uint64](ids, func(val string) uint64 {
		return cast.ToUint64(val)
	})

	biz.ErrIsNil(d.DbTransferTask.DeleteById(rc.MetaCtx, uids...))
}

func (d *DbTransferTask) ChangeStatus(rc *req.Ctx) {
	form := &form.DbTransferTaskStatusForm{}
	task := req.BindJsonAndCopyTo[*entity.DbTransferTask](rc, form, new(entity.DbTransferTask))
	_ = d.DbTransferTask.UpdateById(rc.MetaCtx, task)

	task, err := d.DbTransferTask.GetById(task.Id)
	biz.ErrIsNil(err, "task not found")
	d.DbTransferTask.AddCronJob(rc.MetaCtx, task)

	// 记录请求日志
	rc.ReqParam = form
}

func (d *DbTransferTask) Run(rc *req.Ctx) {
	taskId := uint64(rc.PathParamInt("taskId"))
	logId, _ := d.DbTransferTask.CreateLog(rc.MetaCtx, taskId)
	go d.DbTransferTask.Run(rc.MetaCtx, taskId, logId)
	rc.ResData = logId
}

func (d *DbTransferTask) Stop(rc *req.Ctx) {
	biz.ErrIsNil(d.DbTransferTask.Stop(rc.MetaCtx, uint64(rc.PathParamInt("taskId"))))
}

func (d *DbTransferTask) Files(rc *req.Ctx) {
	queryCond, page := req.BindQueryAndPage[*entity.DbTransferFileQuery](rc, new(entity.DbTransferFileQuery))
	res, err := d.DbTransferFile.GetPageList(queryCond, page, new([]vo.DbTransferFileListVO))
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
	biz.ErrIsNil(d.DbTransferFile.Delete(rc.MetaCtx, uIds...))
}

func (d *DbTransferTask) FileRun(rc *req.Ctx) {
	fm := req.BindJsonAndValid(rc, &form.DbTransferFileRunForm{})

	rc.ReqParam = fm

	tFile, err := d.DbTransferFile.GetById(fm.Id)
	biz.IsTrue(tFile != nil && err == nil, "file not found")

	targetDbConn, err := d.DbApp.GetDbConn(fm.TargetDbId, fm.TargetDbName)
	biz.ErrIsNilAppendErr(err, "failed to connect to the target database: %s")
	biz.ErrIsNilAppendErr(d.TagApp.CanAccess(rc.GetLoginAccount().Id, targetDbConn.Info.CodePath...), "%s")

	filename, reader, err := d.FileApp.GetReader(context.TODO(), tFile.FileKey)
	biz.ErrIsNil(err)
	go func() {
		biz.ErrIsNil(d.DbSqlExecApp.ExecReader(rc.MetaCtx, &dto.SqlReaderExec{
			Reader:   reader,
			Filename: filename,
			DbConn:   targetDbConn,
			ClientId: fm.ClientId,
		}))
	}()

}

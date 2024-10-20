package api

import (
	"fmt"
	"io"
	"mayfly-go/internal/db/api/form"
	"mayfly-go/internal/db/api/vo"
	"mayfly-go/internal/db/application"
	"mayfly-go/internal/db/config"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/domain/entity"
	msgapp "mayfly-go/internal/msg/application"
	msgdto "mayfly-go/internal/msg/application/dto"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/ws"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type DbTransferTask struct {
	DbTransferTask application.DbTransferTask `inject:"DbTransferTaskApp"`
	DbTransferFile application.DbTransferFile `inject:"DbTransferFileApp"`
	DbApp          application.Db             `inject:""`
	TagApp         tagapp.TagTree             `inject:"TagTreeApp"`
	MsgApp         msgapp.Msg                 `inject:""`
	DbSqlExecApp   application.DbSqlExec      `inject:""`
}

func (d *DbTransferTask) Tasks(rc *req.Ctx) {
	queryCond, page := req.BindQueryAndPage[*entity.DbTransferTaskQuery](rc, new(entity.DbTransferTaskQuery))
	res, err := d.DbTransferTask.GetPageList(queryCond, page, new([]vo.DbTransferTaskListVO))
	biz.ErrIsNil(err)

	list := res.List.(*[]vo.DbTransferTaskListVO)
	for _, item := range *list {
		item.RunningState = entity.DbTransferTaskRunStateSuccess
		if d.DbTransferTask.IsRunning(item.Id) {
			item.RunningState = entity.DbTransferTaskRunStateRunning
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

	uids := make([]uint64, len(ids))
	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		uids = append(uids, uint64(value))
	}

	biz.ErrIsNil(d.DbTransferTask.DeleteById(rc.MetaCtx, uids...))
}

func (d *DbTransferTask) ChangeStatus(rc *req.Ctx) {
	form := &form.DbTransferTaskStatusForm{}
	task := req.BindJsonAndCopyTo[*entity.DbTransferTask](rc, form, new(entity.DbTransferTask))
	_ = d.DbTransferTask.UpdateById(rc.MetaCtx, task)

	task, err := d.DbTransferTask.GetById(task.Id)
	biz.ErrIsNil(err, "该任务不存在")
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

func (d *DbTransferTask) FileRename(rc *req.Ctx) {
	fm := &form.DbTransferFileForm{}
	tFile := req.BindJsonAndCopyTo[*entity.DbTransferFile](rc, fm, new(entity.DbTransferFile))
	_ = d.DbTransferFile.UpdateById(rc.MetaCtx, tFile)
	rc.ReqParam = fm
}

func (d *DbTransferTask) FileDel(rc *req.Ctx) {
	fileId := rc.PathParam("fileId")
	rc.ReqParam = fileId // 记录操作日志
	ids := strings.Split(fileId, ",")

	uIds := make([]uint64, len(ids))
	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		uIds = append(uIds, uint64(value))
	}
	biz.ErrIsNil(d.DbTransferFile.Delete(rc.MetaCtx, uIds...))
}

func (d *DbTransferTask) FileDown(rc *req.Ctx) {
	fileUuid := rc.PathParam("fileUuid")
	if fileUuid == "" {
		panic(errorx.NewBiz("文件id不能为空"))
	}

	tFile := &entity.DbTransferFile{FileUuid: fileUuid}

	err := d.DbTransferFile.GetByCond(model.NewModelCond(tFile).Dest(tFile))
	biz.ErrIsNilAppendErr(err, "查询文件出错 %s")

	// 拼接文件地址，并把文件流输出到客户端
	brc := config.GetDbBackupRestore()
	filePath := filepath.Join(fmt.Sprintf("%s/%d/%s.sql", brc.TransferPath, tFile.TaskId, fileUuid))

	file, err := os.Open(filePath)
	biz.ErrIsNilAppendErr(err, "读取文件失败：%s")

	defer file.Close()

	// Get the file information to set the correct response headers
	fileInfo, err := file.Stat()
	biz.ErrIsNilAppendErr(err, "读取文件失败：%s")

	rc.ReqParam = tFile // 记录操作日志
	// 如果文件名不以 .sql 结尾，则加上 .sql
	if !strings.HasSuffix(tFile.FileName, ".sql") {
		tFile.FileName += ".sql"
	}

	rc.Header("Content-Type", "application/octet-stream")
	rc.Header("Content-Disposition", "attachment; filename="+tFile.FileName)
	rc.Header("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
	_, err = io.Copy(rc.GetWriter(), file)

}

func (d *DbTransferTask) FileRun(rc *req.Ctx) {

	fm := req.BindJsonAndValid(rc, &form.DbTransferFileRunForm{})

	rc.ReqParam = fm

	tFile, err := d.DbTransferFile.GetById(fm.Id)
	biz.IsTrue(tFile != nil && err == nil, "文件不存在")

	targetDbConn, err := d.DbApp.GetDbConn(fm.TargetDbId, fm.TargetDbName)
	biz.ErrIsNilAppendErr(err, "连接目标数据库失败: %s")
	biz.ErrIsNilAppendErr(d.TagApp.CanAccess(rc.GetLoginAccount().Id, targetDbConn.Info.CodePath...), "%s")

	defer func() {
		if err := recover(); err != nil {
			errInfo := anyx.ToString(err)
			if len(errInfo) > 300 {
				errInfo = errInfo[:300] + "..."
			}
			d.MsgApp.CreateAndSend(rc.GetLoginAccount(), msgdto.ErrSysMsg("sql脚本执行失败", fmt.Sprintf("[%s][%s]执行失败: [%s]", tFile.FileName, targetDbConn.Info.GetLogDesc(), errInfo)).WithClientId(fm.ClientId))
		}
	}()

	go func() {
		d.fileRun(rc.GetLoginAccount(), fm, tFile, targetDbConn)
	}()

}

func (d *DbTransferTask) fileRun(la *model.LoginAccount, fm *form.DbTransferFileRunForm, tFile *entity.DbTransferFile, targetDbConn *dbi.DbConn) {

	filePath := d.DbTransferFile.GetFilePath(tFile)
	_, err := os.Stat(filePath)
	biz.ErrIsNilAppendErr(err, "sql文件不存在：%s")

	file, err := os.Open(filePath)
	biz.ErrIsNilAppendErr(err, "sql文件读取出错：%s")

	executedStatements := 0
	progressId := stringx.Rand(32)
	laId := la.Id

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	if err != nil {
		biz.ErrIsNilAppendErr(err, "连接目标数据库失败: %s")
	}

	err = sqlparser.SQLSplit(file, func(sql string) error {
		select {
		case <-ticker.C:
			ws.SendJsonMsg(ws.UserId(laId), fm.ClientId, msgdto.InfoSqlProgressMsg("sql脚本执行进度", &progressMsg{
				Id:                 progressId,
				Title:              tFile.FileName,
				ExecutedStatements: executedStatements,
				Terminated:         false,
			}).WithCategory(progressCategory))
		default:
		}
		executedStatements++
		_, err = targetDbConn.Exec(sql)
		return err
	})

	if err != nil {
		biz.ErrIsNilAppendErr(err, "执行sql失败: %s")
	}

	d.MsgApp.CreateAndSend(la, msgdto.SuccessSysMsg("sql脚本执行成功", fmt.Sprintf("sql脚本执行完成：%s", tFile.FileName)).WithClientId(fm.ClientId))
}

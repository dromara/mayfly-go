package application

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	fileapp "mayfly-go/internal/file/application"
	sysapp "mayfly-go/internal/sys/application"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/scheduler"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/timex"
	"strings"
	"time"

	"github.com/google/uuid"

	"golang.org/x/sync/errgroup"
)

type DbTransferTask interface {
	base.App[*entity.DbTransferTask]

	// GetPageList 分页获取数据库实例
	GetPageList(condition *entity.DbTransferTaskQuery, orderBy ...string) (*model.PageResult[*entity.DbTransferTask], error)

	Save(ctx context.Context, instanceEntity *entity.DbTransferTask) error

	Delete(ctx context.Context, id uint64) error

	InitCronJob()

	AddCronJob(ctx context.Context, taskEntity *entity.DbTransferTask)

	RemoveCronJobById(taskId uint64)

	CreateLog(ctx context.Context, taskId uint64) (uint64, error)

	Run(ctx context.Context, taskId uint64, logId uint64)

	IsRunning(taskId uint64) bool

	Stop(ctx context.Context, taskId uint64) error

	// TimerDeleteTransferFile 定时删除迁移文件
	TimerDeleteTransferFile()
}

var _ (DbTransferTask) = (*dbTransferAppImpl)(nil)

type dbTransferAppImpl struct {
	base.AppImpl[*entity.DbTransferTask, repository.DbTransferTask]

	dbApp           Db             `inject:"T"`
	logApp          sysapp.Syslog  `inject:"T"`
	transferFileApp DbTransferFile `inject:"T"`
	fileApp         fileapp.File   `inject:"T"`
}

func (app *dbTransferAppImpl) GetPageList(condition *entity.DbTransferTaskQuery, orderBy ...string) (*model.PageResult[*entity.DbTransferTask], error) {
	return app.GetRepo().GetTaskList(condition, orderBy...)
}

func (app *dbTransferAppImpl) Save(ctx context.Context, taskEntity *entity.DbTransferTask) error {
	var err error
	if taskEntity.Id == 0 { // 新建时生成key
		taskEntity.TaskKey = uuid.New().String()
		err = app.Insert(ctx, taskEntity)
	} else {
		err = app.UpdateById(ctx, taskEntity)
	}
	if err != nil {
		return err
	}
	// 视情况添加或删除任务
	task, err := app.GetById(taskEntity.Id)
	if err != nil {
		return err
	}
	app.AddCronJob(ctx, task)
	return nil
}

func (app *dbTransferAppImpl) Delete(ctx context.Context, id uint64) error {
	if err := app.DeleteById(ctx, id); err != nil {
		return err
	}
	app.RemoveCronJobById(id)

	return nil
}

func (app *dbTransferAppImpl) AddCronJob(ctx context.Context, taskEntity *entity.DbTransferTask) {
	key := taskEntity.TaskKey
	// 先移除旧的任务
	scheduler.RemoveByKey(key)

	// 根据状态添加新的任务
	if taskEntity.Status == entity.DbTransferTaskStatusEnable && taskEntity.CronAble == entity.DbTransferTaskCronAbleEnable {
		if key == "" {
			taskEntity.TaskKey = uuid.New().String()
			key = taskEntity.TaskKey
			_ = app.UpdateById(ctx, taskEntity)
		}

		taskId := taskEntity.Id
		if err := scheduler.AddFunByKey(key, taskEntity.Cron, func() {
			logx.Infof("start the synchronization task: %d", taskId)
			logId, _ := app.CreateLog(ctx, taskId)
			app.Run(ctx, taskId, logId)
		}); err != nil {
			logx.ErrorTrace("add db transfer cron job failed", err)
		}
	}
}

func (app *dbTransferAppImpl) InitCronJob() {
	// 重启后，把正在运行的状态设置为停止
	_ = app.UpdateByCond(context.TODO(), &entity.DbTransferTask{RunningState: entity.DbTransferTaskRunStateStop}, &entity.DbTransferTask{RunningState: entity.DbTransferTaskRunStateRunning})
	ent := &entity.DbTransferTask{}
	list, err := app.ListByCond(model.NewModelCond(ent).Columns("id"))
	if err != nil {
		return
	}
	if len(list) > 0 {
		// 移除所有正在运行的任务
		for _, task := range list {
			app.MarkStop(task.Id)
		}
	}
	// 把所有运行中的文件状态设置为失败
	_ = app.transferFileApp.UpdateByCond(context.TODO(), &entity.DbTransferFile{Status: entity.DbTransferFileStatusFail}, &entity.DbTransferFile{Status: entity.DbTransferFileStatusRunning})

	if err := app.CursorByCond(&entity.DbTransferTaskQuery{Status: entity.DbTransferTaskStatusEnable, CronAble: entity.DbTransferTaskCronAbleEnable}, func(dtt *entity.DbTransferTask) error {
		app.AddCronJob(contextx.NewTraceId(), dtt)
		return nil
	}); err != nil {
		logx.ErrorTrace("the db data transfer task failed to initialize", err)
	}
}

func (app *dbTransferAppImpl) CreateLog(ctx context.Context, taskId uint64) (uint64, error) {
	logId, err := app.logApp.CreateLog(ctx, &sysapp.CreateLogReq{
		Description: "DBMS - Execution DB Transfer",
		ReqParam:    collx.Kvs("taskId", taskId),
		Type:        sysentity.SyslogTypeRunning,
		Resp:        "Data transfer starts...",
	})
	return logId, err
}

func (app *dbTransferAppImpl) Run(ctx context.Context, taskId uint64, logId uint64) {
	task, err := app.GetById(taskId)
	if err != nil {
		logx.Errorf("Create DBMS- Failed to perform data transfer log: %v", err)
		return
	}

	if app.IsRunning(taskId) {
		logx.Error("[%d] the task is running...", taskId)
		return
	}

	start := time.Now()
	// 修改状态与关联日志id
	task.LogId = logId
	task.RunningState = entity.DbTransferTaskRunStateRunning
	if err = app.UpdateById(ctx, task); err != nil {
		logx.Errorf("failed to update task execution status")
		return
	}

	// 标记该任务开始执行
	app.MarkRunning(taskId)

	// 获取源库连接、目标库连接，判断连接可用性，否则记录日志：xx连接不可用
	// 获取源库表信息
	srcConn, err := app.dbApp.GetDbConn(ctx, uint64(task.SrcDbId), task.SrcDbName)
	if err != nil {
		app.EndTransfer(ctx, logId, taskId, "failed to obtain source db connection", err, nil)
		return
	}

	// 获取迁移表信息
	var tables []dbi.Table
	if task.CheckedKeys == "all" {
		tables, err = srcConn.GetMetadata().GetTables()
		if err != nil {
			app.EndTransfer(ctx, logId, taskId, "failed to get source table information", err, nil)
			return
		}
	} else {
		tableNames := strings.Split(task.CheckedKeys, ",")
		tables, err = srcConn.GetMetadata().GetTables(tableNames...)
		if err != nil {
			app.EndTransfer(ctx, logId, taskId, "failed to get source table information", err, nil)
			return
		}
	}

	// 迁移到文件或数据库
	if task.Mode == entity.DbTransferTaskModeFile {
		app.transfer2File(ctx, taskId, logId, task, srcConn, start, tables)
	} else if task.Mode == entity.DbTransferTaskModeDb {
		defer app.MarkStop(taskId)
		defer app.logApp.Flush(logId, true)
		app.transfer2Db(ctx, taskId, logId, task, srcConn, start, tables)
	} else {
		app.EndTransfer(ctx, logId, taskId, "error in transfer mode, only migrating to files or databases is currently supported", err, nil)
		return
	}
}

func (app *dbTransferAppImpl) transfer2Db(ctx context.Context, taskId uint64, logId uint64, task *entity.DbTransferTask, srcConn *dbi.DbConn, start time.Time, tables []dbi.Table) {
	// 获取目标库表信息
	targetConn, err := app.dbApp.GetDbConn(ctx, uint64(task.TargetDbId), task.TargetDbName)
	if err != nil {
		app.EndTransfer(ctx, logId, taskId, "failed to get target db connection", err, nil)
		return
	}

	ctx = context.Background()

	tableNames := collx.ArrayMap(tables, func(t dbi.Table) string { return t.TableName })
	// 分组迁移
	tableGroups := collx.ArraySplit[string](tableNames, 2)
	errGroup, _ := errgroup.WithContext(ctx)

	for _, tables := range tableGroups {
		errGroup.Go(func() error {
			if !app.IsRunning(taskId) {
				return errorx.NewBiz("transfer stopped")
			}

			currentDumpTable := tables[0]
			pr, pw := io.Pipe()
			go func() {
				err := app.dbApp.DumpDb(ctx, &dto.DumpDb{
					LogId:        logId,
					DbId:         uint64(task.SrcDbId),
					DbName:       task.SrcDbName,
					TargetDbType: dbi.DbType(task.TargetDbType),
					Tables:       tables,
					DumpDDL:      true,
					DumpData:     true,
					Writer:       pw,
					Log: func(msg string) { // 记录日志
						app.Log(ctx, logId, msg)
					},
					Progress: func(currentTable string, stmtType dbi.StmtType, stmtCount int, currentStmtTypeEnd bool) {
						logExtraKey := fmt.Sprintf("`%s` amount of transfer data currently: ", currentDumpTable)
						if stmtType == dbi.StmtTypeInsert {
							app.logApp.SetExtra(logId, logExtraKey, stmtCount)
							if currentStmtTypeEnd {
								app.Log(ctx, logId, fmt.Sprintf("execute transfer table [%s] insert %d rows", currentDumpTable, stmtCount))
							}
						}

						if currentDumpTable != currentTable {
							currentDumpTable = currentTable
							stmtCount = 0
							// 置空当前表数据迁移量进度
							app.logApp.SetExtra(logId, logExtraKey, nil)
						}
					},
				})
				if err != nil {
					pr.CloseWithError(err)
					return
				}
			}()

			if err != nil {
				pw.CloseWithError(err)
				app.EndTransfer(ctx, logId, taskId, "transfer table failed", err, nil)
				return err
			}
			tx, _ := targetConn.Begin()
			err = sqlparser.SQLSplit(pr, func(stmt string) error {
				if _, err := targetConn.TxExec(tx, stmt); err != nil {
					app.EndTransfer(ctx, logId, taskId, fmt.Sprintf("执行sql出错: %s", stmt), err, nil)
					pw.CloseWithError(err)
					return err
				}
				return nil
			})

			if err != nil {
				tx.Rollback()
				return err
			}

			_ = tx.Commit()

			return nil
		})
	}

	err = errGroup.Wait()

	if err != nil {
		app.EndTransfer(ctx, logId, taskId, "transfer table failed", err, nil)
		return
	}
	app.EndTransfer(ctx, logId, taskId, fmt.Sprintf("execute transfer task [taskId = %d] complete, time: %v", taskId, time.Since(start)), nil, nil)
}

func (app *dbTransferAppImpl) transfer2File(ctx context.Context, taskId uint64, logId uint64, task *entity.DbTransferTask, srcConn *dbi.DbConn, start time.Time, tables []dbi.Table) {
	// 1、新增迁移文件数据
	nowTime := time.Now()
	tFile := &entity.DbTransferFile{
		TaskId:     taskId,
		CreateTime: &nowTime,
		Status:     entity.DbTransferFileStatusRunning,
		FileDbType: cmp.Or(task.TargetFileDbType, task.TargetDbType),
		LogId:      logId,
	}
	_ = app.transferFileApp.Save(ctx, tFile)

	filename := fmt.Sprintf("dtf_%s_%s.sql", task.TaskName, timex.TimeNo())
	fileKey, writer, saveFileFunc, err := app.fileApp.NewWriter(ctx, "", filename)
	if err != nil {
		app.EndTransfer(ctx, logId, taskId, "create file error", err, nil)
		return
	}

	// 从tables提取表名
	tableNames := collx.ArrayMap(tables, func(t dbi.Table) string { return t.TableName })
	// 2、把源库数据迁移到文件
	app.Log(ctx, logId, fmt.Sprintf("start transfer table data to files: %s", filename))
	app.Log(ctx, logId, fmt.Sprintf("dialect type of target db file: %s", task.TargetFileDbType))

	go func() {
		var err error
		defer saveFileFunc(&err)
		defer app.MarkStop(taskId)
		defer app.logApp.Flush(logId, true)
		ctx = context.Background()

		err = app.dbApp.DumpDb(ctx, &dto.DumpDb{
			LogId:        logId,
			DbId:         uint64(task.SrcDbId),
			DbName:       task.SrcDbName,
			TargetDbType: dbi.DbType(task.TargetFileDbType),
			Tables:       tableNames,
			DumpDDL:      true,
			DumpData:     true,
			Writer:       writer,
			Log: func(msg string) { // 记录日志
				app.Log(ctx, logId, msg)
			},
		})
		if err != nil {
			app.EndTransfer(ctx, logId, taskId, "db transfer to file failed", err, nil)
			tFile.Status = entity.DbTransferFileStatusFail
			_ = app.transferFileApp.UpdateById(ctx, tFile)
			return
		}
		app.EndTransfer(ctx, logId, taskId, "database transfer complete", err, nil)

		tFile.Status = entity.DbTransferFileStatusSuccess
		tFile.FileKey = fileKey
		_ = app.transferFileApp.UpdateById(ctx, tFile)
	}()
}

func (app *dbTransferAppImpl) Stop(ctx context.Context, taskId uint64) error {
	task, err := app.GetById(taskId)
	if err != nil {
		return errorx.NewBiz("task not found")
	}

	if task.RunningState != entity.DbTransferTaskRunStateRunning {
		return errorx.NewBiz("the task is not being executed")
	}
	task.RunningState = entity.DbTransferTaskRunStateStop
	if err = app.UpdateById(ctx, task); err != nil {
		return err
	}

	app.MarkStop(taskId)
	return nil
}

func (d *dbTransferAppImpl) TimerDeleteTransferFile() {
	logx.Debug("start deleting transfer files periodically...")
	scheduler.AddFun("@every 100m", func() {
		dts, err := d.ListByCond(model.NewCond().Eq("mode", entity.DbTransferTaskModeFile).Ge("file_save_days", 1))
		if err != nil {
			logx.Errorf("the task to periodically get database transfer to file failed: %s", err.Error())
			return
		}
		for _, dt := range dts {
			needDelFiles, err := d.transferFileApp.ListByCond(model.NewCond().Eq("task_id", dt.Id).Le("create_time", time.Now().AddDate(0, 0, -dt.FileSaveDays)))
			if err != nil {
				logx.Errorf("failed to obtain the transfer file periodically: %s", err.Error())
				continue
			}
			for _, nf := range needDelFiles {
				if err := d.transferFileApp.Delete(context.Background(), nf.Id); err != nil {
					logx.Errorf("failed to delete transfer files periodically: %s", err.Error())
				}
			}
		}
	})
}

// MarkRunning 标记任务执行中
func (app *dbTransferAppImpl) MarkRunning(taskId uint64) {
	cache.Set(fmt.Sprintf("mayfly:db:transfer:%d", taskId), 1, -1)
}

// MarkStop 标记任务结束
func (app *dbTransferAppImpl) MarkStop(taskId uint64) {
	cache.Del(fmt.Sprintf("mayfly:db:transfer:%d", taskId))
}

// IsRunning 判断任务是否执行中
func (app *dbTransferAppImpl) IsRunning(taskId uint64) bool {
	return cache.GetStr(fmt.Sprintf("mayfly:db:transfer:%d", taskId)) != ""
}

func (app *dbTransferAppImpl) Log(ctx context.Context, logId uint64, msg string, extra ...any) {
	logType := sysentity.SyslogTypeRunning
	logx.InfoContext(ctx, msg)
	app.logApp.AppendLog(logId, &sysapp.AppendLogReq{
		AppendResp: msg,
		Type:       logType,
	})
}

func (app *dbTransferAppImpl) EndTransfer(ctx context.Context, logId uint64, taskId uint64, msg string, err error, extra map[string]any) {
	app.MarkStop(taskId)

	logType := sysentity.SyslogTypeSuccess
	transferState := entity.DbTransferTaskRunStateSuccess
	if err != nil {
		msg = fmt.Sprintf("%s: %s", msg, err.Error())
		logx.ErrorContext(ctx, msg)
		logType = sysentity.SyslogTypeError
		transferState = entity.DbTransferTaskRunStateFail
	} else {
		logx.InfoContext(ctx, msg)
	}

	app.logApp.AppendLog(logId, &sysapp.AppendLogReq{
		AppendResp: msg,
		Extra:      extra,
		Type:       logType,
	})

	// 修改任务状态
	task := new(entity.DbTransferTask)
	task.Id = taskId
	task.RunningState = transferState
	app.UpdateById(context.Background(), task)
}

func (app *dbTransferAppImpl) RemoveCronJobById(taskId uint64) {
	task, err := app.GetById(taskId)
	if err == nil {
		scheduler.RemoveByKey(task.TaskKey)
	}
}

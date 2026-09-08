package transfer

import (
	"archive/zip"
	"cmp"
	"context"
	"fmt"
	"io"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/application/taskx"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	fileapp "mayfly-go/internal/file/application"
	sysapp "mayfly-go/internal/sys/application"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/scheduler"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/utils/timex"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

// DbApp 迁移链路依赖的窄接口：dump导出与按库取连接（Go惯例：消费者定义接口，
// 主包DbAppImpl天然满足，避免子包反向依赖主包造成循环）
type DbApp interface {
	DumpDb(ctx context.Context, reqParam *dto.DumpDb) error
	GetDbConn(ctx context.Context, dbId uint64, dbName string) (*dbi.DbConn, error)
}

type DbTransferTask interface {
	base.App[*entity.DbTransferTask]

	// GetPageList 分页获取数据库实例
	GetPageList(condition *entity.DbTransferTaskQuery, orderBy ...string) (*model.PageResult[*entity.DbTransferTask], error)

	Save(ctx context.Context, instanceEntity *entity.DbTransferTask) error

	Delete(ctx context.Context, id uint64) error

	InitCronJob()

	// Run 执行迁移任务
	// return logId, error
	Run(ctx context.Context, taskId uint64) (uint64, error)

	// Verify 校验迁移任务源库与目标库数据一致性
	// return logId, error
	Verify(ctx context.Context, taskId uint64) (uint64, error)

	IsRunning(taskId uint64) bool

	Stop(ctx context.Context, taskId uint64) error

	// TimerDeleteTransferFile 定时删除迁移文件
	TimerDeleteTransferFile()
}

var _ (DbTransferTask) = (*DbTransferAppImpl)(nil)

type DbTransferAppImpl struct {
	base.AppImpl[*entity.DbTransferTask, repository.DbTransferTask]

	dbApp           DbApp          `inject:"T"`
	logApp          sysapp.Syslog  `inject:"T"`
	transferFileApp DbTransferFile `inject:"T"`
	fileApp         fileapp.File   `inject:"T"`

	// runGuard 运行态原子守卫：替代原cache标记，修复IsRunning检查与标记间的竞态
	runGuard taskx.RunGuard

	// checkpointRepo 迁移断点检查点仓储
	checkpointRepo repository.DbTransferCheckpoint `inject:"T"`

	// cpStore 检查点存储：默认为gorm实现，测试可替换为内存实现
	cpStore checkpointStore
}

// getCheckpointStore 获取检查点存储（惰性构建gorm实现）
func (app *DbTransferAppImpl) getCheckpointStore() checkpointStore {
	if app.cpStore != nil {
		return app.cpStore
	}
	return newGormCheckpointStore(app.checkpointRepo)
}

func (app *DbTransferAppImpl) GetPageList(condition *entity.DbTransferTaskQuery, orderBy ...string) (*model.PageResult[*entity.DbTransferTask], error) {
	return app.GetRepo().GetTaskList(condition, orderBy...)
}

func (app *DbTransferAppImpl) Save(ctx context.Context, taskEntity *entity.DbTransferTask) error {
	var err error
	if taskEntity.Id == 0 { // 新建时生成key
		taskEntity.TaskKey = stringx.RandUUID()
		err = app.Insert(ctx, taskEntity)
	} else {
		if taskEntity.TaskKey == "" {
			task, err := app.GetById(taskEntity.Id)
			if err != nil {
				return errorx.NewBiz("db transfer task not found")
			}
			taskEntity.TaskKey = task.TaskKey
		}
		err = app.UpdateById(ctx, taskEntity)
	}
	if err != nil {
		return err
	}

	app.addCronJob(ctx, taskEntity)
	return nil
}

func (app *DbTransferAppImpl) Delete(ctx context.Context, id uint64) error {
	task, err := app.GetById(id)
	if err != nil {
		return errorx.NewBiz("db transfer task not found")
	}
	scheduler.RemoveByKey(task.TaskKey)
	app.runGuard.Release(id)

	return app.DeleteById(ctx, id)
}

func (app *DbTransferAppImpl) InitCronJob() {
	// 重启后，把正在运行的状态设置为停止
	// （运行守卫为进程内状态，重启后自然清空，无需遍历任务逐个释放）
	_ = app.UpdateByCond(context.TODO(), &entity.DbTransferTask{RunningState: entity.DbTransferTaskRunStateStop}, &entity.DbTransferTask{RunningState: entity.DbTransferTaskRunStateRunning})
	// 把所有运行中的文件状态设置为失败
	_ = app.transferFileApp.UpdateByCond(context.TODO(), &entity.DbTransferFile{Status: entity.DbTransferFileStatusFail}, &entity.DbTransferFile{Status: entity.DbTransferFileStatusRunning})

	if err := app.CursorByCond(&entity.DbTransferTaskQuery{Status: entity.DbTransferTaskStatusEnable, CronAble: entity.DbTransferTaskCronAbleEnable}, func(dtt *entity.DbTransferTask) error {
		app.addCronJob(contextx.WithTraceId(context.Background()), dtt)
		return nil
	}); err != nil {
		logx.ErrorTrace("the db data transfer task failed to initialize", err)
	}
}

func (app *DbTransferAppImpl) Run(ctx context.Context, taskId uint64) (uint64, error) {
	// 先原子占位再创建运行日志：顺序颠倒时，并发触发的失败方会留下一条永不结束（永不Flush）的Running日志
	if !app.runGuard.Acquire(taskId) {
		return 0, errorx.NewBizf("the db transfer task [%d] is running, please do not repeat the operation", taskId)
	}

	task, err := app.GetById(taskId)
	if err != nil {
		app.runGuard.Release(taskId)
		return 0, errorx.NewBizf("db transfer task [%d] not found", taskId)
	}

	logId, err := app.CreateLog(ctx, taskId)
	if err != nil {
		app.runGuard.Release(taskId)
		return 0, fmt.Errorf("create execution log for db transfer task [%d] failed: %w", taskId, err)
	}
	// 修改状态与关联日志id
	task.LogId = logId
	task.RunningState = entity.DbTransferTaskRunStateRunning
	if err = app.UpdateById(ctx, task); err != nil {
		app.runGuard.Release(taskId)
		return logId, err
	}

	// 协程池/调度等异常兜底：任务已占位则必须由本回调释放守卫并结束日志，
	// 否则任务永久停留在Running状态且运行守卫无法恢复（只能重启服务）
	gox.Go(func() {
		// 任务在后台异步执行，需脱离请求ctx取消信号并保留链路信息，
		// 否则请求响应后ctx取消导致迁移失败
		ctx = context.WithoutCancel(ctx)
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
		switch task.Mode {
		case entity.DbTransferTaskModeFile:
			app.transfer2File(ctx, logId, task, tables)
		case entity.DbTransferTaskModeDb:
			app.transfer2Db(ctx, logId, task, srcConn, tables)
		default:
			// 此时err必为nil（前序GetTables已成功），直接传err会把非法模式误记为成功状态
			app.EndTransfer(ctx, logId, taskId, "error in transfer mode, only migrating to files or databases is currently supported", errorx.NewBizf("unknown transfer mode [%d]", task.Mode), nil)
			return
		}
	}, func(panicErr error) {
		app.EndTransfer(ctx, logId, taskId, "db transfer task panicked", panicErr, nil)
	})

	return logId, nil
}

func (app *DbTransferAppImpl) transfer2Db(ctx context.Context, logId uint64, task *entity.DbTransferTask, srcConn *dbi.DbConn, tables []dbi.Table) {
	startTime := time.Now()
	taskId := task.Id
	defer app.runGuard.Release(taskId)
	defer app.logApp.Flush(logId, true)

	// 获取目标库连接
	targetConn, err := app.dbApp.GetDbConn(ctx, uint64(task.TargetDbId), task.TargetDbName)
	if err != nil {
		app.EndTransfer(ctx, logId, taskId, "failed to get target db connection", err, nil)
		return
	}

	tableNames := collx.ArrayMap(tables, func(t dbi.Table) string { return t.TableName })
	sort.Strings(tableNames)

	// 初始化/加载断点检查点（上次失败重跑时跳过已完成表，实现断点续传）
	cpStore := app.getCheckpointStore()
	cp, resume, err := initCheckpoint(cpStore, taskId, tableNames)
	if err != nil {
		app.EndTransfer(ctx, logId, taskId, "failed to init transfer checkpoint", err, nil)
		return
	}
	pending := tableNames
	if resume {
		pending = remainingTables(cp.PlannedTables, cp.DoneTables)
		app.Log(ctx, logId, fmt.Sprintf("resume from checkpoint: %d/%d tables done, %d tables remaining",
			len(cp.DoneTables), len(cp.PlannedTables), len(pending)))
		if len(pending) == 0 {
			// 上次在收尾前中断：无待迁表，直接完成
			if err := cpStore.Clear(taskId); err != nil {
				logx.Errorf("clear transfer checkpoint of task [%d] failed: %s", taskId, err.Error())
			}
			app.EndTransfer(ctx, logId, taskId, fmt.Sprintf("execute transfer task [taskId = %d] complete, time: %v", taskId, time.Since(startTime)), nil, nil)
			return
		}
	}

	concurrency := normalizeConcurrency(task.Concurrency)

	// 聚合迁移进度指标：经logApp.SetExtra写入日志extra实时呈现
	// （totalTables/completedTables/totalRows/rowsPerSec，Flush后随日志持久化）
	var totalRows, completedTables int64
	if resume {
		completedTables = int64(len(cp.DoneTables))
	}
	progressStart := time.Now()
	updateProgressExtra := func() {
		rows := atomic.LoadInt64(&totalRows)
		app.logApp.SetExtra(logId, "transferTotalTables", len(tableNames))
		app.logApp.SetExtra(logId, "transferCompletedTables", int(atomic.LoadInt64(&completedTables)))
		app.logApp.SetExtra(logId, "transferTotalRows", int(rows))
		if elapsed := time.Since(progressStart).Seconds(); elapsed > 0.5 {
			app.logApp.SetExtra(logId, "transferRowsPerSec", int(float64(rows)/elapsed))
		}
	}
	updateProgressExtra()

	// 阶段1：DDL先行（串行）——所有待迁表先建好（DDL含DROP重建，顺带清理上次失败遗留的部分数据），
	// 解除阶段2分片并行导入的表间依赖
	for _, tableName := range pending {
		if err := app.transferTableDDL2Db(ctx, logId, task, targetConn, tableName); err != nil {
			app.EndTransfer(ctx, logId, taskId, fmt.Sprintf("transfer table [%s] ddl failed", tableName), err, nil)
			return
		}
	}

	// 阶段2：数据导入——按主键范围分片规划后进并发工作池（无分片表=整表单任务）。
	// tableRemaining记录各表剩余分片数，归零时记录表级断点检查点
	type dataTask struct {
		table string
		where string
	}
	var dataTasks []dataTask
	tableRemaining := make(map[string]int)
	for _, tableName := range pending {
		var tableRows int
		for _, tb := range tables {
			if tb.TableName == tableName {
				tableRows = tb.TableRows
				break
			}
		}
		wheres := app.planTableShards(ctx, logId, srcConn, tableName, tableRows)
		if len(wheres) == 0 {
			wheres = []string{""}
		}
		tableRemaining[tableName] = len(wheres)
		for _, w := range wheres {
			dataTasks = append(dataTasks, dataTask{table: tableName, where: w})
		}
	}

	errGroup, gctx := errgroup.WithContext(ctx)
	errGroup.SetLimit(concurrency)
	var mu sync.Mutex
	for _, dt := range dataTasks {
		errGroup.Go(func() (taskErr error) {
			// panic必须转化为错误上报：仅recover不置错会使分片失败被当作成功，
			// 最终任务标成功但实际丢数据（errgroup不会自行recover，故此处recover仍需保留）
			defer gox.Recover(func(e error) {
				if taskErr == nil {
					taskErr = fmt.Errorf("transfer table [%s] data panicked: %w", dt.table, e)
				}
			})

			if !app.runGuard.IsRunning(taskId) {
				return errorx.NewBiz("transfer stopped")
			}

			// 分片内行数增量统计：Progress回调按分片内累计值上报，做差后原子累加全局计数
			lastStmtCount := 0
			if err := app.transferTableData2Db(gctx, logId, task, targetConn, dt.table, dt.where, func(stmtCount int) {
				if stmtCount > lastStmtCount {
					atomic.AddInt64(&totalRows, int64(stmtCount-lastStmtCount))
					lastStmtCount = stmtCount
				}
			}); err != nil {
				return err
			}

			// 该表全部分片完成 → 原子记录断点检查点并刷新进度
			mu.Lock()
			tableRemaining[dt.table]--
			last := tableRemaining[dt.table] == 0
			mu.Unlock()
			if last {
				atomic.AddInt64(&completedTables, 1)
				updateProgressExtra()
				if err := cpStore.AppendDone(taskId, dt.table); err != nil {
					app.Log(gctx, logId, fmt.Sprintf("save transfer checkpoint failed: %s", err.Error()))
				}
			}
			return nil
		})
	}

	if err = errGroup.Wait(); err != nil {
		app.EndTransfer(ctx, logId, taskId, "transfer table failed", err, nil)
		return
	}
	// 全部完成 → 清除断点检查点并汇总耗时与总行数
	if err := cpStore.Clear(taskId); err != nil {
		logx.Errorf("clear transfer checkpoint of task [%d] failed: %s", taskId, err.Error())
	}
	app.EndTransfer(ctx, logId, taskId, fmt.Sprintf("execute transfer task [taskId = %d] complete, time: %v, total rows: %d", taskId, time.Since(startTime), atomic.LoadInt64(&totalRows)), nil, nil)
}

// transferTableDDL2Db 迁移单表结构：源库dump DDL → 目标库导入
func (app *DbTransferAppImpl) transferTableDDL2Db(ctx context.Context, logId uint64, task *entity.DbTransferTask, targetConn *dbi.DbConn, tableName string) error {
	return app.dumpAndImport(ctx, logId, targetConn, &dto.DumpDb{
		LogId:        logId,
		DbId:         uint64(task.SrcDbId),
		DbName:       task.SrcDbName,
		TargetDbType: dbi.DbType(task.TargetDbType),
		Tables:       []string{tableName},
		DumpDDL:      true,
		DumpData:     false,
		Log: func(msg string) { // 记录日志
			app.Log(ctx, logId, msg)
		},
	})
}

// transferTableData2Db 迁移单表数据：源库dump（可带where分片过滤）→ 目标库批级事务导入。
// where为空表示整表导入。onRows为分片内insert行数回调（可为nil），用于上层聚合迁移进度。
// 分片导入无DDL，失败重跑时由阶段1的DROP重建保证幂等。
func (app *DbTransferAppImpl) transferTableData2Db(ctx context.Context, logId uint64, task *entity.DbTransferTask, targetConn *dbi.DbConn, tableName, where string, onRows func(stmtCount int)) error {
	dump := &dto.DumpDb{
		LogId:        logId,
		DbId:         uint64(task.SrcDbId),
		DbName:       task.SrcDbName,
		TargetDbType: dbi.DbType(task.TargetDbType),
		Tables:       []string{tableName},
		DumpDDL:      false,
		DumpData:     true,
		Log: func(msg string) { // 记录日志
			app.Log(ctx, logId, msg)
		},
		Progress: func(currentTable string, stmtType dbi.StmtType, stmtCount int, currentStmtTypeEnd bool) {
			logExtraKey := fmt.Sprintf("`%s` amount of transfer data currently: ", currentTable)
			if stmtType == dbi.StmtTypeInsert {
				if onRows != nil {
					onRows(stmtCount)
				}
				app.logApp.SetExtra(logId, logExtraKey, stmtCount)
				if currentStmtTypeEnd {
					app.Log(ctx, logId, fmt.Sprintf("execute transfer table [%s] insert %d rows", currentTable, stmtCount))
				}
			}
		},
	}
	if where != "" {
		dump.TableFilter = map[string]string{tableName: where}
	}
	return app.dumpAndImport(ctx, logId, targetConn, dump)
}

// dumpAndImport 将dump生成的SQL产物流式导入目标库：dump侧goroutine写入管道，
// 导入侧从管道批级消费。任一侧失败即中止管道并等待对侧退出，避免goroutine与管道泄漏。
func (app *DbTransferAppImpl) dumpAndImport(ctx context.Context, logId uint64, targetConn *dbi.DbConn, dump *dto.DumpDb) error {
	pr, pw := io.Pipe()
	defer pr.Close()
	dump.Writer = pw
	dumpDone := make(chan error, 1)

	gox.Go(func() {
		defer pw.Close()
		err := app.dbApp.DumpDb(ctx, dump)
		dumpDone <- err
		if err != nil {
			app.Log(ctx, logId, fmt.Sprintf("db dump failed: %s", err.Error()))
			pr.CloseWithError(err)
		}
	}, func(panicErr error) {
		// dump协程panic时必须也投递结果并中断管道，否则消费侧<-dumpDone永久阻塞（任务挂死、守卫不释放）；
		// 用select+default防止panic发生在首次投递之后时重复发送阻塞
		select {
		case dumpDone <- fmt.Errorf("db dump panicked: %w", panicErr):
		default:
		}
		pr.CloseWithError(panicErr)
	})

	if err := app.importDumpStream(ctx, logId, targetConn, pr); err != nil {
		// 通知dump侧中止并等待其退出，避免goroutine与管道泄漏
		pr.CloseWithError(err)
		<-dumpDone
		return err
	}
	return <-dumpDone
}

func (app *DbTransferAppImpl) transfer2File(ctx context.Context, logId uint64, task *entity.DbTransferTask, tables []dbi.Table) {
	taskId := task.Id
	// 1、新增迁移文件数据
	nowTime := time.Now()
	tFile := &entity.DbTransferFile{
		TaskId:     taskId,
		CreateTime: &nowTime,
		Status:     entity.DbTransferFileStatusRunning,
		FileDbType: cmp.Or(task.TargetFileDbType, task.TargetDbType),
		LogId:      logId,
	}
	// 后续异步写入依赖tFile.Id更新状态，保存失败需直接中止迁移
	if err := app.transferFileApp.Save(ctx, tFile); err != nil {
		app.EndTransfer(ctx, logId, taskId, "failed to save transfer file record", err, nil)
		return
	}

	fileType := cmp.Or(task.GetExtraString("fileType"), "sql")
	// 文件类型白名单：Extra可被任务保存接口写入，非预期值（如含路径分隔符）会污染存储文件名与扩展名
	if fileType != "sql" && fileType != "zip" {
		app.Log(ctx, logId, fmt.Sprintf("unsupported transfer file type [%s], fallback to sql", fileType))
		fileType = "sql"
	}
	filename := fmt.Sprintf("dtf_%s.%s", timex.TimeNo(), fileType)
	fileKey, writer, closeFunc, err := app.fileApp.NewWriter(ctx, "", filename)
	if err != nil {
		app.EndTransfer(ctx, logId, taskId, "create file error", err, nil)
		return
	}

	// 从tables提取表名
	tableNames := collx.ArrayMap(tables, func(t dbi.Table) string { return t.TableName })
	// 2、把源库数据迁移到文件
	app.Log(ctx, logId, fmt.Sprintf("start transfer table data to files: %s", filename))
	app.Log(ctx, logId, fmt.Sprintf("dialect type of target db file: %s", task.TargetFileDbType))

	gox.Go(func() {
		var err error

		// defer LIFO下注册越靠后执行越早，以下注册顺序确保实际执行顺序为：
		// panic捕获 → 关闭/保存文件 → 任务与文件状态收尾 → 释放运行守卫 → Flush日志
		defer app.logApp.Flush(logId, true)
		defer app.runGuard.Release(taskId)

		// 任务状态与文件记录统一收尾：必须在closeFunc之后执行，
		// closeFunc才会真正关闭写入流并完成文件落盘/上传与记录保存，
		// 否则会出现“记录已标成功但物理文件不存在/关闭失败未反映到状态”的不一致
		defer func() {
			if err != nil {
				app.EndTransfer(ctx, logId, taskId, "db transfer to file failed", err, nil)
				tFile.Status = entity.DbTransferFileStatusFail
			} else {
				app.EndTransfer(ctx, logId, taskId, "database transfer complete", nil, nil)
				tFile.Status = entity.DbTransferFileStatusSuccess
				tFile.FileKey = fileKey
			}
			if updErr := app.transferFileApp.UpdateById(ctx, tFile); updErr != nil {
				logx.ErrorfContext(ctx, "failed to update transfer file [%d] status: %s", tFile.Id, updErr.Error())
			}
		}()

		// closeFunc自身失败（关闭/上传/保存新文件失败并已删除物理文件）也必须并入err
		defer func() {
			if closeErr := closeFunc(&err); closeErr != nil && err == nil {
				err = fmt.Errorf("failed to close and save the transfer file: %w", closeErr)
			}
		}()
		defer gox.Recover(func(e error) {
			if err == nil {
				err = e
			}
		})

		ctx = contextx.WithTraceId(context.Background())

		if fileType == "zip" {
			zipWriter := zip.NewWriter(writer)
			defer zipWriter.Close()
			// 创建SQL文件在ZIP内的条目
			writer, err = zipWriter.Create(strings.TrimSuffix(filename, ".zip") + ".sql")
			if err != nil {
				return
			}
		}

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
			// 任务/文件状态由上方收尾defer统一按err落定（含closeFunc失败），此处不重复结束
			return
		}
	})
}

func (app *DbTransferAppImpl) Stop(ctx context.Context, taskId uint64) error {
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

	app.runGuard.Release(taskId)
	return nil
}

func (d *DbTransferAppImpl) TimerDeleteTransferFile() {
	logx.Debug("start deleting transfer files periodically...")
	scheduler.AddFun("@every 100m", func() {
		defer gox.Recover()
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

func (app *DbTransferAppImpl) addCronJob(ctx context.Context, taskEntity *entity.DbTransferTask) {
	key := taskEntity.TaskKey
	enabled := taskEntity.Status == entity.DbTransferTaskStatusEnable && taskEntity.CronAble == entity.DbTransferTaskCronAbleEnable
	if !enabled {
		taskx.UnbindCronTask(key)
		return
	}

	taskId := taskEntity.Id
	// 统一内核：移除旧绑定后按状态注册新任务
	if err := taskx.BindCronTask(key, taskEntity.Cron, true, func() {
		logx.Infof("start the transfer task: %d", taskId)
		if _, err := app.Run(ctx, taskId); err != nil {
			logx.Warn(err.Error())
		}
	}); err != nil {
		logx.ErrorTrace("add db transfer cron job failed", err)
	}
}

// IsRunning 判断任务是否执行中（供api层查询展示）
func (app *DbTransferAppImpl) IsRunning(taskId uint64) bool {
	return app.runGuard.IsRunning(taskId)
}

func (app *DbTransferAppImpl) CreateLog(ctx context.Context, taskId uint64) (uint64, error) {
	logId, err := app.logApp.CreateLog(ctx, &sysapp.CreateLogReq{
		Description: "DBMS - Execution DB Transfer",
		ReqParam:    collx.Kvs("taskId", taskId),
		Type:        sysentity.SyslogTypeRunning,
		Resp:        "Data transfer starts...",
	})
	return logId, err
}

func (app *DbTransferAppImpl) Log(ctx context.Context, logId uint64, msg string, extra ...any) {
	logType := sysentity.SyslogTypeRunning
	logx.InfoContext(ctx, msg)
	if app.logApp == nil {
		// 集成测试等未注入日志应用的场景下仅保留本地日志
		return
	}
	app.logApp.AppendLog(logId, &sysapp.AppendLogReq{
		AppendResp: msg,
		Type:       logType,
	})
}

func (app *DbTransferAppImpl) EndTransfer(ctx context.Context, logId uint64, taskId uint64, msg string, err error, extra map[string]any) {
	app.runGuard.Release(taskId)

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
	if err := app.UpdateById(context.Background(), task); err != nil {
		logx.Errorf("failed to update transfer task [%d] running state: %s", taskId, err.Error())
	}
}

// GetDbTransferTaskApp 获取迁移任务应用门面
func GetDbTransferTaskApp() DbTransferTask {
	return ioc.Get[DbTransferTask]()
}

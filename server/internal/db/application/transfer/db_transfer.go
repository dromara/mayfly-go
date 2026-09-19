package transfer

import (
	"archive/zip"
	"cmp"
	"context"
	"fmt"
	"io"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	fileapp "mayfly-go/internal/file/application"
	sysapp "mayfly-go/internal/sys/application"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/scheduler"
	"mayfly-go/pkg/taskx"
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

	// GetLogList 获取任务执行日志列表
	GetLogList(condition *entity.DbTransferLogQuery, orderBy ...string) (*model.PageResult[*entity.DbTransferLog], error)
}

var _ (DbTransferTask) = (*DbTransferAppImpl)(nil)

type DbTransferAppImpl struct {
	base.AppImpl[*entity.DbTransferTask, repository.DbTransferTask]

	dbApp           DbApp          `inject:"T"`
	logApp          sysapp.Syslog  `inject:"T"`
	transferFileApp DbTransferFile `inject:"T"`
	fileApp         fileapp.File   `inject:"T"`

	// runGuard 运行态原子守卫：替代原cache标记，修复IsRunning检查与标记间的竞态
	runGuard taskx.RunGuard[uint64]

	// checkpointRepo 迁移断点检查点仓储
	checkpointRepo repository.DbTransferCheckpoint `inject:"T"`

	// transferLogRepo 迁移执行日志仓储
	transferLogRepo repository.DbTransferLog `inject:"T"`

	// cpStore 检查点存储：默认为gorm实现，测试可替换为内存实现
	cpStore checkpointStore

	// pipeline 迁移流水线（惰性初始化，策略注册表模式）
	pipeline *TransferPipeline
}

// getCheckpointStore 获取检查点存储（惰性构建gorm实现）
func (app *DbTransferAppImpl) getCheckpointStore() checkpointStore {
	if app.cpStore != nil {
		return app.cpStore
	}
	return newGormCheckpointStore(app.checkpointRepo)
}

// getPipeline 获取迁移流水线（惰性初始化，注册所有内置策略）
func (app *DbTransferAppImpl) getPipeline() *TransferPipeline {
	if app.pipeline != nil {
		return app.pipeline
	}
	registry := DefaultTransferStrategyRegistry(app)
	app.pipeline = NewTransferPipeline(registry)
	return app.pipeline
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

	// 清理关联的断点检查点，防止任务重建后加载旧检查点
	if cp, err := app.checkpointRepo.GetByTaskId(id); err == nil && cp != nil {
		if err := app.checkpointRepo.DeleteById(ctx, cp.Id); err != nil {
			logx.WarnfContext(ctx, "failed to delete transfer checkpoint for task [%d]: %s", id, err.Error())
		}
	}

	return app.DeleteById(ctx, id)
}

func (app *DbTransferAppImpl) InitCronJob() {
	ctx := contextx.WithTraceId(context.Background())
	// 重启后，把正在运行的状态设置为停止
	// （运行守卫为进程内状态，重启后自然清空，无需遍历任务逐个释放）
	if err := app.UpdateByCond(ctx, &entity.DbTransferTask{RunningState: entity.DbTransferTaskRunStateStop}, &entity.DbTransferTask{RunningState: entity.DbTransferTaskRunStateRunning}); err != nil {
		logx.ErrorfContext(ctx, "failed to reset running transfer tasks on startup: %s", err.Error())
	}
	// 把所有运行中的文件状态设置为失败
	if err := app.transferFileApp.UpdateByCond(ctx, &entity.DbTransferFile{Status: entity.DbTransferFileStatusFail}, &entity.DbTransferFile{Status: entity.DbTransferFileStatusRunning}); err != nil {
		logx.ErrorfContext(ctx, "failed to reset running transfer files on startup: %s", err.Error())
	}

	if err := app.CursorByCond(&entity.DbTransferTaskQuery{Status: entity.DbTransferTaskStatusEnable, CronAble: entity.DbTransferTaskCronAbleEnable}, func(dtt *entity.DbTransferTask) error {
		app.addCronJob(ctx, dtt)
		return nil
	}); err != nil {
		logx.ErrorTraceContext(ctx, "the db data transfer task failed to initialize", err)
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

	logx.InfofContext(ctx, "start the db transfer task [%d]: %s", taskId, task.TaskKey)

	// 协程池/调度等异常兜底：任务已占位则必须由本回调释放守卫并结束日志，
	// 否则任务永久停留在Running状态且运行守卫无法恢复（只能重启服务）
	gox.Go(func() {
		// 任务在后台异步执行，需脱离请求ctx取消信号并保留链路信息，
		// 否则请求响应后ctx取消导致迁移失败
		ctx = context.WithoutCancel(ctx)
		// 获取源库连接、目标库连接，判断连接可用性，否则记录日志：xx连接不可用
		// 获取源库表信息
		app.Log(ctx, logId, "[0/4] 连接源数据库...")
		srcConn, err := app.dbApp.GetDbConn(ctx, uint64(task.SrcDbId), task.SrcDbName)
		if err != nil {
			app.EndTransfer(ctx, logId, taskId, "failed to obtain source db connection", err, nil)
			return
		}
		app.Log(ctx, logId, fmt.Sprintf("✓ 源库连接成功: %s", task.SrcDbName))
		logx.InfofContext(ctx, "transfer task [%d]: source db connected", taskId)

		// 获取迁移表信息
		app.Log(ctx, logId, "获取源库表信息...")
		var tables []dbi.Table
		if task.CheckedKeys == "all" {
			tables, err = srcConn.Metadata().GetTables()
			if err != nil {
				app.EndTransfer(ctx, logId, taskId, "failed to get source table information", err, nil)
				return
			}
			app.Log(ctx, logId, fmt.Sprintf("✓ 获取全部表信息: %d 个表", len(tables)))
		} else {
			tableNames := strings.Split(task.CheckedKeys, ",")
			tables, err = srcConn.Metadata().GetTables(tableNames...)
			if err != nil {
				app.EndTransfer(ctx, logId, taskId, "failed to get source table information", err, nil)
				return
			}
			app.Log(ctx, logId, fmt.Sprintf("✓ 获取指定表信息: %d 个表", len(tables)))
		}

		// 通过策略注册表分派迁移模式（开闭原则：新增模式只需注册策略，无需修改此处）
		pctx := &PipelineContext{
			Task:    task,
			TaskId:  taskId,
			LogId:   logId,
			SrcConn: srcConn,
			Tables:  tables,
		}
		if err := app.getPipeline().Execute(ctx, pctx); err != nil {
			app.EndTransfer(ctx, logId, taskId, "unsupported transfer mode", err, nil)
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
	// runGuard 由 EndTransfer 统一释放，此处不重复释放

	app.Log(ctx, logId, "========================================")
	app.Log(ctx, logId, fmt.Sprintf("开始执行迁移任务: %s", task.TaskName))
	app.Log(ctx, logId, "========================================")
	app.Log(ctx, logId, "迁移模式: 数据库 -> 数据库")
	app.Log(ctx, logId, fmt.Sprintf("源库: %s", task.SrcDbName))
	app.Log(ctx, logId, fmt.Sprintf("目标库: %s", task.TargetDbName))
	app.Log(ctx, logId, fmt.Sprintf("待迁移表: %d 个", len(tables)))
	if task.Concurrency > 0 {
		app.Log(ctx, logId, fmt.Sprintf("并发度: %d", task.Concurrency))
	}

	// 获取目标库连接
	app.Log(ctx, logId, "----------------------------------------")
	app.Log(ctx, logId, "[1/4] 连接目标数据库...")
	targetConn, err := app.dbApp.GetDbConn(ctx, uint64(task.TargetDbId), task.TargetDbName)
	if err != nil {
		app.EndTransfer(ctx, logId, taskId, "failed to get target db connection", err, nil)
		return
	}
	app.Log(ctx, logId, fmt.Sprintf("✓ 目标库连接成功: %s", task.TargetDbName))

	tableNames := collx.ArrayMap(tables, func(t dbi.Table) string { return t.TableName })
	sort.Strings(tableNames)

	// 初始化/加载断点检查点（上次失败重跑时跳过已完成表，实现断点续传）
	app.Log(ctx, logId, "[2/4] 初始化断点检查点...")
	cpStore := app.getCheckpointStore()
	cp, resume, err := initCheckpoint(cpStore, taskId, tableNames)
	if err != nil {
		app.EndTransfer(ctx, logId, taskId, "failed to init transfer checkpoint", err, nil)
		return
	}
	pending := tableNames
	if resume {
		pending = remainingTables(cp.PlannedTables, cp.DoneTables)
		app.Log(ctx, logId, fmt.Sprintf("✓ 从断点恢复: %d/%d 表已完成，%d 表待迁移",
			len(cp.DoneTables), len(cp.PlannedTables), len(pending)))
		if len(pending) == 0 {
			// 上次在收尾前中断：无待迁表，直接完成
			if err := cpStore.Clear(taskId); err != nil {
				logx.ErrorfContext(ctx, "clear transfer checkpoint of task [%d] failed: %s", taskId, err.Error())
			}
			app.EndTransfer(ctx, logId, taskId, fmt.Sprintf("execute transfer task [taskId = %d] complete, time: %v", taskId, time.Since(startTime)), nil, nil)
			return
		}
	} else {
		app.Log(ctx, logId, "✓ 断点检查点初始化完成")
	}

	concurrency := normalizeConcurrency(task.Concurrency)

	// 聚合迁移进度指标（统一使用 mutex 保护，避免 atomic + mutex 混用）
	var totalRows int64
	var completedTablesCount int
	if resume {
		completedTablesCount = len(cp.DoneTables)
	}
	var progressMu sync.Mutex
	// 只更新缓存，不保存数据库（最后统一保存）
	updateProgress := func() {
		progressMu.Lock()
		rows := totalRows
		progressMu.Unlock()
		log := app.getTransferLog(logId)
		if log != nil {
			log.TableCount = len(tableNames)
			log.TotalRows = rows
			app.setTransferLog(logId, log)
		}
	}
	updateProgress()

	// 阶段1：DDL先行（串行）
	app.Log(ctx, logId, "[3/4] 阶段1: 迁移表结构 (DDL)...")
	if err := app.executeDDLPhase(ctx, logId, task, targetConn, pending); err != nil {
		app.EndTransfer(ctx, logId, taskId, fmt.Sprintf("transfer table [%s] ddl failed", err.Error()), err, nil)
		return
	}

	// 阶段2：数据导入（并发工作池）
	app.Log(ctx, logId, "[4/4] 阶段2: 迁移表数据...")
	finalRows, err := app.executeDataPhase(ctx, logId, task, srcConn, targetConn, tables, pending, concurrency, &totalRows, &completedTablesCount, &progressMu, updateProgress, cpStore)
	if err != nil {
		app.EndTransfer(ctx, logId, taskId, "transfer table failed", err, nil)
		return
	}
	// 最终同步 + 清除断点检查点
	updateProgress()
	if err := cpStore.Clear(taskId); err != nil {
		logx.ErrorfContext(ctx, "clear transfer checkpoint of task [%d] failed: %s", taskId, err.Error())
	}
	app.Log(ctx, logId, fmt.Sprintf("✓ 阶段2完成: %d 个表数据迁移成功", len(pending)))
	app.EndTransfer(ctx, logId, taskId, fmt.Sprintf("execute transfer task [taskId = %d] complete, time: %v, total rows: %d", taskId, time.Since(startTime), finalRows), nil, nil)
}

// executeDDLPhase 阶段1：串行迁移所有待迁表的结构（DDL含DROP重建）。
func (app *DbTransferAppImpl) executeDDLPhase(ctx context.Context, logId uint64, task *entity.DbTransferTask, targetConn *dbi.DbConn, pending []string) error {
	for i, tableName := range pending {
		app.Log(ctx, logId, fmt.Sprintf("  [%d/%d] 迁移表结构: %s", i+1, len(pending), tableName))
		if err := app.transferTableDDL2Db(ctx, logId, task, targetConn, tableName); err != nil {
			return fmt.Errorf("table [%s]: %s", tableName, err.Error())
		}
		app.Log(ctx, logId, fmt.Sprintf("  ✓ 表结构迁移完成: %s", tableName))
	}
	app.Log(ctx, logId, fmt.Sprintf("✓ 阶段1完成: %d 个表结构迁移成功", len(pending)))
	return nil
}

// executeDataPhase 阶段2：按主键分片规划后进并发工作池导入数据。
// 返回最终总行数。
func (app *DbTransferAppImpl) executeDataPhase(
	ctx context.Context, logId uint64, task *entity.DbTransferTask,
	srcConn *dbi.DbConn, targetConn *dbi.DbConn,
	tables []dbi.Table, pending []string,
	concurrency int,
	totalRows *int64, completedTablesCount *int,
	progressMu *sync.Mutex, updateProgress func(),
	cpStore checkpointStore,
) (int64, error) {
	taskId := task.Id
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
		wheres := app.PlanTableShards(ctx, logId, srcConn, tableName, tableRows)
		if len(wheres) == 0 {
			wheres = []string{""}
		}
		tableRemaining[tableName] = len(wheres)
		if len(wheres) > 1 {
			app.Log(ctx, logId, fmt.Sprintf("  表 %s: %d 个分片, 约 %d 行", tableName, len(wheres), tableRows))
		}
		for _, w := range wheres {
			dataTasks = append(dataTasks, dataTask{table: tableName, where: w})
		}
	}
	app.Log(ctx, logId, fmt.Sprintf("共 %d 个数据迁移任务 (含分片)", len(dataTasks)))

	errGroup, gctx := errgroup.WithContext(ctx)
	errGroup.SetLimit(concurrency)
	for _, dt := range dataTasks {
		errGroup.Go(func() (taskErr error) {
			defer gox.Recover(func(e error) {
				if taskErr == nil {
					taskErr = fmt.Errorf("transfer table [%s] data panicked: %w", dt.table, e)
				}
			})
			if !app.runGuard.IsRunning(taskId) {
				return errorx.NewBiz("transfer stopped")
			}
			lastStmtCount := 0
			if err := app.transferTableData2Db(gctx, logId, task, targetConn, dt.table, dt.where, func(stmtCount int) {
				if stmtCount < lastStmtCount {
					lastStmtCount = 0
				}
				if stmtCount > lastStmtCount {
					progressMu.Lock()
					*totalRows += int64(stmtCount - lastStmtCount)
					progressMu.Unlock()
					lastStmtCount = stmtCount
				}
			}); err != nil {
				return err
			}
			progressMu.Lock()
			tableRemaining[dt.table]--
			last := tableRemaining[dt.table] == 0
			if last {
				*completedTablesCount++
			}
			completed := *completedTablesCount
			progressMu.Unlock()
			if last {
				updateProgress()
				app.Log(gctx, logId, fmt.Sprintf("  ✓ 表数据迁移完成: %s (%d/%d)", dt.table, completed, len(pending)))
				if err := cpStore.AppendDone(taskId, dt.table); err != nil {
					app.Log(gctx, logId, fmt.Sprintf("保存断点检查点失败: %s", err.Error()))
				}
			}
			return nil
		})
	}
	if err := errGroup.Wait(); err != nil {
		return 0, err
	}
	progressMu.Lock()
	finalRows := *totalRows
	progressMu.Unlock()
	return finalRows, nil
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
			if stmtType == dbi.StmtTypeInsert {
				if onRows != nil {
					onRows(stmtCount)
				}
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

	if err := app.ImportDumpStream(ctx, logId, targetConn, pr); err != nil {
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
		// transfer2File的行数统计：DumpDbScript每表dataCount从0重新计数，
		// 用lastStmtCount差值法跨表累加（与transfer2Db同理）
		var fileTotalRows int64
		lastStmtCount := 0

		// defer LIFO下注册越靠后执行越早，以下注册顺序确保实际执行顺序为：
		// panic捕获 → 关闭/保存文件 → 任务与文件状态收尾
		// runGuard 由 EndTransfer 统一释放，此处不重复释放

		// 任务状态与文件记录统一收尾：必须在closeFunc之后执行，
		// closeFunc才会真正关闭写入流并完成文件落盘/上传与记录保存，
		// 否则会出现“记录已标成功但物理文件不存在/关闭失败未反映到状态”的不一致
		defer func() {
			// 更新迁移指标
			log := app.getTransferLog(logId)
			if log != nil {
				log.TableCount = len(tableNames)
				log.TotalRows = atomic.LoadInt64(&fileTotalRows)
				app.setTransferLog(logId, log)
			}
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
			// zip收尾失败（磁盘满/IO错误）必须并入err，否则文件损坏但状态标成功
			defer func() {
				if closeErr := zipWriter.Close(); closeErr != nil && err == nil {
					err = fmt.Errorf("failed to finalize zip archive: %w", closeErr)
				}
			}()
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
			Progress: func(currentTable string, stmtType dbi.StmtType, stmtCount int, currentStmtTypeEnd bool) {
				if stmtType == dbi.StmtTypeInsert {
					if stmtCount < lastStmtCount {
						lastStmtCount = 0
					}
					if stmtCount > lastStmtCount {
						atomic.AddInt64(&fileTotalRows, int64(stmtCount-lastStmtCount))
						lastStmtCount = stmtCount
					}
				}
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
	ctx := contextx.WithTraceId(context.Background())
	logx.DebugContext(ctx, "start deleting transfer files periodically...")
	scheduler.AddFun("@every 100m", func() {
		defer gox.Recover()
		dts, err := d.ListByCond(model.NewCond().Eq("mode", entity.DbTransferTaskModeFile).Ge("file_save_days", 1))
		if err != nil {
			logx.ErrorfContext(ctx, "the task to periodically get database transfer to file failed: %s", err.Error())
			return
		}
		for _, dt := range dts {
			needDelFiles, err := d.transferFileApp.ListByCond(model.NewCond().Eq("task_id", dt.Id).Le("create_time", time.Now().AddDate(0, 0, -dt.FileSaveDays)))
			if err != nil {
				logx.ErrorfContext(ctx, "failed to obtain the transfer file periodically: %s", err.Error())
				continue
			}
			for _, nf := range needDelFiles {
				if err := d.transferFileApp.Delete(context.Background(), nf.Id); err != nil {
					logx.ErrorfContext(ctx, "failed to delete transfer files periodically: %s", err.Error())
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
		logx.InfofContext(ctx, "start the transfer task: %d", taskId)
		if _, err := app.Run(ctx, taskId); err != nil {
			logx.WarnContext(ctx, err.Error())
		}
	}); err != nil {
		logx.ErrorTraceContext(ctx, "add db transfer cron job failed", err)
	}
}

// IsRunning 判断任务是否执行中（供api层查询展示）
func (app *DbTransferAppImpl) IsRunning(taskId uint64) bool {
	return app.runGuard.IsRunning(taskId)
}

func (app *DbTransferAppImpl) CreateLog(ctx context.Context, taskId uint64) (uint64, error) {
	task, err := app.GetById(taskId)
	if err != nil {
		return 0, err
	}
	now := time.Now()
	log := &entity.DbTransferLog{
		TaskId:     taskId,
		CreateTime: &now,
		Mode:       task.Mode,
		Status:     2, // 执行中（对应 entity DbTransferLog status: 2=执行中 1=成功 0=失败）
	}
	if err := app.transferLogRepo.Insert(ctx, log); err != nil {
		return 0, err
	}
	return log.Id, nil
}

// GetLogList 获取任务执行日志列表
func (app *DbTransferAppImpl) GetLogList(condition *entity.DbTransferLogQuery, orderBy ...string) (*model.PageResult[*entity.DbTransferLog], error) {
	res, err := app.transferLogRepo.GetLogList(condition, orderBy...)
	if err != nil {
		return nil, err
	}
	// 如果查询的是最新日志（第一页），尝试从缓存获取最新的 runLog
	if res != nil && len(res.List) > 0 && condition.PageNum <= 1 {
		latestLog := res.List[0]
		// 尝试从缓存获取最新日志（执行中的任务日志在缓存里）
		cachedLog := app.getTransferLog(latestLog.Id)
		if cachedLog != nil {
			if cachedLog.RunLog != "" {
				latestLog.RunLog = cachedLog.RunLog
			}
			// 缓存中的指标为实时值，覆盖DB中可能过时的数据
			if cachedLog.TotalRows > 0 {
				latestLog.TotalRows = cachedLog.TotalRows
			}
			if cachedLog.TableCount > 0 {
				latestLog.TableCount = cachedLog.TableCount
			}
			if cachedLog.DurationMs > 0 {
				latestLog.DurationMs = cachedLog.DurationMs
			}
		}
	}
	return res, nil
}

// Log 向运行日志追加一行带时间戳的记录，并同步更新缓存以供前端实时查看。
// 仅写入 RunLog（用户可见），不重复写入系统日志，与 sync 模块 appendRunLog 行为一致。
func (app *DbTransferAppImpl) Log(ctx context.Context, logId uint64, msg string) {
	log := app.getTransferLog(logId)
	if log != nil {
		ts := time.Now().Format("15:04:05.000")
		log.RunLog += fmt.Sprintf("[%s] %s\n", ts, msg)
		app.setTransferLog(logId, log)
	}
}

func (app *DbTransferAppImpl) EndTransfer(ctx context.Context, logId uint64, taskId uint64, msg string, err error, extra map[string]any) {
	// runGuard.Release 延迟到任务状态更新之后，防止窗口期内另一个 Run 获取守卫并启动

	transferState := entity.DbTransferTaskRunStateSuccess
	logStatus := int8(1) // 成功
	if err != nil {
		msg = fmt.Sprintf("%s: %s", msg, err.Error())
		logx.ErrorContext(ctx, msg)
		transferState = entity.DbTransferTaskRunStateFail
		logStatus = 0 // 失败
	} else {
		logx.InfoContext(ctx, msg)
	}

	// 追加最终消息到运行日志
	log := app.getTransferLog(logId)
	if log != nil {
		// 计算耗时（防止负数：CreateTime 可能因时区/缓存问题晚于当前时间）
		if log.CreateTime != nil {
			ms := time.Since(*log.CreateTime).Milliseconds()
			if ms > 0 {
				log.DurationMs = ms
			}
		}

		// 输出执行摘要
		app.Log(ctx, logId, "========================================")
		if err != nil {
			app.Log(ctx, logId, "迁移执行失败")
			app.Log(ctx, logId, fmt.Sprintf("错误信息: %s", err.Error()))
		} else {
			app.Log(ctx, logId, "迁移执行完成")
		}
		app.Log(ctx, logId, "========================================")
		if log.TableCount > 0 {
			app.Log(ctx, logId, fmt.Sprintf("迁移表数: %d", log.TableCount))
		}
		if log.TotalRows > 0 {
			app.Log(ctx, logId, fmt.Sprintf("迁移行数: %d", log.TotalRows))
		}
		if log.DurationMs > 0 {
			app.Log(ctx, logId, fmt.Sprintf("总耗时: %d ms", log.DurationMs))
			if log.TotalRows > 0 && log.DurationMs > 0 {
				throughput := log.TotalRows * 1000 / log.DurationMs
				app.Log(ctx, logId, fmt.Sprintf("吞吐量: %d 行/秒", throughput))
			}
		}
		app.Log(ctx, logId, "========================================")

		log.Status = logStatus
		log.ErrText = ""
		if err != nil {
			log.ErrText = err.Error()
		}
		// 落库
		if err := app.transferLogRepo.UpdateById(context.Background(), log); err != nil {
			logx.ErrorfContext(context.Background(), "failed to save transfer log [%d]: %s", log.Id, err.Error())
		}
	}

	// 修改任务状态
	task := new(entity.DbTransferTask)
	task.Id = taskId
	task.RunningState = transferState
	if err := app.UpdateById(context.Background(), task); err != nil {
		logx.ErrorfContext(context.Background(), "failed to update transfer task [%d] running state: %s", taskId, err.Error())
	}

	// 状态更新完成后再释放守卫，防止窗口期内另一个 Run 获取守卫并启动
	app.runGuard.Release(taskId)
}

// getTransferLog 获取迁移日志（优先从内存缓存）
func (app *DbTransferAppImpl) getTransferLog(logId uint64) *entity.DbTransferLog {
	if logId == 0 {
		return nil // logId=0 表示无日志上下文（如单测直接调用），跳过 DB 查询
	}
	log := new(entity.DbTransferLog)
	if cache.Get(getTransferLogKey(logId), log) {
		return log
	}
	if app.transferLogRepo == nil {
		return nil // 仓储未注入时（测试场景）安全返回
	}
	log, err := app.transferLogRepo.GetById(logId)
	if err != nil {
		return nil
	}
	if log != nil {
		app.setTransferLog(logId, log)
	}
	return log
}

// setTransferLog 设置迁移日志缓存
func (app *DbTransferAppImpl) setTransferLog(logId uint64, log *entity.DbTransferLog) {
	cache.Set(getTransferLogKey(logId), log, time.Duration(TransferLogCacheTTLSeconds)*time.Second)
}

func getTransferLogKey(logId uint64) string {
	return fmt.Sprintf("mayfly:db_transfer_log:%d", logId)
}

// GetDbTransferTaskApp 获取迁移任务应用门面
func GetDbTransferTaskApp() DbTransferTask {
	return ioc.Get[DbTransferTask]()
}

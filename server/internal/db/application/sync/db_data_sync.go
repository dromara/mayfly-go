package sync

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"mayfly-go/internal/db/application/taskx"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/scheduler"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cast"
)

// DbApp 数据同步依赖的窄接口：仅需按库获取连接（Go惯例：消费者定义接口，
// 主包DbAppImpl天然满足，避免子包反向依赖主包造成循环）
type DbApp interface {
	GetDbConn(ctx context.Context, dbId uint64, dbName string) (*dbi.DbConn, error)
}

type DataSyncTask interface {
	base.App[*entity.DataSyncTask]

	// GetPageList 分页获取数据库实例
	GetPageList(condition *entity.DataSyncTaskQuery, orderBy ...string) (*model.PageResult[*entity.DataSyncTask], error)

	Save(ctx context.Context, instanceEntity *entity.DataSyncTask) error

	Delete(ctx context.Context, id uint64) error

	InitCronJob()

	Run(ctx context.Context, id uint64) error

	StopTask(ctx context.Context, id uint64) error

	GetTaskLogList(condition *entity.DataSyncLogQuery, orderBy ...string) (*model.PageResult[*entity.DataSyncLog], error)
}

var _ (DataSyncTask) = (*DataSyncAppImpl)(nil)

type DataSyncAppImpl struct {
	base.AppImpl[*entity.DataSyncTask, repository.DataSyncTask]

	dbDataSyncLogRepo repository.DataSyncLog `inject:"T"`

	dbApp DbApp `inject:"T"`

	// runGuard 运行态原子守卫：替代原cache标记，修复IsRunning检查与标记间的竞态
	runGuard taskx.RunGuard
}

var (
	whereReg = regexp.MustCompile(`(?i)where`)

	// syncBatchMaxBytes 单个同步批次估算字节预算：防止少量大值行（text/blob）拼出
	// 超过目标库单包上限（mysql max_allowed_packet 5.7默认4MB）的INSERT
	syncBatchMaxBytes = 2 << 20
)

func (app *DataSyncAppImpl) GetPageList(condition *entity.DataSyncTaskQuery, orderBy ...string) (*model.PageResult[*entity.DataSyncTask], error) {
	return app.GetRepo().GetTaskList(condition, orderBy...)
}

func (app *DataSyncAppImpl) Save(ctx context.Context, taskEntity *entity.DataSyncTask) error {
	// 保存时校验DataSql与增量字段合法性（运行时Run内还会再校验，双重拦截）
	dialect, err := app.getSrcDialect(taskEntity)
	if err != nil {
		return err
	}
	if err := validateDataSyncSql(dialect, taskEntity); err != nil {
		return err
	}

	if taskEntity.Id == 0 {
		// 新建时生成key
		taskEntity.TaskKey = stringx.RandUUID()
		err = app.Insert(ctx, taskEntity)
	} else {
		if taskEntity.TaskKey == "" {
			task, err := app.GetById(taskEntity.Id)
			if err != nil {
				return errorx.NewBiz("db sync task not found")
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

func (app *DataSyncAppImpl) Delete(ctx context.Context, id uint64) error {
	task, err := app.GetById(id)
	if err != nil {
		return errorx.NewBiz("sync task not found")
	}
	scheduler.RemoveByKey(task.TaskKey)
	app.runGuard.Release(id)

	return app.DeleteById(ctx, id)
}

func (app *DataSyncAppImpl) Run(ctx context.Context, id uint64) error {
	if app.runGuard.IsRunning(id) {
		logx.Warnf("[%d] the db sync task is running...", id)
		return nil
	}

	task, err := app.GetById(id)
	if err != nil {
		return errorx.NewBiz("task not found")
	}

	logx.InfofContext(ctx, "start the data sync task: %s => %s", task.TaskName, task.TaskKey)

	if task.RunningState == entity.DataSyncTaskRunStateRunning {
		return errorx.NewBiz("the task is in progress")
	}

	// 原子标记该任务运行中：手动执行与cron同时触发时仅一方成功
	if !app.runGuard.Acquire(id) {
		logx.Warnf("[%d] the db sync task is running...", id)
		return nil
	}

	updateStateTask := &entity.DataSyncTask{
		RunningState: entity.DataSyncTaskRunStateRunning,
	}
	updateStateTask.Id = id
	if err := app.UpdateById(ctx, updateStateTask); err != nil {
		app.runGuard.Release(id)
		return errorx.NewBizf("failed to update task running state: %s", err.Error())
	}

	gox.Go(func() {
		// 任务在后台异步执行，请求ctx随响应结束而取消；
		// 需脱离请求取消信号但保留traceId等链路信息，否则后续日志与同步执行均报ctx canceled
		ctx = context.WithoutCancel(ctx)
		now := time.Now()
		syncLog := &entity.DataSyncLog{
			TaskId:     task.Id,
			CreateTime: &now,
			Status:     entity.DataSyncTaskStateFail, // 默认失败
		}

		defer app.endRunning(task, syncLog)
		defer gox.Recover(func(err error) {
			syncLog.ErrText = i18n.T(imsg.DataSyncFailMsg, "msg", err.Error())
			logx.ErrorContext(ctx, syncLog.ErrText)
			syncLog.Status = entity.DataSyncTaskStateFail
		})

		// 获取源库连接：同时用于运行时双重校验方言与后续增量字段类型探测
		srcConn, err := app.dbApp.GetDbConn(ctx, uint64(task.SrcDbId), task.SrcDbName)
		if err != nil {
			syncLog.ErrText = i18n.T(imsg.DataSyncFailMsg, "msg", err.Error())
			logx.ErrorContext(ctx, syncLog.ErrText)
			return
		}
		dialect := srcConn.GetDialect()
		// 运行时双重拦截：存量脏数据（校验引入前创建的任务）防御
		if verr := validateDataSyncSql(dialect, task); verr != nil {
			syncLog.ErrText = i18n.T(imsg.DataSyncFailMsg, "msg", verr.Error())
			logx.ErrorContext(ctx, syncLog.ErrText)
			return
		}

		// 通过占位符格式化sql
		updSql := ""
		orderSql := ""
		if task.UpdFieldVal != "0" && task.UpdFieldVal != "" && task.UpdField != "" {
			updFieldDataType := dbi.DefaultDbDataType
			_, probeErr := srcConn.WalkQueryRows(ctx, task.DataSql, func(row map[string]any, columns []*dbi.QueryColumn) error {
				for _, column := range columns {
					if strings.EqualFold(column.Name, cmp.Or(task.UpdFieldSrc, task.UpdField)) {
						updFieldDataType = column.DbDataType
						break
					}
				}
				return dbi.NewStopWalkQueryError("get column data type... ignore~")
			})
			// 探测失败不中止（后续正式查询会以相同原因显式失败），但不可静默丢弃错误
			if probeErr != nil {
				logx.WarnfContext(ctx, "failed to probe the data type of sync update field [%s]: %s", task.UpdField, probeErr.Error())
			}

			updSql = fmt.Sprintf("and %s > %s", task.UpdField, updFieldDataType.DataType.SQLValue(task.UpdFieldVal))
		}

		// 即使是首次同步，如果有更新字段也要添加排序，确保每次查询结果顺序一致，避免首次未同步完后续增量同步时漏数据或重复数据
		if task.UpdField != "" {
			orderSql = "order by " + task.UpdField + " asc "
		}

		// 解析器判断DataSql是否已含where条件，若是则不添加where 1 = 1
		// （替代旧(?i)where正则，避免字段名/字符串字面量含"where"子串时误判）
		var where = "where 1=1"
		if dataSqlHasWhere(dialect, task) {
			where = ""
		}

		// DataSql尾部可能带分号（用户从脚本粘贴），拼接where/order后会生成非法SQL，需先剖离
		dataSql := strings.TrimRight(strings.TrimSpace(task.DataSql), ";")

		// 组装查询sql
		sqlStr := fmt.Sprintf("%s %s %s %s", dataSql, where, updSql, orderSql)
		syncLog.DataSqlFull = sqlStr

		err = app.doDataSync(ctx, sqlStr, task, syncLog)
		if err != nil {
			syncLog.ErrText = i18n.T(imsg.DataSyncFailMsg, "msg", err.Error())
			logx.ErrorContext(ctx, syncLog.ErrText)
			syncLog.Status = entity.DataSyncTaskStateFail
		} else {
			syncLog.Status = entity.DataSyncTaskStateSuccess
		}
	})

	return nil
}

func (app *DataSyncAppImpl) doDataSync(ctx context.Context, sql string, task *entity.DataSyncTask, syncLog *entity.DataSyncLog) error {
	// 获取源数据库连接
	srcConn, err := app.dbApp.GetDbConn(ctx, uint64(task.SrcDbId), task.SrcDbName)

	if err != nil {
		return errorx.NewBizf("failed to connect to the source database: %s", err.Error())
	}

	// 获取目标数据库连接
	targetConn, err := app.dbApp.GetDbConn(ctx, uint64(task.TargetDbId), task.TargetDbName)
	if err != nil {
		return errorx.NewBizf("failed to connect to the target database: %s", err.Error())
	}

	// task.FieldMap为json数组字符串 [{"src":"id","target":"id"}]，转为map
	var fieldMap []map[string]string
	err = json.Unmarshal([]byte(task.FieldMap), &fieldMap)
	if err != nil {
		return errorx.NewBizf("there was an error parsing the field map json: %s", err.Error())
	}

	// 记录本次同步数据总数
	total := 0
	batchSize := normalizeBatchSize(task.PageSize)
	result := make([]map[string]any, 0)
	// 当前待提交批次估算字节数（行数+字节数双预算）
	pendingBytes := 0
	// 批间速率统计：上一批完成时刻
	lastBatchTime := time.Now()

	// 如果有数据库别名，则从UpdField中去掉数据库别名, 如：a.id => id，用于获取字段具体名称
	updFieldName := task.UpdField
	if task.UpdField != "" && strings.Contains(task.UpdField, ".") {
		updFieldName = strings.Split(task.UpdField, ".")[1]
	}

	targetTableColumns, err := targetConn.GetMetadata().GetColumns(task.TargetTableName)
	if err != nil {
		return errorx.NewBizf("failed to get target table columns: %s", err.Error())
	}
	targetColumnName2Column := collx.ArrayToMap(targetTableColumns, func(column dbi.Column) string {
		return column.ColumnName
	})

	// 目标库对应的insert columns
	targetInsertColumns := make([]dbi.Column, 0, len(fieldMap))
	for _, val := range fieldMap {
		column, ok := targetColumnName2Column[val["target"]]
		if !ok {
			return errorx.NewBizf("field map target column [%s] not found in target table [%s]", val["target"], task.TargetTableName)
		}
		// 目标列为生成列/计算列（MySQL生成列、pg STORED生成列、SQL Server计算列）时必须剔除：
		// 其值由目标库表达式派生，显式插入直接报错（MySQL 3105/pg 3402/SQL Server不能向计算列插入值），
		// 会使整条同步链路全量失败；此处基于目标表元数据判定，与SQL生成阶段的方言判定互不依赖
		if column.IsGenerated {
			logx.WarnfContext(ctx, "the target column [%s] of the synchronization task [%s] is a generated column, and its mapping will be ignored (the value is derived by the target database)", val["target"], task.TaskName)
			continue
		}
		targetInsertColumns = append(targetInsertColumns, column)
	}
	if len(targetInsertColumns) == 0 {
		// 所有映射列均为生成列时无列可插，继续执行只会产生空转或非法SQL，显式失败便于定位配置问题
		return errorx.NewBizf("no insertable columns are configured for the target table [%s]", task.TargetTableName)
	}

	// 构建目标表元信息（任务级只查询一次），用于生成 upsert/merge 类插入语句
	targetTableMeta := dbi.BuildTargetTableMeta(targetConn, task.TargetTableName, targetTableColumns)

	_, err = srcConn.WalkQueryRows(ctx, sql, func(row map[string]any, columns []*dbi.QueryColumn) error {
		total++
		result = append(result, row)
		pendingBytes += estimateRowBytes(row)
		// 行数与字节数双预算：固定行数分批在单行大值（大text/blob）场景下会生成
		// 超大单条INSERT，内存峰值暴涨且可能超目标库单包上限（如mysql max_allowed_packet）
		if total%batchSize == 0 || pendingBytes >= syncBatchMaxBytes {
			if err := app.srcData2TargetDb(ctx, result, fieldMap, updFieldName, task, targetConn, targetInsertColumns, targetTableMeta); err != nil {
				return err
			}

			// 批次提交成功后立即持久化同步水位，避免进程崩溃丢失进度（endRunning仅为兑底）
			app.persistUpdFieldVal(ctx, task)

			// 批间速率日志
			now := time.Now()
			if elapsed := now.Sub(lastBatchTime).Seconds(); elapsed > 0 {
				logx.DebugfContext(ctx, "data sync batch rate: %.0f rows/s", float64(batchSize)/elapsed)
			}
			lastBatchTime = now

			// 记录当前已同步的数据量
			syncLog.ErrText = i18n.T(imsg.DataSyncingMsg, "count", total)
			logx.InfoContext(ctx, syncLog.ErrText)
			syncLog.ResNum = total
			app.saveLog(syncLog)

			result = result[:0]
			pendingBytes = 0

			// 运行过程中，判断状态是否为已关闭，是则结束运行，否则继续运行
			if !app.runGuard.IsRunning(task.Id) {
				return errorx.NewBiz("the task has been terminated manually")
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 处理剩余的数据
	if len(result) > 0 {
		if err := app.srcData2TargetDb(ctx, result, fieldMap, updFieldName, task, targetConn, targetInsertColumns, targetTableMeta); err != nil {
			return err
		}
		app.persistUpdFieldVal(ctx, task)
	}

	logx.InfofContext(ctx, "synchronous task: [%s], finished execution, save records successfully: [%d]", task.TaskName, total)

	// 执行成功日志
	syncLog.ErrText = i18n.T(imsg.DataSyncSuccessMsg, "count", total)
	syncLog.ResNum = total

	return nil
}

func (app *DataSyncAppImpl) srcData2TargetDb(ctx context.Context, srcRes []map[string]any, fieldMap []map[string]string, updFieldName string, task *entity.DataSyncTask, targetDbConn *dbi.DbConn, targetInsertColumns []dbi.Column, targetTableMeta *dbi.TargetTableMeta) (err error) {
	// 遍历res，组装数据
	var targetData = make([]map[string]any, 0)
	for _, srcData := range srcRes {
		var data = make(map[string]any)
		// 遍历字段映射, target字段的值为src字段取值
		for _, item := range fieldMap {
			// target字段的值为src字段取值
			data[item["target"]] = srcData[item["src"]]
		}
		targetData = append(targetData, data)
	}

	targetValues := make([][]any, 0)
	for _, item := range targetData {
		var values = make([]any, 0)
		for _, column := range targetInsertColumns {
			values = append(values, item[column.ColumnName])
		}
		targetValues = append(targetValues, values)
	}

	// 执行插入
	targetDialect := targetDbConn.GetDialect()

	// 生成目标数据库批量插入sql，并执行
	sqls := targetDialect.GetSQLGenerator().GenInsert(task.TargetTableName, targetInsertColumns, targetValues, cmp.Or(task.DuplicateStrategy, dbi.DuplicateStrategyNone), targetTableMeta)

	// 开启本批次执行事务
	targetDbTx, err := targetDbConn.Begin()
	if err != nil {
		return errorx.NewBizf("failed to start the target database transaction: %s", err.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			dbi.RollbackTx(targetDbTx)
			err = fmt.Errorf("%v", r)
		}
	}()

	for _, sql := range sqls {
		if _, execErr := targetDbConn.TxExecContext(ctx, targetDbTx, sql); execErr != nil {
			dbi.RollbackTx(targetDbTx)
			return execErr
		}
	}

	// mssql驱动显式事务下Commit可能报no corresponding BEGIN TRANSACTION（驱动自管提交时序），
	// 与导入链路复用同一兼容处理，避免两处逻辑发散
	if commitErr := dbi.CommitTargetTx(targetDbConn, targetDbTx); commitErr != nil {
		return errorx.NewBizf("data synchronization - The target database transaction failed to commit: %s", commitErr.Error())
	}

	if watermarkErr := advanceSyncWatermark(srcRes, task, updFieldName); watermarkErr != nil {
		logx.WarnfContext(ctx, "failed to advance the data sync watermark of task [%d]: %s", task.Id, watermarkErr.Error())
	}

	return nil
}

// advanceSyncWatermark 推进同步水位：取本批最后一行的更新字段值。
// 旧实现取不到值（列名大小写差异、值为NULL）时会将水位写为空串，
// 下次同步从水位O重新并全量重跑，属同步正确性缺陷；
// 现在改为取不到有效值时保持原水位不变并记录原因
func advanceSyncWatermark(srcRes []map[string]any, task *entity.DataSyncTask, updFieldName string) error {
	if len(srcRes) == 0 {
		return nil
	}
	waterField := cmp.Or(task.UpdFieldSrc, updFieldName)
	if waterField == "" {
		return nil // 未配置增量字段，无需水位
	}
	val, ok := lookupRowValue(srcRes[len(srcRes)-1], waterField)
	if !ok {
		return fmt.Errorf("update field [%s] not found in the last synced row, keep watermark [%s] unchanged", waterField, task.UpdFieldVal)
	}
	if val == nil {
		return fmt.Errorf("update field [%s] of the last synced row is NULL, keep watermark [%s] unchanged", waterField, task.UpdFieldVal)
	}
	task.UpdFieldVal = cast.ToString(val)
	return nil
}

// lookupRowValue 从查询行中取列值：先按原名列命中，再大小写不敏感回退
// （不同驱动返回的列标签大小写不一致，如oracle为大写），并区分“列不存在”与“值为NULL”
func lookupRowValue(row map[string]any, column string) (any, bool) {
	if column == "" {
		return nil, false
	}
	if val, ok := row[column]; ok {
		return val, true
	}
	for name, val := range row {
		if strings.EqualFold(name, column) {
			return val, true
		}
	}
	return nil, false
}

// estimateRowBytes 估算一行的SQL字面量量级字节数（字符串/二进制取实际长度，其余按16字节估算）
func estimateRowBytes(row map[string]any) int {
	size := 0
	for _, val := range row {
		switch v := val.(type) {
		case string:
			size += len(v)
		case []byte:
			size += len(v)
		default:
			size += 16
		}
	}
	return size
}

// persistUpdFieldVal 批次提交成功后持久化同步水位（当前UpdFieldVal），
// 避免进程崩溃丢失进度只能靠endRunning兑底。
// 持久化失败仅记录日志不中断同步：endRunning仍会兑底落库，且水位幂等可重放
func (app *DataSyncAppImpl) persistUpdFieldVal(ctx context.Context, task *entity.DataSyncTask) {
	if task.UpdField == "" {
		return
	}
	ut := &entity.DataSyncTask{Id: task.Id, UpdFieldVal: task.UpdFieldVal}
	if err := app.UpdateById(ctx, ut); err != nil {
		logx.ErrorfContext(ctx, "failed to persist the data sync watermark: %s", err.Error())
	}
}

func (app *DataSyncAppImpl) StopTask(ctx context.Context, taskId uint64) error {
	task := new(entity.DataSyncTask)
	task.Id = taskId
	task.RunningState = entity.DataSyncTaskRunStateStop
	if err := app.UpdateById(ctx, task); err != nil {
		return err
	}
	app.runGuard.Release(taskId)
	return nil
}

func (app *DataSyncAppImpl) endRunning(taskEntity *entity.DataSyncTask, log *entity.DataSyncLog) {
	logx.Info(log.ErrText)

	state := log.Status
	task := new(entity.DataSyncTask)
	task.Id = taskEntity.Id
	task.RecentState = state
	task.UpdFieldVal = taskEntity.UpdFieldVal
	task.RunningState = entity.DataSyncTaskRunStateReady
	_ = app.UpdateById(context.Background(), task)
	// 保存执行日志
	app.saveLog(log)
	app.runGuard.Release(task.Id)
}

func (app *DataSyncAppImpl) saveLog(log *entity.DataSyncLog) {
	app.dbDataSyncLogRepo.Save(context.Background(), log)
}

func (app *DataSyncAppImpl) InitCronJob() {
	ctx := contextx.WithTraceId(context.Background())

	defer func() {
		if err := recover(); err != nil {
			logx.ErrorTraceContext(ctx, "the data synchronization task failed to initialize", err)
		}
	}()

	// 修改执行中状态为待执行
	_ = app.UpdateByCond(context.TODO(), &entity.DataSyncTask{RunningState: entity.DataSyncTaskRunStateReady}, &entity.DataSyncTask{RunningState: entity.DataSyncTaskRunStateRunning})

	if err := app.CursorByCond(&entity.DataSyncTaskQuery{Status: entity.DataSyncTaskStatusEnable}, func(dst *entity.DataSyncTask) error {
		app.addCronJob(ctx, dst)
		return nil
	}); err != nil {
		logx.ErrorTraceContext(ctx, "the db data sync task failed to initialize: %v", err)
	}
}

func (app *DataSyncAppImpl) GetTaskLogList(condition *entity.DataSyncLogQuery, orderBy ...string) (*model.PageResult[*entity.DataSyncLog], error) {
	return app.dbDataSyncLogRepo.GetTaskLogList(condition, orderBy...)
}

func (app *DataSyncAppImpl) addCronJob(ctx context.Context, taskEntity *entity.DataSyncTask) {
	key := taskEntity.TaskKey
	if taskEntity.Status != entity.DataSyncTaskStatusEnable {
		taskx.UnbindCronTask(key)
		return
	}

	taskId := taskEntity.Id
	// 统一内核：移除旧绑定后注册新任务
	logx.InfofContext(ctx, "start add the data sync task job: %s, cron[%s]", taskEntity.TaskName, taskEntity.TaskCron)
	if err := taskx.BindCronTask(key, taskEntity.TaskCron, true, func() {
		if err := app.Run(context.Background(), taskId); err != nil {
			logx.ErrorfContext(ctx, "the data sync task failed to execute at a scheduled time: %s", err.Error())
		}
	}); err != nil {
		logx.ErrorTraceContext(ctx, "add db data sync job failed", err)
	}
}

// normalizeBatchSize 存量任务的PageSize可能为0（api层required校验无法约束旧数据），
// 避免分批取模除零panic，无有效批次大小时使用默认值
func normalizeBatchSize(size int) int {
	if size <= 0 {
		return 500
	}
	return size
}

// GetDataSyncTaskApp 获取数据同步任务应用门面
func GetDataSyncTaskApp() DataSyncTask {
	return ioc.Get[DataSyncTask]()
}

// SyncBatch 单批同步执行：将源库查询结果经字段映射、目标方言INSERT生成（含冲突策略）
// 写入目标库。Run链路由doDataSync按预算分批调用本能力；导出为稳定入口供迁移联动、
// 流程编排等上层复用
func (app *DataSyncAppImpl) SyncBatch(ctx context.Context, srcRes []map[string]any, fieldMap []map[string]string, updFieldName string, task *entity.DataSyncTask, targetConn *dbi.DbConn, targetColumns []dbi.Column, targetTableMeta *dbi.TargetTableMeta) error {
	return app.srcData2TargetDb(ctx, srcRes, fieldMap, updFieldName, task, targetConn, targetColumns, targetTableMeta)
}

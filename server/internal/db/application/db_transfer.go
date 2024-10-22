package application

import (
	"cmp"
	"context"
	"encoding/hex"
	"fmt"
	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
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
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"golang.org/x/sync/errgroup"
)

type DbTransferTask interface {
	base.App[*entity.DbTransferTask]

	// GetPageList 分页获取数据库实例
	GetPageList(condition *entity.DbTransferTaskQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

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

type dbTransferAppImpl struct {
	base.AppImpl[*entity.DbTransferTask, repository.DbTransferTask]

	dbApp           Db             `inject:"DbApp"`
	logApp          sysapp.Syslog  `inject:"SyslogApp"`
	transferFileApp DbTransferFile `inject:"DbTransferFileApp"`
	fileApp         fileapp.File   `inject:"FileApp"`
}

func (app *dbTransferAppImpl) InjectDbTransferTaskRepo(repo repository.DbTransferTask) {
	app.Repo = repo
}

func (app *dbTransferAppImpl) GetPageList(condition *entity.DbTransferTaskQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.GetRepo().GetTaskList(condition, pageParam, toEntity, orderBy...)
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
		scheduler.AddFunByKey(key, taskEntity.Cron, func() {
			logx.Infof("开始执行同步任务: %d", taskId)
			logId, _ := app.CreateLog(ctx, taskId)
			app.Run(ctx, taskId, logId)
		})
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

	// 把所有需要定时执行的任务添加到定时任务中
	pageParam := &model.PageParam{
		PageSize: 100,
		PageNum:  1,
	}
	cond := new(entity.DbTransferTaskQuery)
	cond.Status = entity.DbTransferTaskStatusEnable
	cond.CronAble = entity.DbTransferTaskCronAbleEnable
	jobs := new([]entity.DbTransferTask)

	pr, _ := app.GetPageList(cond, pageParam, jobs)
	if nil == pr || pr.Total == 0 {
		return
	}
	total := pr.Total
	add := 0

	for {
		for _, job := range *jobs {
			app.AddCronJob(contextx.NewTraceId(), &job)
			add++
		}
		if add >= int(total) {
			return
		}
		pageParam.PageNum++
		_, _ = app.GetPageList(cond, pageParam, jobs)
	}
}

func (app *dbTransferAppImpl) CreateLog(ctx context.Context, taskId uint64) (uint64, error) {
	logId, err := app.logApp.CreateLog(ctx, &sysapp.CreateLogReq{
		Description: "DBMS-执行数据迁移",
		ReqParam:    collx.Kvs("taskId", taskId),
		Type:        sysentity.SyslogTypeRunning,
		Resp:        "开始执行数据迁移...",
	})
	return logId, err
}

func (app *dbTransferAppImpl) Run(ctx context.Context, taskId uint64, logId uint64) {

	task, err := app.GetById(taskId)
	if err != nil {
		logx.Errorf("创建DBMS-执行数据迁移日志失败：%v", err)
		return
	}

	if app.IsRunning(taskId) {
		logx.Panicf("[%d]该任务正在运行中...", taskId)
		return
	}

	start := time.Now()
	// 修改状态与关联日志id
	task.LogId = logId
	task.RunningState = entity.DbTransferTaskRunStateRunning
	if err = app.UpdateById(ctx, task); err != nil {
		logx.Errorf("更新任务执行状态失败")
		return
	}

	// 标记该任务开始执行
	app.MarkRunning(taskId)

	// 获取源库连接、目标库连接，判断连接可用性，否则记录日志：xx连接不可用
	// 获取源库表信息
	srcConn, err := app.dbApp.GetDbConn(uint64(task.SrcDbId), task.SrcDbName)
	if err != nil {
		app.EndTransfer(ctx, logId, taskId, "获取源库连接失败", err, nil)
		return
	}

	// 获取迁移表信息
	var tables []dbi.Table
	if task.CheckedKeys == "all" {
		tables, err = srcConn.GetMetaData().GetTables()
		if err != nil {
			app.EndTransfer(ctx, logId, taskId, "获取源表信息失败", err, nil)
			return
		}
	} else {
		tableNames := strings.Split(task.CheckedKeys, ",")
		tables, err = srcConn.GetMetaData().GetTables(tableNames...)
		if err != nil {
			app.EndTransfer(ctx, logId, taskId, "获取源表信息失败", err, nil)
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
		app.EndTransfer(ctx, logId, taskId, "迁移模式出错，目前仅支持迁移到文件或数据库", err, nil)
		return
	}
}

func (app *dbTransferAppImpl) transfer2Db(ctx context.Context, taskId uint64, logId uint64, task *entity.DbTransferTask, srcConn *dbi.DbConn, start time.Time, tables []dbi.Table) {
	// 获取目标库表信息
	targetConn, err := app.dbApp.GetDbConn(uint64(task.TargetDbId), task.TargetDbName)
	if err != nil {
		app.EndTransfer(ctx, logId, taskId, "获取目标库连接失败", err, nil)
		return
	}
	// 迁移表
	if err = app.transferDbTables(ctx, logId, task, srcConn, targetConn, tables); err != nil {
		app.EndTransfer(ctx, logId, taskId, "迁移表失败", err, nil)
		return
	}

	app.EndTransfer(ctx, logId, taskId, fmt.Sprintf("执行迁移完成，执行迁移任务[taskId = %d]完成, 耗时：%v", taskId, time.Since(start)), nil, nil)
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
		app.EndTransfer(ctx, logId, taskId, "创建文件失败", err, nil)
		return
	}

	// 从tables提取表名
	tableNames := make([]string, 0)
	for _, table := range tables {
		tableNames = append(tableNames, table.TableName)
	}
	// 2、把源库数据迁移到文件
	app.Log(ctx, logId, fmt.Sprintf("开始迁移表数据到文件： %s", filename))

	app.Log(ctx, logId, fmt.Sprintf("目标库文件语言类型： %s", task.TargetFileDbType))

	go func() {
		defer saveFileFunc()
		defer app.MarkStop(taskId)
		defer app.logApp.Flush(logId, true)
		ctx = context.Background()
		err := app.dbApp.DumpDb(ctx, &dto.DumpDb{
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
			app.EndTransfer(ctx, logId, taskId, "数据库迁移失败", err, nil)
			tFile.Status = entity.DbTransferFileStatusFail
			_ = app.transferFileApp.UpdateById(ctx, tFile)
			// 删除文件
			_ = app.fileApp.Remove(ctx, fileKey)
			return
		}
		app.EndTransfer(ctx, logId, taskId, "数据库迁移完成", err, nil)

		tFile.Status = entity.DbTransferFileStatusSuccess
		tFile.FileKey = fileKey
		_ = app.transferFileApp.UpdateById(ctx, tFile)
	}()

}

func (app *dbTransferAppImpl) Stop(ctx context.Context, taskId uint64) error {
	task, err := app.GetById(taskId)
	if err != nil {
		return errorx.NewBiz("任务不存在")
	}

	if task.RunningState != entity.DbTransferTaskRunStateRunning {
		return errorx.NewBiz("该任务未在执行")
	}
	task.RunningState = entity.DbTransferTaskRunStateStop
	if err = app.UpdateById(ctx, task); err != nil {
		return err
	}

	app.MarkStop(taskId)
	return nil
}

// 迁移表
func (app *dbTransferAppImpl) transferDbTables(ctx context.Context, logId uint64, task *entity.DbTransferTask, srcConn *dbi.DbConn, targetConn *dbi.DbConn, tables []dbi.Table) error {
	tableNames := make([]string, 0)
	tableMap := make(map[string]dbi.Table) // 以表名分组，存放表信息
	for _, table := range tables {
		tableNames = append(tableNames, table.TableName)
		tableMap[table.TableName] = table
	}

	if len(tableNames) == 0 {
		return errorx.NewBiz("没有需要迁移的表")
	}
	srcMeta := srcConn.GetMetaData()
	// 查询源表列信息
	columns, err := srcMeta.GetColumns(tableNames...)
	if err != nil {
		return errorx.NewBiz("获取源表列信息失败")
	}

	// 以表名分组，存放每个表的列信息
	columnMap := make(map[string][]dbi.Column)
	for _, column := range columns {
		columnMap[column.TableName] = append(columnMap[column.TableName], column)
	}

	// 以表名排序
	sortTableNames := collx.MapKeys(columnMap)
	sort.Strings(sortTableNames)

	targetDialect := targetConn.GetDialect()
	srcColumnHelper := srcMeta.GetColumnHelper()
	targetColumnHelper := targetConn.GetMetaData().GetColumnHelper()

	// 分组迁移
	tableGroups := collx.ArraySplit[string](sortTableNames, 2)
	errGroup, _ := errgroup.WithContext(ctx)

	for _, tables := range tableGroups {
		errGroup.Go(func() error {
			for _, tbName := range tables {
				cols := columnMap[tbName]
				targetCols := make([]dbi.Column, 0)
				for _, col := range cols {
					colPtr := &col
					// 源库列转为公共列
					srcColumnHelper.ToCommonColumn(colPtr)
					// 公共列转为目标库列
					targetColumnHelper.ToColumn(colPtr)
					targetCols = append(targetCols, *colPtr)
				}

				// 通过公共列信息生成目标库的建表语句，并执行目标库建表
				app.Log(ctx, logId, fmt.Sprintf("开始创建目标表: 表名：%s", tbName))
				_, err := targetDialect.CreateTable(targetCols, tableMap[tbName], true)
				if err != nil {
					return errorx.NewBiz(fmt.Sprintf("创建目标表失败: 表名：%s, error: %s", tbName, err.Error()))
				}
				app.Log(ctx, logId, fmt.Sprintf("创建目标表成功: 表名：%s", tbName))

				// 迁移数据
				app.Log(ctx, logId, fmt.Sprintf("开始迁移数据: 表名：%s", tbName))
				total, err := app.transferData(ctx, logId, task.Id, tbName, targetCols, srcConn, targetConn)
				if err != nil {
					return errorx.NewBiz(fmt.Sprintf("迁移数据失败: 表名：%s, error: %s", tbName, err.Error()))
				}
				app.Log(ctx, logId, fmt.Sprintf("迁移数据成功: 表名：%s, 数据：%d 条", tbName, total))

				// 有些数据库迁移完数据之后，需要更新表自增序列为当前表最大值
				targetDialect.UpdateSequence(tbName, targetCols)

				// 迁移索引信息
				app.Log(ctx, logId, fmt.Sprintf("开始迁移索引: 表名：%s", tbName))
				err = app.transferIndex(ctx, tableMap[tbName], srcConn, targetDialect)
				if err != nil {
					return errorx.NewBiz(fmt.Sprintf("迁移索引失败: 表名：%s, error: %s", tbName, err.Error()))
				}
				app.Log(ctx, logId, fmt.Sprintf("迁移索引成功: 表名：%s", tbName))
			}

			return nil
		})
	}

	return errGroup.Wait()
}

func (app *dbTransferAppImpl) transferData(ctx context.Context, logId uint64, taskId uint64, tableName string, targetColumns []dbi.Column, srcConn *dbi.DbConn, targetConn *dbi.DbConn) (int, error) {
	result := make([]map[string]any, 0)
	total := 0        // 总条数
	batchSize := 1000 // 每次查询并迁移1000条数据
	var err error
	srcMeta := srcConn.GetMetaData()
	srcConverter := srcMeta.GetDataHelper()
	targetDialect := targetConn.GetDialect()
	logExtraKey := fmt.Sprintf("`%s` 当前已迁移数据量: ", tableName)

	// 游标查询源表数据，并批量插入目标表
	_, err = srcConn.WalkTableRows(context.Background(), tableName, func(row map[string]any, columns []*dbi.QueryColumn) error {
		total++
		rawValue := map[string]any{}
		for _, column := range columns {
			// 某些情况，如oracle，需要转换时间类型的字符串为time类型
			res := srcConverter.ParseData(row[column.Name], srcConverter.GetDataType(column.Type))
			rawValue[column.Name] = res
		}
		result = append(result, rawValue)
		if total%batchSize == 0 {
			err = app.transfer2Target(taskId, targetConn, targetColumns, result, targetDialect, tableName)
			if err != nil {
				logx.ErrorfContext(ctx, "批量插入目标表数据失败: %v", err)
				return err
			}
			result = result[:0]
			app.logApp.SetExtra(logId, logExtraKey, total)
		}
		return nil
	})

	if err != nil {
		return total, err
	}

	// 处理剩余的数据
	if len(result) > 0 {
		err = app.transfer2Target(taskId, targetConn, targetColumns, result, targetDialect, tableName)
		if err != nil {
			logx.ErrorfContext(ctx, "批量插入目标表数据失败，表名：%s error: %v", tableName, err)
			return 0, err
		}
	}
	// 置空当前表数据迁移量进度
	app.logApp.SetExtra(logId, logExtraKey, nil)
	return total, err
}

func (app *dbTransferAppImpl) transfer2Target(taskId uint64, targetConn *dbi.DbConn, targetColumns []dbi.Column, result []map[string]any, targetDialect dbi.Dialect, tbName string) error {
	if !app.IsRunning(taskId) {
		return errorx.NewBiz("迁移终止")
	}

	tx, err := targetConn.Begin()
	if err != nil {
		return err
	}
	targetMeta := targetConn.GetMetaData()

	// 收集字段名
	var columnNames []string
	for _, col := range targetColumns {
		columnNames = append(columnNames, targetMeta.QuoteIdentifier(col.ColumnName))
	}

	dataHelper := targetMeta.GetDataHelper()

	// 从目标库数据中取出源库字段对应的值
	values := make([][]any, 0)
	for _, record := range result {
		rawValue := make([]any, 0)
		for _, tc := range targetColumns {
			columnName := tc.ColumnName
			val := record[targetMeta.RemoveQuote(columnName)]
			if !tc.Nullable {
				// 如果val是文本，则设置为空格字符
				switch val.(type) {
				case string:
					if val == "" {
						val = " "
					}
				}
			}

			if dataHelper.GetDataType(string(tc.DataType)) == dbi.DataTypeBlob {
				decodeBytes, err := hex.DecodeString(val.(string))
				if err == nil {
					val = decodeBytes
				}
			}

			rawValue = append(rawValue, val)
		}
		values = append(values, rawValue)
	}
	// 批量插入
	_, err = targetDialect.BatchInsert(tx, tbName, columnNames, values, -1)

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logx.Errorf("批量插入目标表数据失败: %v", r)
		}
	}()

	_ = tx.Commit()
	return err
}

func (app *dbTransferAppImpl) transferIndex(_ context.Context, tableInfo dbi.Table, srcConn *dbi.DbConn, targetDialect dbi.Dialect) error {
	// 查询源表索引信息
	indexs, err := srcConn.GetMetaData().GetTableIndex(tableInfo.TableName)
	if err != nil {
		logx.Error("获取索引信息失败", err)
		return err
	}
	if len(indexs) == 0 {
		return nil
	}

	// 通过表名、索引信息生成建索引语句，并执行到目标表
	return targetDialect.CreateIndex(tableInfo, indexs)
}

func (d *dbTransferAppImpl) TimerDeleteTransferFile() {
	logx.Debug("开始定时删除迁移文件...")
	scheduler.AddFun("@every 100m", func() {
		dts, err := d.ListByCond(model.NewCond().Eq("mode", entity.DbTransferTaskModeFile).Ge("file_save_days", 1))
		if err != nil {
			logx.Errorf("定时获取数据库迁移至文件任务失败: %s", err.Error())
			return
		}
		for _, dt := range dts {
			needDelFiles, err := d.transferFileApp.ListByCond(model.NewCond().Eq("task_id", dt.Id).Le("create_time", time.Now().AddDate(0, 0, -dt.FileSaveDays)))
			if err != nil {
				logx.Errorf("定时获取迁移文件失败: %s", err.Error())
				continue
			}
			for _, nf := range needDelFiles {
				if err := d.transferFileApp.Delete(context.Background(), nf.Id); err != nil {
					logx.Errorf("定时删除迁移文件失败: %s", err.Error())
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

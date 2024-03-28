package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	sysapp "mayfly-go/internal/sys/application"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"sort"
	"strings"
	"time"
)

type DbTransferTask interface {
	base.App[*entity.DbTransferTask]

	// GetPageList 分页获取数据库实例
	GetPageList(condition *entity.DbTransferTaskQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	Save(ctx context.Context, instanceEntity *entity.DbTransferTask) error

	Delete(ctx context.Context, id uint64) error

	InitJob()

	Run(ctx context.Context, taskId uint64)

	Stop(ctx context.Context, taskId uint64) error
}

type dbTransferAppImpl struct {
	base.AppImpl[*entity.DbTransferTask, repository.DbTransferTask]

	dbApp  Db            `inject:"DbApp"`
	logApp sysapp.Syslog `inject:"SyslogApp"`
}

func (app *dbTransferAppImpl) InjectDbTransferTaskRepo(repo repository.DbTransferTask) {
	app.Repo = repo
}

func (app *dbTransferAppImpl) GetPageList(condition *entity.DbTransferTaskQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.GetRepo().GetTaskList(condition, pageParam, toEntity, orderBy...)
}

func (app *dbTransferAppImpl) Save(ctx context.Context, taskEntity *entity.DbTransferTask) error {
	var err error
	if taskEntity.Id == 0 {
		err = app.Insert(ctx, taskEntity)
	} else {
		err = app.UpdateById(ctx, taskEntity)
	}
	return err
}

func (app *dbTransferAppImpl) Delete(ctx context.Context, id uint64) error {
	if err := app.DeleteById(ctx, id); err != nil {
		return err
	}
	return nil
}

func (app *dbTransferAppImpl) InitJob() {
	// 修改执行状态为待执行
	updateMap := map[string]interface{}{
		"running_state": entity.DbTransferTaskRunStateStop,
	}
	taskParam := new(entity.DbTransferTask)
	taskParam.RunningState = entity.DbTransferTaskRunStateRunning
	_ = gormx.Updates(taskParam, taskParam, updateMap)
}

func (app *dbTransferAppImpl) Run(ctx context.Context, taskId uint64) {
	task, err := app.GetById(new(entity.DbTransferTask), taskId)
	if err != nil {
		return
	}

	start := time.Now()
	logId, err := app.logApp.CreateLog(ctx, &sysapp.CreateLogReq{
		Description: "DBMS-执行数据迁移",
		ReqParam:    collx.Kvs("taskId", task.Id),
		Type:        sysentity.SyslogTypeRunning,
		Resp:        "开始执行数据迁移...",
	})
	if err != nil {
		logx.Errorf("创建DBMS-执行数据迁移日志失败：%v", err)
		return
	}
	defer app.logApp.Flush(logId)

	// 修改状态与关联日志id
	task.LogId = logId
	task.RunningState = entity.DbTransferTaskRunStateRunning
	app.UpdateById(ctx, task)

	// 获取源库连接、目标库连接，判断连接可用性，否则记录日志：xx连接不可用
	// 获取源库表信息
	srcConn, err := app.dbApp.GetDbConn(uint64(task.SrcDbId), task.SrcDbName)
	if err != nil {
		app.EndTransfer(ctx, logId, taskId, "获取源库连接失败", err, nil)
		return
	}
	// 获取目标库表信息
	targetConn, err := app.dbApp.GetDbConn(uint64(task.TargetDbId), task.TargetDbName)
	if err != nil {
		app.EndTransfer(ctx, logId, taskId, "获取目标库连接失败", err, nil)
		return
	}

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

	// 迁移表
	if err = app.transferTables(ctx, logId, task, srcConn, targetConn, tables); err != nil {
		app.EndTransfer(ctx, logId, taskId, "迁移表失败", err, nil)
		return
	}

	app.EndTransfer(ctx, logId, taskId, fmt.Sprintf("执行迁移完成，执行迁移任务[taskId = %d]完成, 耗时：%v", taskId, time.Since(start)), nil, nil)
}

func (app *dbTransferAppImpl) Log(ctx context.Context, logId uint64, msg string) {
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

func (app *dbTransferAppImpl) Stop(ctx context.Context, taskId uint64) error {
	task, err := app.GetById(new(entity.DbTransferTask), taskId)
	if err != nil {
		return errorx.NewBiz("任务不存在")
	}

	if task.RunningState != entity.DbTransferTaskRunStateRunning {
		return errorx.NewBiz("该任务未在执行")
	}
	task.RunningState = entity.DbTransferTaskRunStateStop
	return app.UpdateById(ctx, task)
}

// 迁移表
func (app *dbTransferAppImpl) transferTables(ctx context.Context, logId uint64, task *entity.DbTransferTask, srcConn *dbi.DbConn, targetConn *dbi.DbConn, tables []dbi.Table) error {
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

	for _, tbName := range sortTableNames {
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
		total, err := app.transferData(ctx, task.Id, tbName, targetCols, srcConn, targetConn)
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
}

func (app *dbTransferAppImpl) transferData(ctx context.Context, taskId uint64, tableName string, targetColumns []dbi.Column, srcConn *dbi.DbConn, targetConn *dbi.DbConn) (int, error) {
	result := make([]map[string]any, 0)
	total := 0        // 总条数
	batchSize := 1000 // 每次查询并迁移1000条数据
	var err error
	srcMeta := srcConn.GetMetaData()
	srcConverter := srcMeta.GetDataHelper()
	targetDialect := targetConn.GetDialect()

	// 游标查询源表数据，并批量插入目标表
	err = srcConn.WalkTableRows(context.Background(), tableName, func(row map[string]any, columns []*dbi.QueryColumn) error {
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
	return total, err
}

func (app *dbTransferAppImpl) transfer2Target(taskId uint64, targetConn *dbi.DbConn, targetColumns []dbi.Column, result []map[string]any, targetDialect dbi.Dialect, tbName string) error {
	task, err := app.GetById(new(entity.DbTransferTask), taskId)
	if err != nil {
		return errorx.NewBiz("任务不存在")
	}
	if task.RunningState == entity.DbTransferTaskRunStateStop {
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

	if len(indexs) == 0 {
		return nil
	}

	// 通过表名、索引信息生成建索引语句，并执行到目标表
	return targetDialect.CreateIndex(tableInfo, indexs)
}

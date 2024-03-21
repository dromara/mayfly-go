package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"strings"
)

type DbTransferTask interface {
	base.App[*entity.DbTransferTask]

	// GetPageList 分页获取数据库实例
	GetPageList(condition *entity.DbTransferTaskQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	Save(ctx context.Context, instanceEntity *entity.DbTransferTask) error

	Delete(ctx context.Context, id uint64) error

	InitJob()

	GetTaskLogList(condition *entity.DbTransferLogQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	Run(taskId uint64, end func(msg string, err error))

	Stop(taskId uint64)
}

type dbTransferAppImpl struct {
	base.AppImpl[*entity.DbTransferTask, repository.DbTransferTask]

	dbTransferLogRepo repository.DbTransferLog `inject:"DbTransferLogRepo"`

	dbApp Db `inject:"DbApp"`
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
	taskParam.RunningState = 1
	_ = gormx.Updates(taskParam, taskParam, updateMap)
}

func (app *dbTransferAppImpl) GetTaskLogList(condition *entity.DbTransferLogQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.dbTransferLogRepo.GetTaskLogList(condition, pageParam, toEntity, orderBy...)
}

func (app *dbTransferAppImpl) Run(taskId uint64, end func(msg string, err error)) {
	task, err := app.GetById(new(entity.DbTransferTask), taskId)
	if err != nil {
		return
	}
	// 获取源库连接、目标库连接，判断连接可用性，否则记录日志：xx连接不可用
	// 获取源库表信息
	srcConn, err := app.dbApp.GetDbConn(uint64(task.SrcDbId), task.SrcDbName)
	if err != nil {
		end("获取源库连接失败", err)
		return
	}
	// 获取目标库表信息
	targetConn, err := app.dbApp.GetDbConn(uint64(task.TargetDbId), task.TargetDbName)
	if err != nil {
		end("获取目标库连接失败", err)
		return
	}
	// 查询出源库表信息
	srcDialect := srcConn.GetDialect()
	targetDialect := targetConn.GetDialect()

	var tables []dbi.Table

	if task.CheckedKeys == "all" {
		tables, err = srcConn.GetMetaData().GetTables()
		if err != nil {
			end("获取源表信息失败", err)
			return
		}
	} else {
		tableNames := strings.Split(task.CheckedKeys, ",")
		tables, err = srcConn.GetMetaData().GetTables(tableNames...)
		if err != nil {
			end("获取源表信息失败", err)
			return
		}
	}

	// 迁移表
	app.transferTables(task, srcConn, srcDialect, targetConn, targetDialect, tables, end)

	end(fmt.Sprintf("执行迁移任务完成:[%d]", task.Id), nil)
}

func (app *dbTransferAppImpl) Stop(taskId uint64) {

}

func (app *dbTransferAppImpl) recLog(taskId uint64) {

}

// 迁移表
func (app *dbTransferAppImpl) transferTables(task *entity.DbTransferTask, srcConn *dbi.DbConn, srcDialect dbi.Dialect, targetConn *dbi.DbConn, targetDialect dbi.Dialect, tables []dbi.Table, end func(msg string, err error)) {

	tableNames := make([]string, 0)
	tableMap := make(map[string]dbi.Table) // 以表名分组，存放表信息
	for _, table := range tables {
		tableNames = append(tableNames, table.TableName)
		tableMap[table.TableName] = table
	}

	if len(tableNames) == 0 {
		end("没有需要迁移的表", nil)
		return
	}

	// 查询源表列信息
	columns, err := srcConn.GetMetaData().GetColumns(tableNames...)
	if err != nil {
		end("获取源表列信息失败", err)
		return
	}
	// 以表名分组，存放每个表的列信息
	columnMap := make(map[string][]dbi.Column)
	for _, column := range columns {
		columnMap[column.TableName] = append(columnMap[column.TableName], column)
	}

	ctx := context.Background()

	for tbName, cols := range columnMap {
		targetCols := make([]dbi.Column, 0)
		for _, col := range cols {
			colPtr := &col
			// 源库列转为公共列
			srcDialect.ToCommonColumn(colPtr)
			// 公共列转为目标库列
			targetDialect.ToColumn(colPtr)
			targetCols = append(targetCols, *colPtr)
		}

		// 通过公共列信息生成目标库的建表语句，并执行目标库建表
		logx.Infof("开始创建目标表: 表名：%s", tbName)
		_, err := targetDialect.CreateTable(targetCols, tableMap[tbName], true)
		if err != nil {
			end(fmt.Sprintf("创建目标表失败: 表名：%s, error: %s", tbName, err.Error()), err)
			return
		}
		logx.Infof("创建目标表成功: 表名：%s", tbName)

		// 迁移数据
		logx.Infof("开始迁移数据: 表名：%s", tbName)
		total, err := app.transferData(ctx, tbName, srcConn, srcDialect, targetConn, targetDialect)
		if err != nil {
			end(fmt.Sprintf("迁移数据失败: 表名：%s, error: %s", tbName, err.Error()), err)
			return
		}
		logx.Infof("迁移数据成功: 表名：%s, 数据：%d 条", tbName, total)

		// 有些数据库迁移完数据之后，需要更新表自增序列为当前表最大值
		targetDialect.UpdateSequence(tbName, targetCols)

		// 迁移索引信息
		logx.Infof("开始迁移索引: 表名：%s", tbName)
		err = app.transferIndex(ctx, tableMap[tbName], srcConn, targetDialect)
		if err != nil {
			end(fmt.Sprintf("迁移索引失败: 表名：%s, error: %s", tbName, err.Error()), err)
			return
		}
		logx.Infof("迁移索引成功: 表名：%s", tbName)

		// 记录任务执行日志
	}

	// 修改任务状态
	taskParam := &entity.DbTransferTask{}
	taskParam.Id = task.Id
	taskParam.RunningState = entity.DbTransferTaskRunStateStop

	if err := app.UpdateById(ctx, task); err != nil {
		end("修改任务状态失败", err)
		return
	}
}

func (app *dbTransferAppImpl) transferData(ctx context.Context, tableName string, srcConn *dbi.DbConn, srcDialect dbi.Dialect, targetConn *dbi.DbConn, targetDialect dbi.Dialect) (int, error) {
	result := make([]map[string]any, 0)
	total := 0        // 总条数
	batchSize := 1000 // 每次查询并迁移1000条数据
	var queryColumns []*dbi.QueryColumn
	var err error
	srcMeta := srcConn.GetMetaData()
	srcConverter := srcMeta.GetDataConverter()

	// 游标查询源表数据，并批量插入目标表
	err = srcConn.WalkTableRows(ctx, tableName, func(row map[string]any, columns []*dbi.QueryColumn) error {
		if len(queryColumns) == 0 {

			for _, col := range columns {
				queryColumns = append(queryColumns, &dbi.QueryColumn{
					Name: targetConn.GetMetaData().QuoteIdentifier(col.Name),
					Type: col.Type,
				})
			}

		}
		total++
		rawValue := map[string]any{}
		for _, column := range columns {
			// 某些情况，如oracle，需要转换时间类型的字符串为time类型
			res := srcConverter.ParseData(row[column.Name], srcConverter.GetDataType(column.Type))
			rawValue[column.Name] = res
		}
		result = append(result, rawValue)
		if total%batchSize == 0 {
			err = app.transfer2Target(targetConn, queryColumns, result, targetDialect, tableName)
			if err != nil {
				logx.Error("批量插入目标表数据失败", err)
				return err
			}
			result = result[:0]
		}
		return nil
	})
	// 处理剩余的数据
	if len(result) > 0 {
		err = app.transfer2Target(targetConn, queryColumns, result, targetDialect, tableName)
		if err != nil {
			logx.Error(fmt.Sprintf("批量插入目标表数据失败，表名：%s", tableName), err)
			return 0, err
		}
	}
	return total, err
}

func (app *dbTransferAppImpl) transfer2Target(targetConn *dbi.DbConn, cols []*dbi.QueryColumn, result []map[string]any, targetDialect dbi.Dialect, tbName string) error {
	tx, err := targetConn.Begin()
	if err != nil {
		return err
	}
	// 收集字段名
	var columnNames []string
	for _, col := range cols {
		columnNames = append(columnNames, col.Name)
	}

	// 从目标库数据中取出源库字段对应的值
	values := make([][]any, 0)
	for _, record := range result {
		rawValue := make([]any, 0)
		for _, cn := range columnNames {
			rawValue = append(rawValue, record[targetConn.GetMetaData().RemoveQuote(cn)])
		}
		values = append(values, rawValue)
	}
	// 批量插入
	_, err = targetDialect.BatchInsert(tx, tbName, columnNames, values, -1)

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logx.Error("批量插入目标表数据失败", r)
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

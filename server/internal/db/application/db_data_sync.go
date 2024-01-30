package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/scheduler"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type DataSyncTask interface {
	base.App[*entity.DataSyncTask]

	// GetPageList 分页获取数据库实例
	GetPageList(condition *entity.DataSyncTaskQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	Save(ctx context.Context, instanceEntity *entity.DataSyncTask) error

	Delete(ctx context.Context, id uint64) error

	InitCronJob()

	AddCronJob(taskEntity *entity.DataSyncTask)

	RemoveCronJobById(taskId uint64)

	RunCronJob(id uint64) error

	GetTaskLogList(condition *entity.DataSyncLogQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}

type dataSyncAppImpl struct {
	base.AppImpl[*entity.DataSyncTask, repository.DataSyncTask]

	dbDataSyncLogRepo repository.DataSyncLog `inject:"DbDataSyncLogRepo"`

	dbApp Db `inject:"DbApp"`
}

var (
	dateTimeReg = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`)
)

func (app *dataSyncAppImpl) InjectDbDataSyncTaskRepo(repo repository.DataSyncTask) {
	app.Repo = repo
}

func (app *dataSyncAppImpl) GetPageList(condition *entity.DataSyncTaskQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.GetRepo().GetTaskList(condition, pageParam, toEntity, orderBy...)
}

func (app *dataSyncAppImpl) Save(ctx context.Context, taskEntity *entity.DataSyncTask) error {
	var err error
	if taskEntity.Id == 0 {
		// 新建时生成key
		taskEntity.TaskKey = uuid.New().String()
		err = app.Insert(ctx, taskEntity)
	} else {
		err = app.UpdateById(ctx, taskEntity)
	}

	if err != nil {
		return err
	}

	task, err := app.GetById(new(entity.DataSyncTask), taskEntity.Id)
	if err != nil {
		return err
	}
	app.AddCronJob(task)
	return nil
}

func (app *dataSyncAppImpl) Delete(ctx context.Context, id uint64) error {
	if err := app.DeleteById(ctx, id); err != nil {
		return err
	}
	app.RemoveCronJobById(id)
	return nil
}

func (app *dataSyncAppImpl) AddCronJob(taskEntity *entity.DataSyncTask) {
	key := taskEntity.TaskKey
	// 先移除旧的任务
	scheduler.RemoveByKey(key)

	// 根据状态添加新的任务
	if taskEntity.Status == entity.DataSyncTaskStatusEnable {
		taskId := taskEntity.Id
		scheduler.AddFunByKey(key, taskEntity.TaskCron, func() {
			logx.Infof("开始执行同步任务: %d", taskId)
			if err := app.RunCronJob(taskId); err != nil {
				logx.Errorf("定时执行数据同步任务失败: %s", err.Error())
			}
		})
	}
}

func (app *dataSyncAppImpl) RemoveCronJobById(taskId uint64) {
	task, err := app.GetById(new(entity.DataSyncTask), taskId)
	if err == nil {
		scheduler.RemoveByKey(task.TaskKey)
	}
}

func (app *dataSyncAppImpl) changeRunningState(id uint64, state int8) {
	task := new(entity.DataSyncTask)
	task.Id = id
	task.RunningState = state
	_ = app.UpdateById(context.Background(), task)
}

func (app *dataSyncAppImpl) RunCronJob(id uint64) error {
	// 查询最新的任务信息
	task, err := app.GetById(new(entity.DataSyncTask), id)
	if err != nil {
		return errorx.NewBiz("任务不存在")
	}
	if task.RunningState == entity.DataSyncTaskRunStateRunning {
		return errorx.NewBiz("该任务正在执行中")
	}
	// 开始运行时，修改状态为运行中
	app.changeRunningState(id, entity.DataSyncTaskRunStateRunning)

	logx.Infof("开始执行数据同步任务：%s => %s", task.TaskName, task.TaskKey)

	go func() {
		// 通过占位符格式化sql
		updSql := ""
		orderSql := ""
		if task.UpdFieldVal != "0" && task.UpdFieldVal != "" && task.UpdField != "" {
			srcConn, _ := app.dbApp.GetDbConn(uint64(task.SrcDbId), task.SrcDbName)

			task.UpdFieldVal = strings.Trim(task.UpdFieldVal, " ")
			// 把UpdFieldVal尝试转为int，如果可以转为int，则不添加引号，否则添加引号
			if _, err := strconv.Atoi(task.UpdFieldVal); err != nil {
				updSql = fmt.Sprintf("and %s > '%s'", task.UpdField, task.UpdFieldVal)
			} else {
				updSql = fmt.Sprintf("and %s > %s", task.UpdField, task.UpdFieldVal)
			}

			// 如果是oracle且数据类型是时间类型，则需要加上to_date函数
			if srcConn.Info.Type == dbi.DbTypeOracle {
				// 用正则判断数据类型是时间
				if dateTimeReg.MatchString(task.UpdFieldVal) {
					updSql = fmt.Sprintf("and %s > to_date('%s','yyyy-mm-dd hh24:mi:ss')", task.UpdField, task.UpdFieldVal)
				}
			}
			orderSql = "order by " + task.UpdField + " asc "
		}
		// 组装查询sql
		sql := fmt.Sprintf("select * from (%s) t where 1 = 1 %s %s", task.DataSql, updSql, orderSql)

		log, err := app.doDataSync(sql, task)
		if err != nil {
			log.ErrText = fmt.Sprintf("执行失败: %s", err.Error())
			log.Status = entity.DataSyncTaskStateFail
			app.endRunning(task, log)
		}
	}()

	return nil
}

func (app *dataSyncAppImpl) doDataSync(sql string, task *entity.DataSyncTask) (*entity.DataSyncLog, error) {
	now := time.Now()
	syncLog := &entity.DataSyncLog{
		TaskId:      task.Id,
		CreateTime:  &now,
		DataSqlFull: sql,
		Status:      entity.DataSyncTaskStateRunning,
	}

	// 获取源数据库连接
	srcConn, err := app.dbApp.GetDbConn(uint64(task.SrcDbId), task.SrcDbName)
	if err != nil {
		return syncLog, errorx.NewBiz("连接源数据库失败: %s", err.Error())
	}

	// 获取目标数据库连接
	targetConn, err := app.dbApp.GetDbConn(uint64(task.TargetDbId), task.TargetDbName)
	if err != nil {
		return syncLog, errorx.NewBiz("连接目标数据库失败: %s", err.Error())
	}
	targetDbTx, err := targetConn.Begin()
	if err != nil {
		return syncLog, errorx.NewBiz("开启目标数据库事务失败: %s", err.Error())
	}
	defer func() {
		if r := recover(); r != nil {
			targetDbTx.Rollback()
			err = fmt.Errorf("%v", r)
		}
	}()

	srcDialect := srcConn.GetDialect()

	// task.FieldMap为json数组字符串 [{"src":"id","target":"id"}]，转为map
	var fieldMap []map[string]string
	err = json.Unmarshal([]byte(task.FieldMap), &fieldMap)
	if err != nil {
		return syncLog, errorx.NewBiz("解析字段映射json出错: %s", err.Error())
	}
	var updFieldType dbi.DataType

	// 记录本次同步数据总数
	total := 0
	batchSize := task.PageSize
	result := make([]map[string]any, 0)
	var queryColumns []*dbi.QueryColumn

	err = srcConn.WalkQueryRows(context.Background(), sql, func(row map[string]any, columns []*dbi.QueryColumn) error {
		if len(queryColumns) == 0 {
			queryColumns = columns

			// 遍历columns 取task.UpdField的字段类型
			updFieldType = dbi.DataTypeString
			for _, column := range columns {
				if strings.EqualFold(strings.ToLower(column.Name), strings.ToLower(task.UpdField)) {
					updFieldType = srcDialect.GetDataConverter().GetDataType(column.Type)
					break
				}
			}
		}

		total++
		result = append(result, row)
		if total%batchSize == 0 {
			if err := app.srcData2TargetDb(result, fieldMap, columns, updFieldType, task, srcDialect, targetConn, targetDbTx); err != nil {
				return err
			}

			// 记录当前已同步的数据量
			syncLog.ErrText = fmt.Sprintf("本次任务执行中，已同步：%d条", total)
			syncLog.ResNum = total
			app.saveLog(syncLog)

			result = result[:0]
		}

		return nil
	})

	if err != nil {
		targetDbTx.Rollback()
		return syncLog, err
	}

	// 处理剩余的数据
	if len(result) > 0 {
		if err := app.srcData2TargetDb(result, fieldMap, queryColumns, updFieldType, task, srcDialect, targetConn, targetDbTx); err != nil {
			targetDbTx.Rollback()
			return syncLog, err
		}
	}

	if err := targetDbTx.Commit(); err != nil {
		return syncLog, errorx.NewBiz("数据同步-目标数据库事务提交失败: %s", err.Error())
	}
	logx.Infof("同步任务：[%s]，执行完毕，保存记录成功：[%d]条", task.TaskName, total)

	// 保存执行成功日志
	syncLog.ErrText = fmt.Sprintf("本次任务执行成功，新数据：%d 条", total)
	syncLog.Status = entity.DataSyncTaskStateSuccess
	syncLog.ResNum = total
	app.endRunning(task, syncLog)

	return syncLog, nil
}

func (app *dataSyncAppImpl) srcData2TargetDb(srcRes []map[string]any, fieldMap []map[string]string, columns []*dbi.QueryColumn, updFieldType dbi.DataType, task *entity.DataSyncTask, srcDialect dbi.Dialect, targetDbConn *dbi.DbConn, targetDbTx *sql.Tx) error {

	// 遍历src字段列表，取出字段对应的类型
	var srcColumnTypes = make(map[string]string)
	for _, column := range columns {
		srcColumnTypes[column.Name] = column.Type
	}

	// 遍历res，组装数据
	var data = make([]map[string]any, 0)
	for _, record := range srcRes {
		var rowData = make(map[string]any)
		// 遍历字段映射, target字段的值为src字段取值
		for _, item := range fieldMap {
			srcField := item["src"]
			targetField := item["target"]
			// target字段的值为src字段取值
			rowData[targetField] = record[srcField]
		}

		data = append(data, rowData)
	}
	// 解决字段大小写问题
	updFieldVal := srcRes[len(srcRes)-1][strings.ToUpper(task.UpdField)]
	if updFieldVal == "" || updFieldVal == nil {
		updFieldVal = srcRes[len(srcRes)-1][strings.ToLower(task.UpdField)]
	}

	task.UpdFieldVal = srcDialect.GetDataConverter().FormatData(updFieldVal, updFieldType)

	// 获取目标库字段数组
	targetWrapColumns := make([]string, 0)
	// 获取源库字段数组
	srcColumns := make([]string, 0)
	srcFieldTypes := make(map[string]dbi.DataType)
	for _, item := range fieldMap {
		targetField := item["target"]
		srcField := item["target"]
		srcFieldTypes[srcField] = srcDialect.GetDataConverter().GetDataType(srcColumnTypes[item["src"]])
		targetWrapColumns = append(targetWrapColumns, targetDbConn.Info.Type.QuoteIdentifier(targetField))
		srcColumns = append(srcColumns, srcField)
	}

	// 从目标库数据中取出源库字段对应的值
	values := make([][]any, 0)
	for _, record := range data {
		rawValue := make([]any, 0)
		for _, column := range srcColumns {
			// 某些情况，如oracle，需要转换时间类型的字符串为time类型
			res := srcDialect.GetDataConverter().ParseData(record[column], srcFieldTypes[column])
			rawValue = append(rawValue, res)
		}
		values = append(values, rawValue)
	}

	// 目标数据库执行sql批量插入
	_, err := targetDbConn.GetDialect().BatchInsert(targetDbTx, task.TargetTableName, targetWrapColumns, values)
	if err != nil {
		return err
	}

	// 运行过程中，判断状态是否为已关闭，是则结束运行，否则继续运行
	taskParam, _ := app.GetById(new(entity.DataSyncTask), task.Id)
	if taskParam.RunningState == entity.DataSyncTaskRunStateStop {
		return errorx.NewBiz("该任务已被手动终止")
	}

	return nil
}

func (app *dataSyncAppImpl) endRunning(taskEntity *entity.DataSyncTask, log *entity.DataSyncLog) {
	logx.Info(log.ErrText)

	state := log.Status
	task := new(entity.DataSyncTask)
	task.Id = taskEntity.Id
	task.RecentState = state
	if state == entity.DataSyncTaskStateSuccess {
		task.UpdFieldVal = taskEntity.UpdFieldVal
	}
	task.RunningState = entity.DataSyncTaskRunStateReady
	// 运行失败之后设置任务状态为禁用
	//if state == entity.DataSyncTaskStateFail {
	//	taskEntity.Status = entity.DataSyncTaskStatusDisable
	//	app.RemoveCronJob(taskEntity)
	//}
	_ = app.UpdateById(context.Background(), task)
	// 保存执行日志
	app.saveLog(log)
}

func (app *dataSyncAppImpl) saveLog(log *entity.DataSyncLog) {
	app.dbDataSyncLogRepo.Save(context.Background(), log)
}

func (app *dataSyncAppImpl) InitCronJob() {
	defer func() {
		if err := recover(); err != nil {
			logx.ErrorTrace("数据同步任务初始化失败: %s", err.(error))
		}
	}()

	// 修改执行状态为待执行
	updateMap := map[string]interface{}{
		"running_state": entity.DataSyncTaskRunStateReady,
	}
	taskParam := new(entity.DataSyncTask)
	taskParam.RunningState = 1
	_ = gormx.Updates(taskParam, taskParam, updateMap)

	// 把所有正常任务添加到定时任务中
	pageParam := &model.PageParam{
		PageSize: 100,
		PageNum:  1,
	}
	cond := new(entity.DataSyncTaskQuery)
	cond.Status = entity.DataSyncTaskStatusEnable
	jobs := new([]entity.DataSyncTask)

	pr, _ := app.GetPageList(cond, pageParam, jobs)
	total := pr.Total
	add := 0

	for {
		for _, job := range *jobs {
			app.AddCronJob(&job)
			add++
		}
		if add >= int(total) {
			return
		}

		pageParam.PageNum++
		_, _ = app.GetPageList(cond, pageParam, jobs)
	}
}

func (app *dataSyncAppImpl) GetTaskLogList(condition *entity.DataSyncLogQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.dbDataSyncLogRepo.GetTaskLogList(condition, pageParam, toEntity, orderBy...)
}

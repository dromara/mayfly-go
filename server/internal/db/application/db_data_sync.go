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
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
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

	AddCronJob(ctx context.Context, taskEntity *entity.DataSyncTask)

	RemoveCronJobById(taskId uint64)

	RunCronJob(ctx context.Context, id uint64) error

	GetTaskLogList(condition *entity.DataSyncLogQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}

type dataSyncAppImpl struct {
	base.AppImpl[*entity.DataSyncTask, repository.DataSyncTask]

	dbDataSyncLogRepo repository.DataSyncLog `inject:"DbDataSyncLogRepo"`

	dbApp Db `inject:"DbApp"`
}

var (
	dateTimeReg    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`)
	dateTimeIsoReg = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.*$`)
	whereReg       = regexp.MustCompile(`(?i)where`)
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

	task, err := app.GetById(taskEntity.Id)
	if err != nil {
		return err
	}
	app.AddCronJob(ctx, task)
	return nil
}

func (app *dataSyncAppImpl) Delete(ctx context.Context, id uint64) error {
	if err := app.DeleteById(ctx, id); err != nil {
		return err
	}
	app.RemoveCronJobById(id)
	return nil
}

func (app *dataSyncAppImpl) AddCronJob(ctx context.Context, taskEntity *entity.DataSyncTask) {
	key := taskEntity.TaskKey
	// 先移除旧的任务
	scheduler.RemoveByKey(key)

	// 根据状态添加新的任务
	if taskEntity.Status == entity.DataSyncTaskStatusEnable {
		taskId := taskEntity.Id
		scheduler.AddFunByKey(key, taskEntity.TaskCron, func() {
			logx.Infof("start the data synchronization task: %d", taskId)
			if err := app.RunCronJob(ctx, taskId); err != nil {
				logx.Errorf("the data synchronization task failed to execute at a scheduled time: %s", err.Error())
			}
		})
	}
}

func (app *dataSyncAppImpl) RemoveCronJobById(taskId uint64) {
	task, err := app.GetById(taskId)
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

func (app *dataSyncAppImpl) RunCronJob(ctx context.Context, id uint64) error {
	// 查询最新的任务信息
	task, err := app.GetById(id)
	if err != nil {
		return errorx.NewBiz("task not found")
	}
	if task.RunningState == entity.DataSyncTaskRunStateRunning {
		return errorx.NewBiz("the task is in progress")
	}
	// 开始运行时，修改状态为运行中
	app.changeRunningState(id, entity.DataSyncTaskRunStateRunning)

	logx.InfofContext(ctx, "start the data synchronization task: %s => %s", task.TaskName, task.TaskKey)

	go func() {
		// 通过占位符格式化sql
		updSql := ""
		orderSql := ""
		if task.UpdFieldVal != "0" && task.UpdFieldVal != "" && task.UpdField != "" {
			srcConn, err := app.dbApp.GetDbConn(uint64(task.SrcDbId), task.SrcDbName)
			if err != nil {
				logx.ErrorfContext(ctx, "data source connection unavailable: %s", err.Error())
				return
			}

			task.UpdFieldVal = strings.Trim(task.UpdFieldVal, " ")

			// 判断UpdFieldVal数据类型
			var updFieldValType dbi.DataType
			if _, err = strconv.Atoi(task.UpdFieldVal); err != nil {
				if dateTimeReg.MatchString(task.UpdFieldVal) || dateTimeIsoReg.MatchString(task.UpdFieldVal) {
					updFieldValType = dbi.DataTypeDateTime
				} else {
					updFieldValType = dbi.DataTypeString
				}
			} else {
				updFieldValType = dbi.DataTypeNumber
			}
			wrapUpdFieldVal := srcConn.GetDialect().GetDataHelper().WrapValue(task.UpdFieldVal, updFieldValType)
			updSql = fmt.Sprintf("and %s > %s", task.UpdField, wrapUpdFieldVal)

			orderSql = "order by " + task.UpdField + " asc "
		}
		// 正则判断DataSql是否以where .*结尾，如果是则不添加where 1 = 1
		var where = "where 1=1"
		if whereReg.MatchString(task.DataSql) {
			where = ""
		}

		// 组装查询sql
		sql := fmt.Sprintf("%s %s %s %s", task.DataSql, where, updSql, orderSql)

		log, err := app.doDataSync(ctx, sql, task)
		if err != nil {
			log.ErrText = fmt.Sprintf("execution failure: %s", err.Error())
			logx.ErrorContext(ctx, log.ErrText)
			log.Status = entity.DataSyncTaskStateFail
			app.endRunning(task, log)
		}
	}()

	return nil
}

func (app *dataSyncAppImpl) doDataSync(ctx context.Context, sql string, task *entity.DataSyncTask) (*entity.DataSyncLog, error) {
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
		return syncLog, errorx.NewBiz("failed to connect to the source database: %s", err.Error())
	}

	// 获取目标数据库连接
	targetConn, err := app.dbApp.GetDbConn(uint64(task.TargetDbId), task.TargetDbName)
	if err != nil {
		return syncLog, errorx.NewBiz("failed to connect to the target database: %s", err.Error())
	}
	targetDbTx, err := targetConn.Begin()
	if err != nil {
		return syncLog, errorx.NewBiz("failed to start the target database transaction: %s", err.Error())
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
		return syncLog, errorx.NewBiz("there was an error parsing the field map json: %s", err.Error())
	}
	var updFieldType dbi.DataType

	// 记录本次同步数据总数
	total := 0
	batchSize := task.PageSize
	result := make([]map[string]any, 0)
	var queryColumns []*dbi.QueryColumn

	// 如果有数据库别名，则从UpdField中去掉数据库别名, 如：a.id => id，用于获取字段具体名称
	updFieldName := task.UpdField
	if task.UpdField != "" && strings.Contains(task.UpdField, ".") {
		updFieldName = strings.Split(task.UpdField, ".")[1]
	}

	_, err = srcConn.WalkQueryRows(context.Background(), sql, func(row map[string]any, columns []*dbi.QueryColumn) error {
		if len(queryColumns) == 0 {
			queryColumns = columns

			// 遍历columns 取task.UpdField的字段类型
			updFieldType = dbi.DataTypeString
			for _, column := range columns {
				if strings.EqualFold(column.Name, updFieldName) {
					updFieldType = srcDialect.GetDataHelper().GetDataType(column.Type)
					break
				}
			}
		}

		total++
		result = append(result, row)
		if total%batchSize == 0 {
			if err := app.srcData2TargetDb(result, fieldMap, columns, updFieldType, updFieldName, task, srcDialect, targetConn, targetDbTx); err != nil {
				return err
			}

			// 记录当前已同步的数据量
			syncLog.ErrText = fmt.Sprintf("during the execution of this task, %d has been synchronized", total)
			logx.InfoContext(ctx, syncLog.ErrText)
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
		if err := app.srcData2TargetDb(result, fieldMap, queryColumns, updFieldType, updFieldName, task, srcDialect, targetConn, targetDbTx); err != nil {
			targetDbTx.Rollback()
			return syncLog, err
		}
	}

	// 如果是mssql，暂不手动提交事务，否则报错 mssql: The COMMIT TRANSACTION request has no corresponding BEGIN TRANSACTION.
	if err := targetDbTx.Commit(); err != nil {
		if targetConn.Info.Type != dbi.DbTypeMssql {
			return syncLog, errorx.NewBiz("data synchronization - The target database transaction failed to commit: %s", err.Error())
		}
	}

	logx.InfofContext(ctx, "synchronous task: [%s], finished execution, save records successfully: [%d]", task.TaskName, total)

	// 保存执行成功日志
	syncLog.ErrText = fmt.Sprintf("the synchronous task was executed successfully. New data: %d", total)
	syncLog.Status = entity.DataSyncTaskStateSuccess
	syncLog.ResNum = total
	app.endRunning(task, syncLog)

	return syncLog, nil
}

func (app *dataSyncAppImpl) srcData2TargetDb(srcRes []map[string]any, fieldMap []map[string]string, columns []*dbi.QueryColumn, updFieldType dbi.DataType, updFieldName string, task *entity.DataSyncTask, srcDialect dbi.Dialect, targetDbConn *dbi.DbConn, targetDbTx *sql.Tx) error {

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

	setUpdateFieldVal := func(field string) {
		// 解决字段大小写问题
		updFieldVal := srcRes[len(srcRes)-1][strings.ToUpper(field)]
		if updFieldVal == "" || updFieldVal == nil {
			updFieldVal = srcRes[len(srcRes)-1][strings.ToLower(field)]
		}
		task.UpdFieldVal = srcDialect.GetDataHelper().FormatData(updFieldVal, updFieldType)
	}

	// 如果指定了更新字段，则以更新字段取值
	if task.UpdFieldSrc != "" {
		setUpdateFieldVal(task.UpdFieldSrc)
	} else {
		setUpdateFieldVal(updFieldName)
	}

	// 获取目标库字段数组
	targetWrapColumns := make([]string, 0)
	// 获取源库字段数组
	srcColumns := make([]string, 0)
	srcFieldTypes := make(map[string]dbi.DataType)
	targetDialect := targetDbConn.GetDialect()
	for _, item := range fieldMap {
		targetField := item["target"]
		srcField := item["target"]
		srcFieldTypes[srcField] = srcDialect.GetDataHelper().GetDataType(srcColumnTypes[item["src"]])
		targetWrapColumns = append(targetWrapColumns, targetDialect.QuoteIdentifier(targetField))
		srcColumns = append(srcColumns, srcField)
	}

	// 从目标库数据中取出源库字段对应的值
	values := make([][]any, 0)
	for _, record := range data {
		rawValue := make([]any, 0)
		for _, column := range srcColumns {
			// 某些情况，如oracle，需要转换时间类型的字符串为time类型
			res := srcDialect.GetDataHelper().ParseData(record[column], srcFieldTypes[column])
			rawValue = append(rawValue, res)
		}
		values = append(values, rawValue)
	}

	// 目标数据库执行sql批量插入
	_, err := targetDialect.BatchInsert(targetDbTx, task.TargetTableName, targetWrapColumns, values, task.DuplicateStrategy)
	if err != nil {
		return err
	}

	// 运行过程中，判断状态是否为已关闭，是则结束运行，否则继续运行
	taskParam, _ := app.GetById(task.Id)
	if taskParam.RunningState == entity.DataSyncTaskRunStateStop {
		return errorx.NewBiz("the task has been terminated manually")
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
			logx.ErrorTrace("the data synchronization task failed to initialize: %s", err.(error))
		}
	}()

	// 修改执行中状态为待执行
	_ = app.UpdateByCond(context.TODO(), &entity.DataSyncTask{RunningState: entity.DataSyncTaskRunStateReady}, &entity.DataSyncTask{RunningState: entity.DataSyncTaskRunStateRunning})

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

func (app *dataSyncAppImpl) GetTaskLogList(condition *entity.DataSyncLogQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return app.dbDataSyncLogRepo.GetTaskLogList(condition, pageParam, toEntity, orderBy...)
}

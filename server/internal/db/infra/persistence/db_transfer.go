package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type dbTransferTaskRepoImpl struct {
	base.RepoImpl[*entity.DbTransferTask]
}

var _ repository.DbTransferTask = (*dbTransferTaskRepoImpl)(nil)

func newDbTransferTaskRepo() repository.DbTransferTask {
	return &dbTransferTaskRepoImpl{}
}

// 分页获取数据迁移任务列表
func (d *dbTransferTaskRepoImpl) GetPageList(condition *entity.DbTransferTaskQuery, orderBy ...string) (*model.PageResult[*entity.DbTransferTask], error) {
	qd := model.NewCond().
		Like("task_name", condition.Name).
		Eq("status", condition.Status).
		Eq("cron_enabled", condition.CronEnabled)
	return d.PageByCond(qd, condition.PageParam)
}

type dbTransferCheckpointRepoImpl struct {
	base.RepoImpl[*entity.DbTransferCheckpoint]
}

var _ repository.DbTransferCheckpoint = (*dbTransferCheckpointRepoImpl)(nil)

func newDbTransferCheckpointRepo() repository.DbTransferCheckpoint {
	return &dbTransferCheckpointRepoImpl{}
}

// GetByTaskId 获取指定任务的检查点，不存在返回nil
func (d *dbTransferCheckpointRepoImpl) GetByTaskId(taskId uint64) (*entity.DbTransferCheckpoint, error) {
	list, err := d.SelectByCond(model.NewCond().Eq("task_id", taskId))
	if err != nil || len(list) == 0 {
		return nil, err
	}
	return list[0], nil
}

type dbTransferLogRepoImpl struct {
	base.RepoImpl[*entity.DbTransferLog]
}

var _ repository.DbTransferLog = (*dbTransferLogRepoImpl)(nil)

func newDbTransferLogRepo() repository.DbTransferLog {
	return &dbTransferLogRepoImpl{}
}

// transferLogListColumns 迁移执行日志列表需要查询的列，不含 run_log。
// 运行日志为追加式 text，单次执行可达数百 KB，列表页只需状态与指标，日志内容由调用方按日志 id 单独获取。
var transferLogListColumns = []string{
	"id", "create_time", "task_id", "mode", "purpose", "target_file", "err_text",
	"status", "duration_ms", "total_rows", "table_count",
}

// GetPageList 分页获取指定任务的日志列表
func (d *dbTransferLogRepoImpl) GetPageList(condition *entity.DbTransferLogQuery, orderBy ...string) (*model.PageResult[*entity.DbTransferLog], error) {
	// task_id 用 Eq0：调用方未传 taskId（零值）时按 task_id = 0 过滤返回空集，
	// 用 Eq 会因零值被忽略而不加条件，退化成跨任务的全量查询
	qd := model.NewCond().Eq0("task_id", condition.TaskId).OrderBy(orderBy...)
	return d.PageByCond(qd, condition.PageParam, transferLogListColumns...)
}

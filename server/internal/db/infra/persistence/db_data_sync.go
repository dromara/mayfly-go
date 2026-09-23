package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type dataSyncTaskRepoImpl struct {
	base.RepoImpl[*entity.DataSyncTask]
}

var _ repository.DataSyncTask = (*dataSyncTaskRepoImpl)(nil)

func newDataSyncTaskRepo() repository.DataSyncTask {
	return &dataSyncTaskRepoImpl{}
}

// 分页获取数据同步任务列表
func (d *dataSyncTaskRepoImpl) GetPageList(condition *entity.DataSyncTaskQuery, orderBy ...string) (*model.PageResult[*entity.DataSyncTask], error) {
	qd := model.NewCond().
		Like("task_name", condition.Name).
		Eq("status", condition.Status)
	return d.PageByCond(qd, condition.PageParam)
}

type dataSyncLogRepoImpl struct {
	base.RepoImpl[*entity.DataSyncLog]
}

var _ repository.DataSyncLog = (*dataSyncLogRepoImpl)(nil)

func newDataSyncLogRepo() repository.DataSyncLog {
	return &dataSyncLogRepoImpl{}
}

// syncLogListColumns 同步执行日志列表需要查询的列，不含 run_log 与 data_sql_full。
// 两者均为 text 大字段（运行日志追加式可达数百 KB，完整 SQL 随表规模增长），列表页只需状态与指标，
// 运行日志由调用方按日志 id 单独获取
var syncLogListColumns = []string{
	"id", "create_time", "task_id", "res_num", "err_text", "status",
	"duration_ms", "throughput", "batch_count", "insert_count", "update_count", "delete_count", "skip_count", "schema_changes",
}

// 分页获取同步执行日志列表
func (d *dataSyncLogRepoImpl) GetPageList(condition *entity.DataSyncLogQuery, orderBy ...string) (*model.PageResult[*entity.DataSyncLog], error) {
	// task_id 用 Eq0：调用方未传 taskId（零值）时按 task_id = 0 过滤返回空集，
	// 用 Eq 会因零值被忽略而不加条件，退化成跨任务的全量查询
	qd := model.NewCond().Eq0("task_id", condition.TaskId).OrderBy(orderBy...)
	return d.PageByCond(qd, condition.PageParam, syncLogListColumns...)
}

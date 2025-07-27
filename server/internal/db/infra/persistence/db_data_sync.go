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

func newDataSyncTaskRepo() repository.DataSyncTask {
	return &dataSyncTaskRepoImpl{}
}

// 分页获取数据库信息列表
func (d *dataSyncTaskRepoImpl) GetTaskList(condition *entity.DataSyncTaskQuery, orderBy ...string) (*model.PageResult[*entity.DataSyncTask], error) {
	qd := model.NewCond().
		Like("task_name", condition.Name).
		Eq("status", condition.Status)
	return d.PageByCond(qd, condition.PageParam)
}

type dataSyncLogRepoImpl struct {
	base.RepoImpl[*entity.DataSyncLog]
}

func newDataSyncLogRepo() repository.DataSyncLog {
	return &dataSyncLogRepoImpl{}
}

// 分页获取数据库信息列表
func (d *dataSyncLogRepoImpl) GetTaskLogList(condition *entity.DataSyncLogQuery, orderBy ...string) (*model.PageResult[*entity.DataSyncLog], error) {
	qd := model.NewCond().
		Eq("task_id", condition.TaskId)
	return d.PageByCond(qd, condition.PageParam)
}

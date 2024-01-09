package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type dataSyncTaskRepoImpl struct {
	base.RepoImpl[*entity.DataSyncTask]
}

func newDataSyncTaskRepo() repository.DataSyncTask {
	return &dataSyncTaskRepoImpl{base.RepoImpl[*entity.DataSyncTask]{M: new(entity.DataSyncTask)}}
}

// 分页获取数据库信息列表
func (d *dataSyncTaskRepoImpl) GetTaskList(condition *entity.DataSyncTaskQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := gormx.NewQuery(new(entity.DataSyncTask)).
		Like("task_name", condition.Name).
		Eq("status", condition.Status)
	return gormx.PageQuery(qd, pageParam, toEntity)
}

type dataSyncLogRepoImpl struct {
	base.RepoImpl[*entity.DataSyncLog]
}

// 分页获取数据库信息列表
func (d *dataSyncLogRepoImpl) GetTaskLogList(condition *entity.DataSyncLogQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := gormx.NewQuery(new(entity.DataSyncLog)).
		Eq("task_id", condition.TaskId)
	return gormx.PageQuery(qd, pageParam, toEntity)
}

func newDataSyncLogRepo() repository.DataSyncLog {
	return &dataSyncLogRepoImpl{base.RepoImpl[*entity.DataSyncLog]{M: new(entity.DataSyncLog)}}
}

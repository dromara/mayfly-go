package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type DataSyncTask interface {
	base.Repo[*entity.DataSyncTask]

	// 分页获取数据库实例信息列表
	GetTaskList(condition *entity.DataSyncTaskQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}

type DataSyncLog interface {
	base.Repo[*entity.DataSyncLog]

	// 分页获取数据库实例信息列表
	GetTaskLogList(condition *entity.DataSyncLogQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}

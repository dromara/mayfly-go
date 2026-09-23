package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type DataSyncTask interface {
	base.Repo[*entity.DataSyncTask]

	// 分页获取数据同步任务列表
	GetPageList(condition *entity.DataSyncTaskQuery, orderBy ...string) (*model.PageResult[*entity.DataSyncTask], error)
}

type DataSyncLog interface {
	base.Repo[*entity.DataSyncLog]

	// 分页获取同步执行日志列表
	GetPageList(condition *entity.DataSyncLogQuery, orderBy ...string) (*model.PageResult[*entity.DataSyncLog], error)
}

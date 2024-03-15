package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type DbTransferTask interface {
	base.Repo[*entity.DbTransferTask]

	// 分页获取数据库实例信息列表
	GetTaskList(condition *entity.DbTransferTaskQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}

type DbTransferLog interface {
	base.Repo[*entity.DbTransferLog]

	// 分页获取数据库实例信息列表
	GetTaskLogList(condition *entity.DbTransferLogQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}

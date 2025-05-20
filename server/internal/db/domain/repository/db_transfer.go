package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type DbTransferTask interface {
	base.Repo[*entity.DbTransferTask]

	// 分页获取数据库实例信息列表
	GetTaskList(condition *entity.DbTransferTaskQuery, orderBy ...string) (*model.PageResult[*entity.DbTransferTask], error)
}

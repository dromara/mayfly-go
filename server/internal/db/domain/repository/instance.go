package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type Instance interface {
	base.Repo[*entity.DbInstance]

	// 分页获取数据库实例信息列表
	GetInstanceList(condition *entity.InstanceQuery, orderBy ...string) (*model.PageResult[*entity.DbInstance], error)
}

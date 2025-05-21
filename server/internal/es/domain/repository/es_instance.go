package repository

import (
	"mayfly-go/internal/es/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type EsInstance interface {
	base.Repo[*entity.EsInstance]

	// 分页获取数据库实例信息列表
	GetInstanceList(condition *entity.InstanceQuery, orderBy ...string) (*model.PageResult[*entity.EsInstance], error)
}

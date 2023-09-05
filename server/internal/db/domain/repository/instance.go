package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/model"
)

type Instance interface {
	// 分页获取数据库实例信息列表
	GetInstanceList(condition *entity.InstanceQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any]

	Count(condition *entity.InstanceQuery) int64

	// 根据条件获取实例信息
	GetInstance(condition *entity.Instance, cols ...string) error

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.Instance

	Insert(db *entity.Instance)

	Update(db *entity.Instance)

	Delete(id uint64)
}

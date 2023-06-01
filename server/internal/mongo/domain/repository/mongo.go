package repository

import (
	"mayfly-go/internal/mongo/domain/entity"
	"mayfly-go/pkg/model"
)

type Mongo interface {
	// 分页获取列表
	GetList(condition *entity.MongoQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult

	Count(condition *entity.MongoQuery) int64

	// 根据条件获取
	Get(condition *entity.Mongo, cols ...string) error

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.Mongo

	Insert(db *entity.Mongo)

	Update(db *entity.Mongo)

	Delete(id uint64)
}

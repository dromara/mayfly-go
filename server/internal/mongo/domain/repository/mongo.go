package repository

import (
	"mayfly-go/internal/mongo/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type Mongo interface {
	base.Repo[*entity.Mongo]

	// 分页获取列表
	GetList(condition *entity.MongoQuery, orderBy ...string) (*model.PageResult[*entity.Mongo], error)
}

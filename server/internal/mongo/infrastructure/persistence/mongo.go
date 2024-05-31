package persistence

import (
	"mayfly-go/internal/mongo/domain/entity"
	"mayfly-go/internal/mongo/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type mongoRepoImpl struct {
	base.RepoImpl[*entity.Mongo]
}

func newMongoRepo() repository.Mongo {
	return &mongoRepoImpl{base.RepoImpl[*entity.Mongo]{M: new(entity.Mongo)}}
}

// 分页获取数据库信息列表
func (d *mongoRepoImpl) GetList(condition *entity.MongoQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := model.NewCond().
		Like("name", condition.Name).
		Eq("code", condition.Code).
		In("code", condition.Codes)
	return d.PageByCondToAny(qd, pageParam, toEntity)
}

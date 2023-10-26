package persistence

import (
	"mayfly-go/internal/mongo/domain/entity"
	"mayfly-go/internal/mongo/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
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
	qd := gormx.NewQuery(new(entity.Mongo)).
		Like("name", condition.Name).
		In("tag_id", condition.TagIds).
		RLike("tag_path", condition.TagPath).
		OrderByAsc("tag_path")
	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (d *mongoRepoImpl) Count(condition *entity.MongoQuery) int64 {
	where := make(map[string]any)
	if len(condition.TagIds) > 0 {
		where["tag_id"] = condition.TagIds
	}
	return gormx.CountByCond(new(entity.Mongo), where)
}

package persistence

import (
	"mayfly-go/internal/mongo/domain/entity"
	"mayfly-go/internal/mongo/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type mongoRepoImpl struct{}

func newMongoRepo() repository.Mongo {
	return new(mongoRepoImpl)
}

// 分页获取数据库信息列表
func (d *mongoRepoImpl) GetList(condition *entity.MongoQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any] {
	qd := gormx.NewQuery(new(entity.Mongo)).
		Like("name", condition.Name).
		In("tag_id", condition.TagIds).
		RLike("tag_path", condition.TagPathLike).
		OrderByAsc("tag_path")
	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (d *mongoRepoImpl) Count(condition *entity.MongoQuery) int64 {
	where := make(map[string]any)
	if len(condition.TagIds) > 0 {
		where["tag_id"] = condition.TagIds
	}
	if condition.TagId != 0 {
		where["tag_id"] = condition.TagId
	}
	return gormx.CountByCond(new(entity.Mongo), where)
}

// 根据条件获取
func (d *mongoRepoImpl) Get(condition *entity.Mongo, cols ...string) error {
	return gormx.GetBy(condition, cols...)
}

// 根据id获取
func (d *mongoRepoImpl) GetById(id uint64, cols ...string) *entity.Mongo {
	db := new(entity.Mongo)
	if err := gormx.GetById(db, id, cols...); err != nil {
		return nil

	}
	return db
}

func (d *mongoRepoImpl) Insert(db *entity.Mongo) {
	biz.ErrIsNil(gormx.Insert(db), "新增mongo信息失败")
}

func (d *mongoRepoImpl) Update(db *entity.Mongo) {
	biz.ErrIsNil(gormx.UpdateById(db), "更新mongo信息失败")
}

func (d *mongoRepoImpl) Delete(id uint64) {
	gormx.DeleteById(new(entity.Mongo), id)
}

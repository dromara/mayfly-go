package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type dbRepoImpl struct{}

func newDbRepo() repository.Db {
	return new(dbRepoImpl)
}

// 分页获取数据库信息列表
func (d *dbRepoImpl) GetDbList(condition *entity.DbQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any] {
	qd := gormx.NewQueryWithTableName("t_db db").
		Select("db.*, inst.name instance_name, inst.type instance_type").
		Joins("JOIN t_db_instance inst ON db.instance_id = inst.id").
		Eq("db.instance_id", condition.InstanceId).
		Like("db.database", condition.Database).
		In("db.tag_id", condition.TagIds).
		RLike("db.tag_path", condition.TagPath).
		Eq0("db."+model.DeletedColumn, model.ModelUndeleted).
		Eq0("inst."+model.DeletedColumn, model.ModelUndeleted).
		OrderByAsc("db.tag_path")

	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (d *dbRepoImpl) Count(condition *entity.DbQuery) int64 {
	where := make(map[string]any)
	if len(condition.TagIds) > 0 {
		where["tag_id"] = condition.TagIds
	}
	if condition.InstanceId > 0 {
		where["instance_id"] = condition.InstanceId
	}
	return gormx.CountByCond(new(entity.Db), where)
}

// 根据条件获取账号信息
func (d *dbRepoImpl) GetDb(condition *entity.Db, cols ...string) error {
	return gormx.GetBy(condition, cols...)
}

// 根据id获取
func (d *dbRepoImpl) GetById(id uint64, cols ...string) *entity.Db {
	db := new(entity.Db)
	if err := gormx.GetById(db, id, cols...); err != nil {
		return nil

	}
	return db
}

func (d *dbRepoImpl) Insert(db *entity.Db) {
	biz.ErrIsNil(gormx.Insert(db), "新增数据库信息失败")
}

func (d *dbRepoImpl) Update(db *entity.Db) {
	biz.ErrIsNil(gormx.UpdateById(db), "更新数据库信息失败")
}

func (d *dbRepoImpl) Delete(id uint64) {
	gormx.DeleteById(new(entity.Db), id)
}

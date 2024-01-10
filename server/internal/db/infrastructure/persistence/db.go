package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type dbRepoImpl struct {
	base.RepoImpl[*entity.Db]
}

func newDbRepo() repository.Db {
	return &dbRepoImpl{base.RepoImpl[*entity.Db]{M: new(entity.Db)}}
}

// 分页获取数据库信息列表
func (d *dbRepoImpl) GetDbList(condition *entity.DbQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := gormx.NewQueryWithTableName("t_db db").
		Select("db.*, inst.name instance_name, inst.type instance_type, inst.host, inst.port, inst.username ").
		Joins("JOIN t_db_instance inst ON db.instance_id = inst.id").
		Eq("db.instance_id", condition.InstanceId).
		Eq("db.id", condition.Id).
		Like("db.database", condition.Database).
		In("db.code", condition.Codes).
		Eq0("db."+model.DeletedColumn, model.ModelUndeleted).
		Eq0("inst."+model.DeletedColumn, model.ModelUndeleted)

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
	return d.CountByCond(where)
}

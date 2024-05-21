package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
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
	pd := model.NewCond().Eq("instance_id", condition.InstanceId).In("code", condition.Codes)
	return d.PageByCondToAny(pd, pageParam, toEntity)
}

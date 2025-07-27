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
	return &dbRepoImpl{}
}

// 分页获取数据库信息列表
func (d *dbRepoImpl) GetDbList(condition *entity.DbQuery, orderBy ...string) (*model.PageResult[*entity.DbListPO], error) {
	pd := model.NewCond().Eq("instance_id", condition.InstanceId).In("code", condition.Codes).Eq("id", condition.Id)
	list := []*entity.DbListPO{}
	return gormx.PageByCond(d.GetModel(), pd, condition.PageParam, list)
}

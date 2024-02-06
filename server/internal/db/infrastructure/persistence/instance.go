package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type instanceRepoImpl struct {
	base.RepoImpl[*entity.DbInstance]
}

func NewInstanceRepo() repository.Instance {
	return &instanceRepoImpl{base.RepoImpl[*entity.DbInstance]{M: new(entity.DbInstance)}}
}

// 分页获取数据库信息列表
func (d *instanceRepoImpl) GetInstanceList(condition *entity.InstanceQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := gormx.NewQuery(new(entity.DbInstance)).
		Eq("id", condition.Id).
		Eq("host", condition.Host).
		Like("name", condition.Name)
	return gormx.PageQuery(qd, pageParam, toEntity)
}

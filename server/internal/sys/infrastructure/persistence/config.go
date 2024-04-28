package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type configRepoImpl struct {
	base.RepoImpl[*entity.Config]
}

func newConfigRepo() repository.Config {
	return &configRepoImpl{base.RepoImpl[*entity.Config]{M: new(entity.Config)}}
}

func (m *configRepoImpl) GetPageList(condition *entity.Config, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := model.NewCond().
		Like("`key`", condition.Key).
		And("permission = 'all' OR permission LIKE ?", "%"+condition.Permission+",%").
		OrderBy(orderBy...)
	return m.PageByCond(qd, pageParam, toEntity)
}

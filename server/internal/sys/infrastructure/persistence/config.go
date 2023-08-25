package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type configRepoImpl struct{}

func newConfigRepo() repository.Config {
	return new(configRepoImpl)
}

func (m *configRepoImpl) GetPageList(condition *entity.Config, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any] {
	qd := gormx.NewQuery(condition).
		Eq("key", condition.Key).
		And("permission = 'all' OR permission LIKE ?", "%"+condition.Permission+",%").
		WithOrderBy(orderBy...)
	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (m *configRepoImpl) Insert(config *entity.Config) {
	biz.ErrIsNil(gormx.Insert(config), "新增系统配置失败")
}

func (m *configRepoImpl) Update(config *entity.Config) {
	biz.ErrIsNil(gormx.UpdateById(config), "更新系统配置失败")
}

func (m *configRepoImpl) GetConfig(condition *entity.Config, cols ...string) error {
	return gormx.GetBy(condition, cols...)
}

func (r *configRepoImpl) GetByCondition(condition *entity.Config, cols ...string) error {
	return gormx.GetBy(condition, cols...)
}

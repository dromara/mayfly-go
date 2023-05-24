package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type configRepoImpl struct{}

func newConfigRepo() repository.Config {
	return new(configRepoImpl)
}

func (m *configRepoImpl) GetPageList(condition *entity.Config, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return model.GetPage(pageParam, condition, condition, toEntity)
}

func (m *configRepoImpl) Insert(config *entity.Config) {
	biz.ErrIsNil(model.Insert(config), "新增系统配置失败")
}

func (m *configRepoImpl) Update(config *entity.Config) {
	biz.ErrIsNil(model.UpdateById(config), "更新系统配置失败")
}

func (m *configRepoImpl) GetConfig(condition *entity.Config, cols ...string) error {
	return model.GetBy(condition, cols...)
}

func (r *configRepoImpl) GetByCondition(condition *entity.Config, cols ...string) error {
	return model.GetBy(condition, cols...)
}

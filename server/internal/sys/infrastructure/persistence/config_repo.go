package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type configRepo struct{}

var ConfigDao repository.Config = &configRepo{}

func (m *configRepo) GetPageList(condition *entity.Config, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return model.GetPage(pageParam, condition, toEntity)
}

func (m *configRepo) Insert(config *entity.Config) {
	biz.ErrIsNil(model.Insert(config), "新增系统配置失败")
}

func (m *configRepo) Update(config *entity.Config) {
	biz.ErrIsNil(model.UpdateById(config), "更新系统配置失败")
}

func (m *configRepo) GetConfig(condition *entity.Config, cols ...string) error {
	return model.GetBy(condition, cols...)
}

func (r *configRepo) GetByCondition(condition *entity.Config, cols ...string) error {
	return model.GetBy(condition, cols...)
}

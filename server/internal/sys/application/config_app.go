package application

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/internal/sys/infrastructure/persistence"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
)

type Config interface {
	GetPageList(condition *entity.Config, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	Save(config *entity.Config)

	// 获取指定key的配置信息, 不会返回nil, 若不存在则值都默认值即空字符串
	GetConfig(key string) *entity.Config
}

type configAppImpl struct {
	configRepo repository.Config
}

var ConfigApp Config = &configAppImpl{
	configRepo: persistence.ConfigDao,
}

func (a *configAppImpl) GetPageList(condition *entity.Config, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return a.configRepo.GetPageList(condition, pageParam, toEntity)
}

func (a *configAppImpl) Save(config *entity.Config) {
	if config.Id == 0 {
		a.configRepo.Insert(config)
	} else {
		a.configRepo.Update(config)
	}
}

func (a *configAppImpl) GetConfig(key string) *entity.Config {
	config := &entity.Config{Key: key}
	if err := a.configRepo.GetConfig(config, "Id", "Key", "Value"); err != nil {
		global.Log.Warnf("不存在key = [%s] 的系统配置", key)
	}
	return config
}

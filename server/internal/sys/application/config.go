package application

import (
	"context"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"strings"
)

const SysConfigKeyPrefix = "mayfly:sys:config:"

type Config interface {
	base.App[*entity.Config]

	GetPageList(condition *entity.Config, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.Config], error)

	Save(ctx context.Context, config *entity.Config) error

	// GetConfig 获取指定key的配置信息, 不会返回nil, 若不存在则值都默认值即空字符串
	GetConfig(key string) *entity.Config
}

var _ (Config) = (*configAppImpl)(nil)

type configAppImpl struct {
	base.AppImpl[*entity.Config, repository.Config]
}

func (a *configAppImpl) GetPageList(condition *entity.Config, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.Config], error) {
	return a.GetRepo().GetPageList(condition, pageParam)
}

func (a *configAppImpl) Save(ctx context.Context, config *entity.Config) error {
	if config.Id == 0 {
		return a.Insert(ctx, config)
	}

	oldConfig := a.GetConfig(config.Key)
	if oldConfig.Permission != "all" && !strings.Contains(oldConfig.Permission, config.Modifier) {
		return errorx.NewBiz("You do not have permission to modify the configuration")
	}

	if err := a.UpdateById(ctx, config); err != nil {
		return err
	}

	cache.Del(SysConfigKeyPrefix + config.Key)
	return nil
}

func (a *configAppImpl) GetConfig(key string) *entity.Config {
	config := &entity.Config{Key: key}
	// 优先从缓存中获取
	if exist := cache.Get(SysConfigKeyPrefix+key, &config); exist {
		return config
	}

	if err := a.GetByCond(model.NewModelCond(config).Columns("Id", "Key", "Value", "Permission")); err != nil {
		logx.Warnf("There is no system configuration with key = [%s]", key)
	} else {
		cache.Set(SysConfigKeyPrefix+key, config, -1)
	}
	return config
}

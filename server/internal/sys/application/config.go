package application

import (
	"context"
	"encoding/json"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/jsonx"
	"strings"
)

const SysConfigKeyPrefix = "mayfly:sys:config:"

type Config interface {
	base.App[*entity.Config]

	GetPageList(condition *entity.Config, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	Save(ctx context.Context, config *entity.Config) error

	// GetConfig 获取指定key的配置信息, 不会返回nil, 若不存在则值都默认值即空字符串
	GetConfig(key string) *entity.Config
}

type configAppImpl struct {
	base.AppImpl[*entity.Config, repository.Config]
}

func (a *configAppImpl) InjectConfigRepo(repo repository.Config) {
	a.Repo = repo
}

func (a *configAppImpl) GetPageList(condition *entity.Config, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return a.GetRepo().GetPageList(condition, pageParam, toEntity)
}

func (a *configAppImpl) Save(ctx context.Context, config *entity.Config) error {
	if config.Id == 0 {
		return a.Insert(ctx, config)
	}

	oldConfig := a.GetConfig(config.Key)
	if oldConfig.Permission != "all" && !strings.Contains(oldConfig.Permission, config.Modifier) {
		return errorx.NewBiz("您无权修改该配置")
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
	cacheStr := cache.GetStr(SysConfigKeyPrefix + key)
	if cacheStr != "" {
		json.Unmarshal([]byte(cacheStr), &config)
		return config
	}

	if err := a.GetBy(config, "Id", "Key", "Value", "Permission"); err != nil {
		logx.Warnf("不存在key = [%s] 的系统配置", key)
	} else {
		cache.SetStr(SysConfigKeyPrefix+key, jsonx.ToStr(config), -1)
	}
	return config
}

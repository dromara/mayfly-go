package api

import (
	"mayfly-go/internal/sys/api/form"
	"mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type Config struct {
	ConfigApp application.Config `inject:""`
}

func (c *Config) Configs(rc *req.Ctx) {
	condition := &entity.Config{Key: rc.Query("key")}
	condition.Permission = rc.GetLoginAccount().Username
	res, err := c.ConfigApp.GetPageList(condition, rc.GetPageParam(), new([]entity.Config))
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (c *Config) GetConfigValueByKey(rc *req.Ctx) {
	key := rc.Query("key")
	biz.NotEmpty(key, "key cannot be empty")

	config := c.ConfigApp.GetConfig(key)
	// 判断是否为公开配置
	if config.Permission != "all" {
		rc.ResData = ""
		return
	}

	rc.ResData = config.Value
}

func (c *Config) SaveConfig(rc *req.Ctx) {
	form := &form.ConfigForm{}
	config := req.BindJsonAndCopyTo(rc, form, new(entity.Config))
	rc.ReqParam = form
	biz.ErrIsNil(c.ConfigApp.Save(rc.MetaCtx, config))
}

// GetServerConfig 获取当前系统启动配置
func (c *Config) GetServerConfig(rc *req.Ctx) {
	conf := config.Conf
	rc.ResData = collx.Kvs("i18n", conf.Server.Lang)
}

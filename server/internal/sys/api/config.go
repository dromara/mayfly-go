package api

import (
	"mayfly-go/internal/sys/api/form"
	"mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
)

type Config struct {
	ConfigApp application.Config
}

func (c *Config) Configs(rc *req.Ctx) {
	g := rc.GinCtx
	condition := &entity.Config{Key: g.Query("key")}
	rc.ResData = c.ConfigApp.GetPageList(condition, ginx.GetPageParam(g), new([]entity.Config))
}

func (c *Config) GetConfigValueByKey(rc *req.Ctx) {
	key := rc.GinCtx.Query("key")
	biz.NotEmpty(key, "key不能为空")
	rc.ResData = c.ConfigApp.GetConfig(key).Value
}

func (c *Config) GetConfigValueByKeyWithNoToken(keys []string) func(rc *req.Ctx) {
	keyMap := make(map[string]struct{})
	for _, key := range keys {
		keyMap[key] = struct{}{}
	}
	return func(rc *req.Ctx) {
		key := rc.GinCtx.Query("key")
		biz.NotEmpty(key, "key不能为空")
		if _, ok := keyMap[key]; !ok {
			biz.ErrIsNil(nil, "无权限获取该配置信息")
		}
		rc.ResData = c.ConfigApp.GetConfig(key).Value
	}
}

func (c *Config) SaveConfig(rc *req.Ctx) {
	form := &form.ConfigForm{}
	config := ginx.BindJsonAndCopyTo(rc.GinCtx, form, new(entity.Config))
	rc.ReqParam = form
	config.SetBaseInfo(rc.LoginAccount)
	c.ConfigApp.Save(config)
}

// 获取oauth2登录配置信息，因为有些字段是敏感字段，故单独使用接口获取
func (c *Config) Oauth2Config(rc *req.Ctx) {
	oauth2LoginConfig := c.ConfigApp.GetConfig(entity.ConfigKeyOauth2Login).ToOauth2Login()
	rc.ResData = map[string]any{
		"enable": oauth2LoginConfig.Enable,
		"name":   oauth2LoginConfig.Name,
	}
}

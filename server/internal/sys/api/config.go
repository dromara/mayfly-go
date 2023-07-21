package api

import (
	"encoding/json"
	"mayfly-go/internal/sys/api/form"
	"mayfly-go/internal/sys/api/vo"
	"mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/global"
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

// AuthConfig auth相关配置
func (c *Config) AuthConfig(rc *req.Ctx) {
	resp := &vo.OAuth2EnableVO{}
	config := c.ConfigApp.GetConfig(AuthOAuth2Key)
	oauth2 := &vo.OAuth2VO{}
	if config.Value != "" {
		if err := json.Unmarshal([]byte(config.Value), oauth2); err != nil {
			global.Log.Warnf("解析自定义oauth2配置失败，err：%s", err.Error())
			biz.ErrIsNil(err, "解析自定义oauth2配置失败")
		} else if oauth2.ClientID != "" {
			resp.OAuth2 = true
		}
	}
	rc.ResData = resp
}

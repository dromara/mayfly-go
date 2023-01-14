package api

import (
	"mayfly-go/internal/sys/api/form"
	"mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils"
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

func (c *Config) SaveConfig(rc *req.Ctx) {
	g := rc.GinCtx
	form := &form.ConfigForm{}
	ginx.BindJsonAndValid(g, form)
	rc.ReqParam = form

	config := new(entity.Config)
	utils.Copy(config, form)
	config.SetBaseInfo(rc.LoginAccount)
	c.ConfigApp.Save(config)
}

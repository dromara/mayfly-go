package api

import (
	"mayfly-go/internal/sys/api/form"
	"mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

type Config struct {
	ConfigApp application.Config `inject:""`
}

func (c *Config) Configs(rc *req.Ctx) {
	condition := &entity.Config{Key: rc.F.Query("key")}
	condition.Permission = rc.GetLoginAccount().Username
	res, err := c.ConfigApp.GetPageList(condition, rc.F.GetPageParam(), new([]entity.Config))
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (c *Config) GetConfigValueByKey(rc *req.Ctx) {
	key := rc.F.Query("key")
	biz.NotEmpty(key, "key不能为空")

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

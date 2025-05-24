package api

import (
	"mayfly-go/internal/pkg/config"
	"mayfly-go/internal/sys/api/form"
	"mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type Config struct {
	configApp application.Config `inject:"T"`
}

func (c *Config) ReqConfs() *req.Confs {
	baseP := req.NewPermission("config:base")

	reqs := [...]*req.Conf{
		req.NewGet("", c.Configs).RequiredPermission(baseP),

		req.NewGet("/server", c.GetServerConfig).DontNeedToken(),

		// 获取指定配置key对应的值
		req.NewGet("/value", c.GetConfigValueByKey).DontNeedToken(),

		req.NewPost("", c.SaveConfig).Log(req.NewLogSaveI(imsg.LogSaveSysConfig)).RequiredPermissionCode("config:save"),
	}

	return req.NewConfs("sys/configs", reqs[:]...)
}

func (c *Config) Configs(rc *req.Ctx) {
	condition := &entity.Config{Key: rc.Query("key")}
	condition.Permission = rc.GetLoginAccount().Username
	res, err := c.configApp.GetPageList(condition, rc.GetPageParam())
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (c *Config) GetConfigValueByKey(rc *req.Ctx) {
	key := rc.Query("key")
	biz.NotEmpty(key, "key cannot be empty")

	config := c.configApp.GetConfig(key)
	// 判断是否为公开配置
	if config.Permission != "all" {
		rc.ResData = ""
		return
	}

	rc.ResData = config.Value
}

func (c *Config) SaveConfig(rc *req.Ctx) {
	form, config := req.BindJsonAndCopyTo[*form.ConfigForm, *entity.Config](rc)
	rc.ReqParam = form
	biz.ErrIsNil(c.configApp.Save(rc.MetaCtx, config))
}

// GetServerConfig 获取当前系统启动配置
func (c *Config) GetServerConfig(rc *req.Ctx) {
	conf := config.Conf
	rc.ResData = collx.Kvs("i18n", conf.Server.Lang)
}

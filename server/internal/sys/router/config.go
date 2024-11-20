package router

import (
	"mayfly-go/internal/sys/api"
	"mayfly-go/internal/sys/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitSysConfigRouter(router *gin.RouterGroup) {
	configG := router.Group("sys/configs")
	r := new(api.Config)
	biz.ErrIsNil(ioc.Inject(r))

	baseP := req.NewPermission("config:base")

	reqs := [...]*req.Conf{
		req.NewGet("", r.Configs).RequiredPermission(baseP),

		req.NewGet("/server", r.GetServerConfig).DontNeedToken(),

		// 获取指定配置key对应的值
		req.NewGet("/value", r.GetConfigValueByKey).DontNeedToken(),

		req.NewPost("", r.SaveConfig).Log(req.NewLogSaveI(imsg.LogSaveSysConfig)).RequiredPermissionCode("config:save"),
	}

	req.BatchSetGroup(configG, reqs[:])
}

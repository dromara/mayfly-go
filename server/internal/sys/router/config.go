package router

import (
	"mayfly-go/internal/sys/api"
	"mayfly-go/internal/sys/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitSysConfigRouter(router *gin.RouterGroup) {
	r := &api.Config{ConfigApp: application.GetConfigApp()}
	db := router.Group("sys/configs")
	{
		db.GET("", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(r.Configs)
		})

		db.GET("/value", func(c *gin.Context) {
			req.NewCtxWithGin(c).DontNeedToken().Handle(r.GetConfigValueByKey)
		})

		saveConfig := req.NewLogInfo("保存系统配置信息").WithSave(true)
		saveConfigP := req.NewPermission("config:base")
		db.POST("", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(saveConfig).
				WithRequiredPermission(saveConfigP).
				Handle(r.SaveConfig)
		})
	}
}

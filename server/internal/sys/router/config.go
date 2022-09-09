package router

import (
	"mayfly-go/internal/sys/api"
	"mayfly-go/internal/sys/application"
	"mayfly-go/pkg/ctx"

	"github.com/gin-gonic/gin"
)

func InitSysConfigRouter(router *gin.RouterGroup) {
	r := &api.Config{ConfigApp: application.GetConfigApp()}
	db := router.Group("sys/configs")
	{
		db.GET("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(r.Configs)
		})

		db.GET("/value", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithNeedToken(false).Handle(r.GetConfigValueByKey)
		})

		saveConfig := ctx.NewLogInfo("保存系统配置信息").WithSave(true)
		db.POST("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(saveConfig).
				Handle(r.SaveConfig)
		})
	}
}

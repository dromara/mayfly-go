package router

import (
	"mayfly-go/internal/tag/api"
	"mayfly-go/internal/tag/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitTagTreeRouter(router *gin.RouterGroup) {
	m := &api.TagTree{
		TagTreeApp: application.GetTagTreeApp(),
	}

	project := router.Group("/tag-trees")
	{
		// 获取标签树列表
		project.GET("", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(m.GetTagTree)
		})

		// 根据条件获取标签
		project.GET("query", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(m.ListByQuery)
		})

		// 获取登录账号拥有的标签信息
		project.GET("account-has", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(m.GetAccountTags)
		})

		saveProjectTreeLog := req.NewLogInfo("标签树-保存信息").WithSave(true)
		savePP := req.NewPermission("tag:save")
		// 保存项目树下的环境信息
		project.POST("", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(saveProjectTreeLog).
				WithRequiredPermission(savePP).
				Handle(m.SaveTagTree)
		})

		delProjectLog := req.NewLogInfo("标签树-删除信息").WithSave(true)
		delPP := req.NewPermission("tag:del")
		// 删除标签
		project.DELETE(":id", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(delProjectLog).
				WithRequiredPermission(delPP).
				Handle(m.DelTagTree)
		})
	}
}

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

	tagTree := router.Group("/tag-trees")
	{
		// 获取标签树列表
		tagTree.GET("", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(m.GetTagTree)
		})

		// 根据条件获取标签
		tagTree.GET("query", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(m.ListByQuery)
		})

		// 获取登录账号拥有的标签信息
		tagTree.GET("account-has", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(m.GetAccountTags)
		})

		saveTagTreeLog := req.NewLogInfo("标签树-保存信息").WithSave(true)
		savePP := req.NewPermission("tag:save")
		// 保存项目树下的环境信息
		tagTree.POST("", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(saveTagTreeLog).
				WithRequiredPermission(savePP).
				Handle(m.SaveTagTree)
		})

		delTagLog := req.NewLogInfo("标签树-删除信息").WithSave(true)
		delPP := req.NewPermission("tag:del")
		// 删除标签
		tagTree.DELETE(":id", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(delTagLog).
				WithRequiredPermission(delPP).
				Handle(m.DelTagTree)
		})
	}
}

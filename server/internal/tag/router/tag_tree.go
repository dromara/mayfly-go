package router

import (
	"mayfly-go/internal/tag/api"
	"mayfly-go/internal/tag/application"
	"mayfly-go/pkg/ctx"

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
			ctx.NewReqCtxWithGin(c).Handle(m.GetTagTree)
		})

		// 获取登录账号拥有的标签信息
		project.GET("account-has", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(m.GetAccountTags)
		})

		saveProjectTreeLog := ctx.NewLogInfo("保存标签树信息").WithSave(true)
		savePP := ctx.NewPermission("tag:save")
		// 保存项目树下的环境信息
		project.POST("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(saveProjectTreeLog).
				WithRequiredPermission(savePP).
				Handle(m.SaveTagTree)
		})

		delProjectLog := ctx.NewLogInfo("删除标签树信息").WithSave(true)
		delPP := ctx.NewPermission("tag:del")
		// 删除标签
		project.DELETE(":id", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(delProjectLog).
				WithRequiredPermission(delPP).
				Handle(m.DelTagTree)
		})
	}
}

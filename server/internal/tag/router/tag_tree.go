package router

import (
	"mayfly-go/internal/tag/api"
	"mayfly-go/internal/tag/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitTagTreeRouter(router *gin.RouterGroup) {
	m := &api.TagTree{
		TagTreeApp:     application.GetTagTreeApp(),
		TagResourceApp: application.GetTagResourceApp(),
	}

	tagTree := router.Group("/tag-trees")
	{
		reqs := [...]*req.Conf{
			// 获取标签树列表
			req.NewGet("", m.GetTagTree),

			// 根据条件获取标签
			req.NewGet("query", m.ListByQuery),

			req.NewPost("", m.SaveTagTree).Log(req.NewLogSave("标签树-保存信息")).RequiredPermissionCode("tag:save"),

			req.NewDelete(":id", m.DelTagTree).Log(req.NewLogSave("标签树-删除信息")).RequiredPermissionCode("tag:del"),

			req.NewGet("/resources/:rtype/tag-paths", m.TagResources),

			req.NewGet("/resources", m.QueryTagResources),
		}

		req.BatchSetGroup(tagTree, reqs[:])
	}
}

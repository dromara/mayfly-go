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
		reqs := [...]*req.Conf{
			// 获取标签树列表
			req.NewGet("", m.GetTagTree),

			// 根据条件获取标签
			req.NewGet("query", m.ListByQuery),

			// 获取登录账号拥有的标签信息
			req.NewGet("account-has", m.GetAccountTags),

			req.NewPost("", m.SaveTagTree).Log(req.NewLogSave("标签树-保存信息")).RequiredPermissionCode("tag:save"),

			req.NewDelete(":id", m.DelTagTree).Log(req.NewLogSave("标签树-删除信息")).RequiredPermissionCode("tag:del"),
		}

		req.BatchSetGroup(tagTree, reqs[:])
	}
}

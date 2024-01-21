package router

import (
	"mayfly-go/internal/tag/api"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitTeamRouter(router *gin.RouterGroup) {
	m := new(api.Team)
	biz.ErrIsNil(ioc.Inject(m))

	team := router.Group("/teams")
	{
		reqs := [...]*req.Conf{
			// 获取团队列表
			req.NewGet("", m.GetTeams),

			req.NewPost("", m.SaveTeam).Log(req.NewLogSave("团队-保存信息")).RequiredPermissionCode("team:save"),

			req.NewDelete(":id", m.DelTeam).Log(req.NewLogSave("团队-删除信息")).RequiredPermissionCode("team:del"),

			// 获取团队的成员信息列表
			req.NewGet("/:id/members", m.GetTeamMembers),

			req.NewPost("/:id/members", m.SaveTeamMember).Log(req.NewLogSave("团队-新增成员")).RequiredPermissionCode("team:member:save"),

			req.NewDelete("/:id/members/:accountId", m.DelTeamMember).Log(req.NewLogSave("团队-删除成员")).RequiredPermissionCode("team:member:del"),

			// 获取团队关联的标签id列表
			req.NewGet("/:id/tags", m.GetTagIds),

			req.NewPost("/:id/tags", m.SaveTags).Log(req.NewLogSave("团队-保存标签关联信息")).RequiredPermissionCode("team:tag:save"),
		}

		req.BatchSetGroup(team, reqs[:])
	}
}

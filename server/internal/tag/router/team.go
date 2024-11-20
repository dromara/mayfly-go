package router

import (
	"mayfly-go/internal/tag/api"
	"mayfly-go/internal/tag/imsg"
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

			req.NewPost("", m.SaveTeam).Log(req.NewLogSaveI(imsg.LogTeamSave)).RequiredPermissionCode("team:save"),

			req.NewDelete(":id", m.DelTeam).Log(req.NewLogSaveI(imsg.LogTeamDelete)).RequiredPermissionCode("team:del"),

			// 获取团队的成员信息列表
			req.NewGet("/:id/members", m.GetTeamMembers),

			req.NewPost("/:id/members", m.SaveTeamMember).Log(req.NewLogSaveI(imsg.LogTeamAddMember)).RequiredPermissionCode("team:member:save"),

			req.NewDelete("/:id/members/:accountId", m.DelTeamMember).Log(req.NewLogSaveI(imsg.LogTeamRemoveMember)).RequiredPermissionCode("team:member:del"),
		}

		req.BatchSetGroup(team, reqs[:])
	}
}

package router

import (
	sysapp "mayfly-go/internal/sys/application"
	"mayfly-go/internal/tag/api"
	"mayfly-go/internal/tag/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitTeamRouter(router *gin.RouterGroup) {
	m := &api.Team{
		TeamApp:    application.GetTeamApp(),
		TagApp:     application.GetTagTreeApp(),
		AccountApp: sysapp.GetAccountApp(),
	}

	project := router.Group("/teams")
	{
		// 获取团队列表
		project.GET("", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(m.GetTeams)
		})

		saveProjectTeamLog := req.NewLogInfo("团队-保存信息").WithSave(true)
		savePP := req.NewPermission("team:save")
		// 保存项目团队信息
		project.POST("", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(saveProjectTeamLog).
				WithRequiredPermission(savePP).
				Handle(m.SaveTeam)
		})

		delProjectTeamLog := req.NewLogInfo("团队-删除信息").WithSave(true)
		delPP := req.NewPermission("team:del")
		project.DELETE(":id", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(delProjectTeamLog).
				WithRequiredPermission(delPP).
				Handle(m.DelTeam)
		})

		// 获取团队的成员信息列表
		project.GET("/:id/members", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(m.GetTeamMembers)
		})

		// 保存团队成员
		saveProjectTeamMemLog := req.NewLogInfo("团队-新增成员").WithSave(true)
		savePmP := req.NewPermission("team:member:save")
		project.POST("/:id/members", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(saveProjectTeamMemLog).
				WithRequiredPermission(savePmP).
				Handle(m.SaveTeamMember)
		})

		// 删除团队成员
		delProjectTeamMemLog := req.NewLogInfo("团队-删除成员").WithSave(true)
		savePmdP := req.NewPermission("team:member:del")
		project.DELETE("/:id/members/:accountId", func(c *gin.Context) {
			req.NewCtxWithGin(c).WithLog(delProjectTeamMemLog).
				WithRequiredPermission(savePmdP).
				Handle(m.DelTeamMember)
		})

		// 获取团队关联的标签id列表
		project.GET("/:id/tags", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(m.GetTagIds)
		})

		// 保存团队标签关联信息
		saveTeamTagLog := req.NewLogInfo("团队-保存标签关联信息").WithSave(true)
		saveTeamTagP := req.NewPermission("team:tag:save")
		project.POST("/:id/tags", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(saveTeamTagLog).
				WithRequiredPermission(saveTeamTagP).
				Handle(m.SaveTags)
		})
	}
}

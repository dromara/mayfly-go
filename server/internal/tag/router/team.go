package router

import (
	sysapp "mayfly-go/internal/sys/application"
	"mayfly-go/internal/tag/api"
	"mayfly-go/internal/tag/application"
	"mayfly-go/pkg/ctx"

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
			ctx.NewReqCtxWithGin(c).Handle(m.GetTeams)
		})

		saveProjectTeamLog := ctx.NewLogInfo("团队-保存信息").WithSave(true)
		savePP := ctx.NewPermission("team:save")
		// 保存项目团队信息
		project.POST("", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(saveProjectTeamLog).
				WithRequiredPermission(savePP).
				Handle(m.SaveTeam)
		})

		delProjectTeamLog := ctx.NewLogInfo("团队-删除信息").WithSave(true)
		delPP := ctx.NewPermission("team:del")
		project.DELETE(":id", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(delProjectTeamLog).
				WithRequiredPermission(delPP).
				Handle(m.DelTeam)
		})

		// 获取团队的成员信息列表
		project.GET("/:id/members", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(m.GetTeamMembers)
		})

		// 保存团队成员
		saveProjectTeamMemLog := ctx.NewLogInfo("团队-新增成员").WithSave(true)
		savePmP := ctx.NewPermission("team:member:save")
		project.POST("/:id/members", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(saveProjectTeamMemLog).
				WithRequiredPermission(savePmP).
				Handle(m.SaveTeamMember)
		})

		// 删除团队成员
		delProjectTeamMemLog := ctx.NewLogInfo("团队-删除成员").WithSave(true)
		savePmdP := ctx.NewPermission("team:member:del")
		project.DELETE("/:id/members/:accountId", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).WithLog(delProjectTeamMemLog).
				WithRequiredPermission(savePmdP).
				Handle(m.DelTeamMember)
		})

		// 获取团队关联的标签id列表
		project.GET("/:id/tags", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).Handle(m.GetTagIds)
		})

		// 保存团队标签关联信息
		saveTeamTagLog := ctx.NewLogInfo("团队-保存标签关联信息").WithSave(true)
		saveTeamTagP := ctx.NewPermission("team:tag:save")
		project.POST("/:id/tags", func(c *gin.Context) {
			ctx.NewReqCtxWithGin(c).
				WithLog(saveTeamTagLog).
				WithRequiredPermission(saveTeamTagP).
				Handle(m.SaveTags)
		})
	}
}

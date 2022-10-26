package api

import (
	"fmt"
	sys_applicaiton "mayfly-go/internal/sys/application"
	sys_entity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/tag/api/form"
	"mayfly-go/internal/tag/application"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ctx"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/utils"
)

type Team struct {
	TeamApp    application.Team
	TagApp     application.TagTree
	AccountApp sys_applicaiton.Account
}

func (p *Team) GetTeams(rc *ctx.ReqCtx) {
	teams := &[]entity.Team{}
	rc.ResData = p.TeamApp.GetPageList(&entity.Team{}, ginx.GetPageParam(rc.GinCtx), teams)
}

func (p *Team) SaveTeam(rc *ctx.ReqCtx) {
	team := &entity.Team{}
	ginx.BindJsonAndValid(rc.GinCtx, team)

	isAdd := team.Id == 0

	loginAccount := rc.LoginAccount
	team.SetBaseInfo(loginAccount)
	p.TeamApp.Save(team)

	// 如果是新增团队则默认将自己加入该团队
	if isAdd {
		teamMem := &entity.TeamMember{}
		teamMem.AccountId = loginAccount.Id
		teamMem.Username = loginAccount.Username
		teamMem.TeamId = team.Id

		teamMem.SetBaseInfo(rc.LoginAccount)
		p.TeamApp.SaveMember(teamMem)
	}
}

func (p *Team) DelTeam(rc *ctx.ReqCtx) {
	p.TeamApp.Delete(uint64(ginx.PathParamInt(rc.GinCtx, "id")))
}

// 获取团队的成员信息
func (p *Team) GetTeamMembers(rc *ctx.ReqCtx) {
	teamMems := &[]entity.TeamMember{}
	rc.ResData = p.TeamApp.GetMemberPage(&entity.TeamMember{TeamId: uint64(ginx.PathParamInt(rc.GinCtx, "id"))},
		ginx.GetPageParam(rc.GinCtx), teamMems)
}

// 保存团队信息
func (p *Team) SaveTeamMember(rc *ctx.ReqCtx) {
	projectMem := &entity.TeamMember{}
	ginx.BindJsonAndValid(rc.GinCtx, projectMem)

	rc.ReqParam = fmt.Sprintf("projectId: %d, username: %s", projectMem.TeamId, projectMem.Username)

	// 校验账号，并赋值username
	account := &sys_entity.Account{}
	account.Id = projectMem.AccountId
	biz.ErrIsNil(p.AccountApp.GetAccount(account, "Id", "Username"), "账号不存在")
	projectMem.Username = account.Username

	projectMem.SetBaseInfo(rc.LoginAccount)
	p.TeamApp.SaveMember(projectMem)
}

// 删除团队成员
func (p *Team) DelTeamMember(rc *ctx.ReqCtx) {
	g := rc.GinCtx
	tid := ginx.PathParamInt(g, "id")
	aid := ginx.PathParamInt(g, "accountId")
	rc.ReqParam = fmt.Sprintf("teamId: %d, accountId: %d", tid, aid)

	p.TeamApp.DeleteMember(uint64(tid), uint64(aid))
}

// 获取团队关联的标签id
func (p *Team) GetTagIds(rc *ctx.ReqCtx) {
	rc.ResData = p.TeamApp.ListTagIds(uint64(ginx.PathParamInt(rc.GinCtx, "id")))
}

// 保存团队关联标签信息
func (p *Team) SaveTags(rc *ctx.ReqCtx) {
	g := rc.GinCtx

	var form form.TagTreeTeam
	ginx.BindJsonAndValid(g, &form)

	teamId := form.TeamId

	// 将[]uint64转为[]interface{}
	oIds := p.TeamApp.ListTagIds(teamId)
	var oldIds []interface{}
	for _, v := range oIds {
		oldIds = append(oldIds, v)
	}

	var newIds []interface{}
	for _, v := range form.TagIds {
		newIds = append(newIds, v)
	}

	// 比较新旧两合集
	addIds, delIds, _ := utils.ArrayCompare(newIds, oldIds, func(i1, i2 interface{}) bool {
		return i1.(uint64) == i2.(uint64)
	})

	loginAccount := rc.LoginAccount
	for _, v := range addIds {
		tagId := v.(uint64)
		tag := p.TagApp.GetById(tagId)
		biz.NotNil(tag, "存在非法标签id")

		ptt := &entity.TagTreeTeam{TeamId: teamId, TagId: tagId, TagPath: tag.CodePath}
		ptt.SetBaseInfo(loginAccount)
		p.TeamApp.SaveTag(ptt)
	}
	for _, v := range delIds {
		p.TeamApp.DeleteTag(teamId, v.(uint64))
	}

	rc.ReqParam = form
}

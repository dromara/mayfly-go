package api

import (
	"fmt"
	sys_applicaiton "mayfly-go/internal/sys/application"
	sys_entity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/tag/api/form"
	"mayfly-go/internal/tag/api/vo"
	"mayfly-go/internal/tag/application"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils"
)

type Team struct {
	TeamApp    application.Team
	TagApp     application.TagTree
	AccountApp sys_applicaiton.Account
}

func (p *Team) GetTeams(rc *req.Ctx) {
	teams := &[]entity.Team{}
	rc.ResData = p.TeamApp.GetPageList(&entity.Team{}, ginx.GetPageParam(rc.GinCtx), teams)
}

func (p *Team) SaveTeam(rc *req.Ctx) {
	team := &entity.Team{}
	ginx.BindJsonAndValid(rc.GinCtx, team)
	rc.ReqParam = team
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

func (p *Team) DelTeam(rc *req.Ctx) {
	p.TeamApp.Delete(uint64(ginx.PathParamInt(rc.GinCtx, "id")))
}

// 获取团队的成员信息
func (p *Team) GetTeamMembers(rc *req.Ctx) {
	condition := &entity.TeamMember{TeamId: uint64(ginx.PathParamInt(rc.GinCtx, "id"))}
	condition.Username = rc.GinCtx.Query("username")

	rc.ResData = p.TeamApp.GetMemberPage(condition, ginx.GetPageParam(rc.GinCtx), &[]vo.TeamMember{})
}

// 保存团队信息
func (p *Team) SaveTeamMember(rc *req.Ctx) {
	teamMems := &form.TeamMember{}
	ginx.BindJsonAndValid(rc.GinCtx, teamMems)

	teamId := teamMems.TeamId

	for _, accountId := range teamMems.AccountIds {
		if p.TeamApp.IsExistMember(teamId, accountId) {
			continue
		}

		// 校验账号，并赋值username
		account := &sys_entity.Account{}
		account.Id = accountId
		biz.ErrIsNil(p.AccountApp.GetAccount(account, "Id", "Username"), "账号不存在")

		teamMember := new(entity.TeamMember)
		teamMember.TeamId = teamId
		teamMember.AccountId = accountId
		teamMember.Username = account.Username
		teamMember.SetBaseInfo(rc.LoginAccount)
		p.TeamApp.SaveMember(teamMember)
	}

	rc.ReqParam = teamMems
}

// 删除团队成员
func (p *Team) DelTeamMember(rc *req.Ctx) {
	g := rc.GinCtx
	tid := ginx.PathParamInt(g, "id")
	aid := ginx.PathParamInt(g, "accountId")
	rc.ReqParam = fmt.Sprintf("teamId: %d, accountId: %d", tid, aid)

	p.TeamApp.DeleteMember(uint64(tid), uint64(aid))
}

// 获取团队关联的标签id
func (p *Team) GetTagIds(rc *req.Ctx) {
	rc.ResData = p.TeamApp.ListTagIds(uint64(ginx.PathParamInt(rc.GinCtx, "id")))
}

// 保存团队关联标签信息
func (p *Team) SaveTags(rc *req.Ctx) {
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

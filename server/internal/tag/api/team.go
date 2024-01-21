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
	"mayfly-go/pkg/utils/collx"
	"strconv"
	"strings"
)

type Team struct {
	TeamApp    application.Team        `inject:""`
	TagTreeApp application.TagTree     `inject:""`
	AccountApp sys_applicaiton.Account `inject:""`
}

func (p *Team) GetTeams(rc *req.Ctx) {
	queryCond, page := ginx.BindQueryAndPage(rc.GinCtx, new(entity.TeamQuery))
	teams := &[]entity.Team{}
	res, err := p.TeamApp.GetPageList(queryCond, page, teams)
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (p *Team) SaveTeam(rc *req.Ctx) {
	team := &entity.Team{}
	ginx.BindJsonAndValid(rc.GinCtx, team)
	rc.ReqParam = team
	isAdd := team.Id == 0

	loginAccount := rc.GetLoginAccount()
	p.TeamApp.Save(rc.MetaCtx, team)

	// 如果是新增团队则默认将自己加入该团队
	if isAdd {
		teamMem := &entity.TeamMember{}
		teamMem.AccountId = loginAccount.Id
		teamMem.Username = loginAccount.Username
		teamMem.TeamId = team.Id

		p.TeamApp.SaveMember(rc.MetaCtx, teamMem)
	}
}

func (p *Team) DelTeam(rc *req.Ctx) {
	idsStr := ginx.PathParam(rc.GinCtx, "id")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		p.TeamApp.Delete(rc.MetaCtx, uint64(value))
	}
}

// 获取团队的成员信息
func (p *Team) GetTeamMembers(rc *req.Ctx) {
	condition := &entity.TeamMember{TeamId: uint64(ginx.PathParamInt(rc.GinCtx, "id"))}
	condition.Username = rc.GinCtx.Query("username")

	res, err := p.TeamApp.GetMemberPage(condition, ginx.GetPageParam(rc.GinCtx), &[]vo.TeamMember{})
	biz.ErrIsNil(err)
	rc.ResData = res
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
		biz.ErrIsNil(p.AccountApp.GetBy(account, "Id", "Username"), "账号不存在")

		teamMember := new(entity.TeamMember)
		teamMember.TeamId = teamId
		teamMember.AccountId = accountId
		teamMember.Username = account.Username
		p.TeamApp.SaveMember(rc.MetaCtx, teamMember)
	}

	rc.ReqParam = teamMems
}

// 删除团队成员
func (p *Team) DelTeamMember(rc *req.Ctx) {
	g := rc.GinCtx
	tid := ginx.PathParamInt(g, "id")
	aid := ginx.PathParamInt(g, "accountId")
	rc.ReqParam = fmt.Sprintf("teamId: %d, accountId: %d", tid, aid)

	p.TeamApp.DeleteMember(rc.MetaCtx, uint64(tid), uint64(aid))
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

	// 将[]uint64转为[]any
	oIds := p.TeamApp.ListTagIds(teamId)

	// 比较新旧两合集
	addIds, delIds, _ := collx.ArrayCompare(form.TagIds, oIds)

	for _, v := range addIds {
		tagId := v
		tag, err := p.TagTreeApp.GetById(new(entity.TagTree), tagId)
		biz.ErrIsNil(err, "存在非法标签id")

		ptt := &entity.TagTreeTeam{TeamId: teamId, TagId: tagId, TagPath: tag.CodePath}
		p.TeamApp.SaveTag(rc.MetaCtx, ptt)
	}
	for _, v := range delIds {
		p.TeamApp.DeleteTag(rc.MetaCtx, teamId, v)
	}

	rc.ReqParam = form
}

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
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"strings"

	"github.com/may-fly/cast"
)

type Team struct {
	TeamApp          application.Team          `inject:""`
	TagTreeApp       application.TagTree       `inject:""`
	TagTreeRelateApp application.TagTreeRelate `inject:""`
	AccountApp       sys_applicaiton.Account   `inject:""`
}

func (p *Team) GetTeams(rc *req.Ctx) {
	queryCond, page := req.BindQueryAndPage(rc, new(entity.TeamQuery))
	var teams []*vo.Team
	res, err := p.TeamApp.GetPageList(queryCond, page, &teams)
	biz.ErrIsNil(err)

	p.TagTreeRelateApp.FillTagInfo(entity.TagRelateTypeTeam, collx.ArrayMap(teams, func(mvo *vo.Team) entity.IRelateTag {
		return mvo
	})...)

	rc.ResData = res
}

func (p *Team) SaveTeam(rc *req.Ctx) {
	team := req.BindJsonAndValid(rc, new(application.SaveTeamParam))
	rc.ReqParam = team
	biz.ErrIsNil(p.TeamApp.Save(rc.MetaCtx, team))
}

func (p *Team) DelTeam(rc *req.Ctx) {
	idsStr := rc.PathParam("id")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		p.TeamApp.Delete(rc.MetaCtx, cast.ToUint64(v))
	}
}

// 获取团队的成员信息
func (p *Team) GetTeamMembers(rc *req.Ctx) {
	condition := &entity.TeamMember{TeamId: uint64(rc.PathParamInt("id"))}
	condition.Username = rc.Query("username")

	res, err := p.TeamApp.GetMemberPage(condition, rc.GetPageParam(), &[]vo.TeamMember{})
	biz.ErrIsNil(err)
	rc.ResData = res
}

// 保存团队信息
func (p *Team) SaveTeamMember(rc *req.Ctx) {
	teamMems := req.BindJsonAndValid(rc, new(form.TeamMember))

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
	tid := rc.PathParamInt("id")
	aid := rc.PathParamInt("accountId")
	rc.ReqParam = fmt.Sprintf("teamId: %d, accountId: %d", tid, aid)

	p.TeamApp.DeleteMember(rc.MetaCtx, uint64(tid), uint64(aid))
}

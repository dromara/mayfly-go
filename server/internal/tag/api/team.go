package api

import (
	"fmt"
	sys_applicaiton "mayfly-go/internal/sys/application"
	sys_entity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/tag/api/form"
	"mayfly-go/internal/tag/api/vo"
	"mayfly-go/internal/tag/application"
	"mayfly-go/internal/tag/application/dto"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"strings"

	"github.com/may-fly/cast"
)

type Team struct {
	teamApp          application.Team          `inject:"T"`
	tagTreeRelateApp application.TagTreeRelate `inject:"T"`
	accountApp       sys_applicaiton.Account   `inject:"T"`
}

func (t *Team) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 获取团队列表
		req.NewGet("", t.GetTeams),

		req.NewPost("", t.SaveTeam).Log(req.NewLogSaveI(imsg.LogTeamSave)).RequiredPermissionCode("team:save"),

		req.NewDelete(":id", t.DelTeam).Log(req.NewLogSaveI(imsg.LogTeamDelete)).RequiredPermissionCode("team:del"),

		// 获取团队的成员信息列表
		req.NewGet("/:id/members", t.GetTeamMembers),

		req.NewPost("/:id/members", t.SaveTeamMember).Log(req.NewLogSaveI(imsg.LogTeamAddMember)).RequiredPermissionCode("team:member:save"),

		req.NewDelete("/:id/members/:accountId", t.DelTeamMember).Log(req.NewLogSaveI(imsg.LogTeamRemoveMember)).RequiredPermissionCode("team:member:del"),
	}

	return req.NewConfs("/teams", reqs[:]...)
}

func (p *Team) GetTeams(rc *req.Ctx) {
	queryCond := req.BindQuery[*entity.TeamQuery](rc)

	res, err := p.teamApp.GetPageList(queryCond)
	biz.ErrIsNil(err)
	resVo := model.PageResultConv[*entity.Team, *vo.Team](res)
	teams := resVo.List
	p.tagTreeRelateApp.FillTagInfo(entity.TagRelateTypeTeam, collx.ArrayMap(teams, func(mvo *vo.Team) entity.IRelateTag {
		return mvo
	})...)

	rc.ResData = resVo
}

func (p *Team) SaveTeam(rc *req.Ctx) {
	team := req.BindJsonAndValid[*dto.SaveTeam](rc)
	rc.ReqParam = team
	biz.ErrIsNil(p.teamApp.SaveTeam(rc.MetaCtx, team))
}

func (p *Team) DelTeam(rc *req.Ctx) {
	idsStr := rc.PathParam("id")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		p.teamApp.Delete(rc.MetaCtx, cast.ToUint64(v))
	}
}

// 获取团队的成员信息
func (p *Team) GetTeamMembers(rc *req.Ctx) {
	condition := &entity.TeamMember{TeamId: uint64(rc.PathParamInt("id"))}
	condition.Username = rc.Query("username")

	res, err := p.teamApp.GetMemberPage(condition, rc.GetPageParam())
	biz.ErrIsNil(err)
	rc.ResData = res
}

// 保存团队信息
func (p *Team) SaveTeamMember(rc *req.Ctx) {
	teamMems := req.BindJsonAndValid[*form.TeamMember](rc)

	teamId := teamMems.TeamId

	for _, accountId := range teamMems.AccountIds {
		if p.teamApp.IsExistMember(teamId, accountId) {
			continue
		}

		// 校验账号，并赋值username
		account := &sys_entity.Account{}
		account.Id = accountId

		biz.ErrIsNil(p.accountApp.GetByCond(model.NewModelCond(account).Columns("Id", "Username")), "账号不存在")

		teamMember := new(entity.TeamMember)
		teamMember.TeamId = teamId
		teamMember.AccountId = accountId
		teamMember.Username = account.Username
		p.teamApp.SaveMember(rc.MetaCtx, teamMember)
	}

	rc.ReqParam = teamMems
}

// 删除团队成员
func (p *Team) DelTeamMember(rc *req.Ctx) {
	tid := rc.PathParamInt("id")
	aid := rc.PathParamInt("accountId")
	rc.ReqParam = fmt.Sprintf("teamId: %d, accountId: %d", tid, aid)

	p.teamApp.DeleteMember(rc.MetaCtx, uint64(tid), uint64(aid))
}

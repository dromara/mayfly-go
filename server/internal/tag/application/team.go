package application

import (
	"context"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/internal/tag/infrastructure/cache"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
)

type SaveTeamParam struct {
	Id     uint64 `json:"id"`
	Name   string `json:"name" binding:"required"` // 名称
	Remark string `json:"remark"`                  // 备注说明

	CodePaths []string `json:"codePaths"` // 关联标签信息
}

type Team interface {
	base.App[*entity.Team]

	// 分页获取项目团队信息列表
	GetPageList(condition *entity.TeamQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	// SaveTeam 保存团队信息
	SaveTeam(ctx context.Context, team *SaveTeamParam) error

	Delete(ctx context.Context, id uint64) error

	//--------------- 团队成员相关接口 ---------------

	GetMemberPage(condition *entity.TeamMember, pageParam *model.PageParam, toEntity any) (*model.PageResult[any], error)

	SaveMember(ctx context.Context, tagTeamMember *entity.TeamMember)

	DeleteMember(tx context.Context, teamId, accountId uint64)

	IsExistMember(teamId, accounId uint64) bool

	DeleteTag(tx context.Context, teamId, tagId uint64) error
}

type teamAppImpl struct {
	base.AppImpl[*entity.Team, repository.Team]

	teamMemberRepo   repository.TeamMember `inject:"TeamMemberRepo"`
	tagTreeRelateApp TagTreeRelate         `inject:"TagTreeRelateApp"`
}

var _ (Team) = (*teamAppImpl)(nil)

func (p *teamAppImpl) InjectTeamRepo(repo repository.Team) {
	p.Repo = repo
}

func (p *teamAppImpl) GetPageList(condition *entity.TeamQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return p.GetRepo().GetPageList(condition, pageParam, toEntity, orderBy...)
}

func (p *teamAppImpl) SaveTeam(ctx context.Context, saveParam *SaveTeamParam) error {
	team := &entity.Team{Name: saveParam.Name, Remark: saveParam.Remark}
	team.Id = saveParam.Id

	if team.Id == 0 {
		if p.CountByCond(&entity.Team{Name: saveParam.Name}) > 0 {
			return errorx.NewBiz("团队名[%s]已存在", saveParam.Name)
		}

		if err := p.Insert(ctx, team); err != nil {
			return err
		}

		loginAccount := contextx.GetLoginAccount(ctx)
		logx.DebugfContext(ctx, "将[%s]默认加入至[%s]团队", loginAccount.Username, team.Name)

		teamMem := &entity.TeamMember{}
		teamMem.AccountId = loginAccount.Id
		teamMem.Username = loginAccount.Username
		teamMem.TeamId = team.Id
		p.SaveMember(ctx, teamMem)
	} else {
		// 置空名称，防止变更
		team.Name = ""
		if err := p.UpdateById(ctx, team); err != nil {
			return err
		}
	}

	// 删除该团队关联账号的标签缓存
	teamMembers, _ := p.teamMemberRepo.SelectByCond(&entity.TeamMember{TeamId: team.Id})
	for _, tm := range teamMembers {
		cache.DelAccountTagPaths(tm.AccountId)
	}

	// 保存团队关联的标签信息
	return p.tagTreeRelateApp.RelateTag(ctx, entity.TagRelateTypeTeam, team.Id, saveParam.CodePaths...)
}

func (p *teamAppImpl) Delete(ctx context.Context, id uint64) error {
	return p.Tx(ctx, func(ctx context.Context) error {
		return p.DeleteById(ctx, id)
	}, func(ctx context.Context) error {
		return p.teamMemberRepo.DeleteByCond(ctx, &entity.TeamMember{TeamId: id})
	}, func(ctx context.Context) error {
		return p.tagTreeRelateApp.DeleteByCond(ctx, &entity.TagTreeRelate{RelateType: entity.TagRelateTypeTeam, RelateId: id})
	})
}

// --------------- 团队成员相关接口 ---------------

func (p *teamAppImpl) GetMemberPage(condition *entity.TeamMember, pageParam *model.PageParam, toEntity any) (*model.PageResult[any], error) {
	return p.teamMemberRepo.GetPageList(condition, pageParam, toEntity)
}

// 保存团队成员信息
func (p *teamAppImpl) SaveMember(ctx context.Context, teamMember *entity.TeamMember) {
	teamMember.Id = 0
	biz.IsTrue(!p.teamMemberRepo.IsExist(teamMember.TeamId, teamMember.AccountId), "该成员已存在")
	p.teamMemberRepo.Insert(ctx, teamMember)
}

// 删除团队成员信息
func (p *teamAppImpl) DeleteMember(ctx context.Context, teamId, accountId uint64) {
	p.teamMemberRepo.DeleteByCond(ctx, &entity.TeamMember{TeamId: teamId, AccountId: accountId})
}

func (p *teamAppImpl) IsExistMember(teamId, accounId uint64) bool {
	return p.teamMemberRepo.IsExist(teamId, accounId)
}

//--------------- 标签相关接口 ---------------

// 删除关联标签信息
func (p *teamAppImpl) DeleteTag(ctx context.Context, teamId, tagId uint64) error {
	return p.tagTreeRelateApp.DeleteByCond(ctx, &entity.TagTreeRelate{RelateType: entity.TagRelateTypeTeam, RelateId: teamId, TagId: tagId})
}

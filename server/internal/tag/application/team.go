package application

import (
	"context"
	"mayfly-go/internal/tag/application/dto"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/internal/tag/imsg"
	"mayfly-go/internal/tag/infrastructure/cache"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
)

type Team interface {
	base.App[*entity.Team]

	// 分页获取项目团队信息列表
	GetPageList(condition *entity.TeamQuery, orderBy ...string) (*model.PageResult[*entity.Team], error)

	// SaveTeam 保存团队信息
	SaveTeam(ctx context.Context, team *dto.SaveTeam) error

	Delete(ctx context.Context, id uint64) error

	//--------------- 团队成员相关接口 ---------------

	GetMemberPage(condition *entity.TeamMember, pageParam model.PageParam) (*model.PageResult[*entity.TeamMemberPO], error)

	SaveMember(ctx context.Context, tagTeamMember *entity.TeamMember) error

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

func (p *teamAppImpl) GetPageList(condition *entity.TeamQuery, orderBy ...string) (*model.PageResult[*entity.Team], error) {
	return p.GetRepo().GetPageList(condition, orderBy...)
}

func (p *teamAppImpl) SaveTeam(ctx context.Context, saveParam *dto.SaveTeam) error {
	team := &entity.Team{
		Name:              saveParam.Name,
		ValidityStartDate: &saveParam.ValidityStartDate.Time,
		ValidityEndDate:   &saveParam.ValidityEndDate.Time,
		Remark:            saveParam.Remark,
	}
	team.Id = saveParam.Id

	return p.Tx(ctx, func(ctx context.Context) error {
		if team.Id == 0 {
			if p.CountByCond(&entity.Team{Name: saveParam.Name}) > 0 {
				return errorx.NewBizI(ctx, imsg.ErrNameExist)
			}

			if err := p.Insert(ctx, team); err != nil {
				return err
			}

			loginAccount := contextx.GetLoginAccount(ctx)
			logx.InfoContext(ctx, "Add [%s] to the [%s] team by default", loginAccount.Username, team.Name)

			teamMem := &entity.TeamMember{}
			teamMem.AccountId = loginAccount.Id
			teamMem.Username = loginAccount.Username
			teamMem.TeamId = team.Id
			if err := p.SaveMember(ctx, teamMem); err != nil {
				return err
			}
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
	})
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

func (p *teamAppImpl) GetMemberPage(condition *entity.TeamMember, pageParam model.PageParam) (*model.PageResult[*entity.TeamMemberPO], error) {
	return p.teamMemberRepo.GetPageList(condition, pageParam)
}

// 保存团队成员信息
func (p *teamAppImpl) SaveMember(ctx context.Context, teamMember *entity.TeamMember) error {
	teamMember.Id = 0
	if p.teamMemberRepo.IsExist(teamMember.TeamId, teamMember.AccountId) {
		return errorx.NewBizI(ctx, imsg.ErrMemeberExist)
	}
	return p.teamMemberRepo.Insert(ctx, teamMember)
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

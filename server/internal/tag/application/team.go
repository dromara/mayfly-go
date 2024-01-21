package application

import (
	"context"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"

	"gorm.io/gorm"
)

type Team interface {
	// 分页获取项目团队信息列表
	GetPageList(condition *entity.TeamQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	Save(ctx context.Context, team *entity.Team) error

	Delete(ctx context.Context, id uint64) error

	//--------------- 团队成员相关接口 ---------------

	GetMemberPage(condition *entity.TeamMember, pageParam *model.PageParam, toEntity any) (*model.PageResult[any], error)

	SaveMember(ctx context.Context, tagTeamMember *entity.TeamMember)

	DeleteMember(tx context.Context, teamId, accountId uint64)

	IsExistMember(teamId, accounId uint64) bool

	//--------------- 关联项目相关接口 ---------------

	ListTagIds(teamId uint64) []uint64

	SaveTag(ctx context.Context, tagTeam *entity.TagTreeTeam) error

	DeleteTag(tx context.Context, teamId, tagId uint64) error
}

type teamAppImpl struct {
	TeamRepo        repository.Team        `inject:""`
	TeamMemberRepo  repository.TeamMember  `inject:""`
	TagTreeTeamRepo repository.TagTreeTeam `inject:""`
}

func (p *teamAppImpl) GetPageList(condition *entity.TeamQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return p.TeamRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

func (p *teamAppImpl) Save(ctx context.Context, team *entity.Team) error {
	if team.Id == 0 {
		return p.TeamRepo.Insert(ctx, team)
	}
	return p.TeamRepo.UpdateById(ctx, team)
}

func (p *teamAppImpl) Delete(ctx context.Context, id uint64) error {
	return gormx.Tx(
		func(db *gorm.DB) error {
			return p.TeamRepo.DeleteByIdWithDb(ctx, db, id)
		},
		func(db *gorm.DB) error {
			return p.TeamMemberRepo.DeleteByCondWithDb(ctx, db, &entity.TeamMember{TeamId: id})
		},
		func(db *gorm.DB) error {
			return p.TagTreeTeamRepo.DeleteByCondWithDb(ctx, db, &entity.TagTreeTeam{TeamId: id})
		},
	)
}

// --------------- 团队成员相关接口 ---------------

func (p *teamAppImpl) GetMemberPage(condition *entity.TeamMember, pageParam *model.PageParam, toEntity any) (*model.PageResult[any], error) {
	return p.TeamMemberRepo.GetPageList(condition, pageParam, toEntity)
}

// 保存团队成员信息
func (p *teamAppImpl) SaveMember(ctx context.Context, teamMember *entity.TeamMember) {
	teamMember.Id = 0
	biz.IsTrue(!p.TeamMemberRepo.IsExist(teamMember.TeamId, teamMember.AccountId), "该成员已存在")
	p.TeamMemberRepo.Insert(ctx, teamMember)
}

// 删除团队成员信息
func (p *teamAppImpl) DeleteMember(ctx context.Context, teamId, accountId uint64) {
	p.TeamMemberRepo.DeleteByCond(ctx, &entity.TeamMember{TeamId: teamId, AccountId: accountId})
}

func (p *teamAppImpl) IsExistMember(teamId, accounId uint64) bool {
	return p.TeamMemberRepo.IsExist(teamId, accounId)
}

//--------------- 关联标签相关接口 ---------------

func (p *teamAppImpl) ListTagIds(teamId uint64) []uint64 {
	tags := &[]entity.TagTreeTeam{}
	p.TagTreeTeamRepo.ListByCondOrder(&entity.TagTreeTeam{TeamId: teamId}, tags)
	ids := make([]uint64, 0)
	for _, v := range *tags {
		ids = append(ids, v.TagId)
	}
	return ids
}

// 保存关联项目信息
func (p *teamAppImpl) SaveTag(ctx context.Context, tagTreeTeam *entity.TagTreeTeam) error {
	tagTreeTeam.Id = 0
	return p.TagTreeTeamRepo.Insert(ctx, tagTreeTeam)
}

// 删除关联项目信息
func (p *teamAppImpl) DeleteTag(ctx context.Context, teamId, tagId uint64) error {
	return p.TagTreeTeamRepo.DeleteByCond(ctx, &entity.TagTreeTeam{TeamId: teamId, TagId: tagId})
}

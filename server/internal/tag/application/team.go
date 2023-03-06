package application

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type Team interface {
	// 分页获取项目团队信息列表
	GetPageList(condition *entity.Team, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	Save(team *entity.Team)

	Delete(id uint64)

	//--------------- 团队成员相关接口 ---------------

	GetMemberPage(condition *entity.TeamMember, pageParam *model.PageParam, toEntity interface{}) *model.PageResult

	SaveMember(tagTeamMember *entity.TeamMember)

	DeleteMember(teamId, accountId uint64)

	IsExistMember(teamId, accounId uint64) bool

	//--------------- 关联项目相关接口 ---------------

	ListTagIds(teamId uint64) []uint64

	SaveTag(tagTeam *entity.TagTreeTeam)

	DeleteTag(teamId, tagId uint64)
}

func newTeamApp(teamRepo repository.Team,
	teamMemberRepo repository.TeamMember,
	tagTreeTeamRepo repository.TagTreeTeam,
) Team {
	return &teamAppImpl{
		teamRepo:        teamRepo,
		teamMemberRepo:  teamMemberRepo,
		tagTreeTeamRepo: tagTreeTeamRepo,
	}
}

type teamAppImpl struct {
	teamRepo        repository.Team
	teamMemberRepo  repository.TeamMember
	tagTreeTeamRepo repository.TagTreeTeam
}

func (p *teamAppImpl) GetPageList(condition *entity.Team, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return p.teamRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

func (p *teamAppImpl) Save(team *entity.Team) {
	if team.Id == 0 {
		p.teamRepo.Insert(team)
	} else {
		p.teamRepo.UpdateById(team)
	}
}

func (p *teamAppImpl) Delete(id uint64) {
	p.teamRepo.Delete(id)
	p.teamMemberRepo.DeleteBy(&entity.TeamMember{TeamId: id})
}

// --------------- 团队成员相关接口 ---------------

func (p *teamAppImpl) GetMemberPage(condition *entity.TeamMember, pageParam *model.PageParam, toEntity interface{}) *model.PageResult {
	return p.teamMemberRepo.GetPageList(condition, pageParam, toEntity)
}

// 保存团队成员信息
func (p *teamAppImpl) SaveMember(teamMember *entity.TeamMember) {
	teamMember.Id = 0
	biz.IsTrue(!p.teamMemberRepo.IsExist(teamMember.TeamId, teamMember.AccountId), "该成员已存在")
	p.teamMemberRepo.Save(teamMember)
}

// 删除团队成员信息
func (p *teamAppImpl) DeleteMember(teamId, accountId uint64) {
	p.teamMemberRepo.DeleteBy(&entity.TeamMember{TeamId: teamId, AccountId: accountId})
}

func (p *teamAppImpl) IsExistMember(teamId, accounId uint64) bool {
	return p.teamMemberRepo.IsExist(teamId, accounId)
}

//--------------- 关联项目相关接口 ---------------

func (p *teamAppImpl) ListTagIds(teamId uint64) []uint64 {
	tags := &[]entity.TagTreeTeam{}
	p.tagTreeTeamRepo.ListTag(&entity.TagTreeTeam{TeamId: teamId}, tags)
	ids := make([]uint64, 0)
	for _, v := range *tags {
		ids = append(ids, v.TagId)
	}
	return ids
}

// 保存关联项目信息
func (p *teamAppImpl) SaveTag(tagTreeTeam *entity.TagTreeTeam) {
	tagTreeTeam.Id = 0
	p.tagTreeTeamRepo.Save(tagTreeTeam)
}

// 删除关联项目信息
func (p *teamAppImpl) DeleteTag(teamId, tagId uint64) {
	p.tagTreeTeamRepo.DeleteBy(&entity.TagTreeTeam{TeamId: teamId, TagId: tagId})
}

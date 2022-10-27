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

	Save(projectTeam *entity.Team)

	Delete(id uint64)

	//--------------- 团队成员相关接口 ---------------

	GetMemberPage(condition *entity.TeamMember, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	SaveMember(projectTeamMember *entity.TeamMember)

	DeleteMember(teamId, accountId uint64)

	// 账号是否有权限访问该项目关联的资源信息
	// CanAccess(accountId, projectId uint64) error

	//--------------- 关联项目相关接口 ---------------

	ListTagIds(teamId uint64) []uint64

	SaveTag(tagTeam *entity.TagTreeTeam)

	DeleteTag(teamId, projectId uint64)
}

func newTeamApp(projectTeamRepo repository.Team,
	projectTeamMemberRepo repository.TeamMember,
	tagTreeTeamRepo repository.TagTreeTeam,
) Team {
	return &projectTeamAppImpl{
		projectTeamRepo:       projectTeamRepo,
		projectTeamMemberRepo: projectTeamMemberRepo,
		tagTreeTeamRepo:       tagTreeTeamRepo,
	}
}

type projectTeamAppImpl struct {
	projectTeamRepo       repository.Team
	projectTeamMemberRepo repository.TeamMember
	tagTreeTeamRepo       repository.TagTreeTeam
}

func (p *projectTeamAppImpl) GetPageList(condition *entity.Team, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return p.projectTeamRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

func (p *projectTeamAppImpl) Save(projectTeam *entity.Team) {
	if projectTeam.Id == 0 {
		p.projectTeamRepo.Insert(projectTeam)
	} else {
		p.projectTeamRepo.UpdateById(projectTeam)
	}
}

func (p *projectTeamAppImpl) Delete(id uint64) {
	p.projectTeamRepo.Delete(id)
	p.projectTeamMemberRepo.DeleteBy(&entity.TeamMember{TeamId: id})
}

// --------------- 团队成员相关接口 ---------------

func (p *projectTeamAppImpl) GetMemberPage(condition *entity.TeamMember, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return p.projectTeamMemberRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

// 保存团队成员信息
func (p *projectTeamAppImpl) SaveMember(projectTeamMember *entity.TeamMember) {
	projectTeamMember.Id = 0
	biz.IsTrue(!p.projectTeamMemberRepo.IsExist(projectTeamMember.TeamId, projectTeamMember.AccountId), "该成员已存在")
	p.projectTeamMemberRepo.Save(projectTeamMember)
}

// 删除团队成员信息
func (p *projectTeamAppImpl) DeleteMember(teamId, accountId uint64) {
	p.projectTeamMemberRepo.DeleteBy(&entity.TeamMember{TeamId: teamId, AccountId: accountId})
}

//--------------- 关联项目相关接口 ---------------

func (p *projectTeamAppImpl) ListTagIds(teamId uint64) []uint64 {
	projects := &[]entity.TagTreeTeam{}
	p.tagTreeTeamRepo.ListProject(&entity.TagTreeTeam{TeamId: teamId}, projects)
	ids := make([]uint64, 0)
	for _, v := range *projects {
		ids = append(ids, v.TagId)
	}
	return ids
}

// 保存关联项目信息
func (p *projectTeamAppImpl) SaveTag(projectTreeTeam *entity.TagTreeTeam) {
	projectTreeTeam.Id = 0
	p.tagTreeTeamRepo.Save(projectTreeTeam)
}

// 删除关联项目信息
func (p *projectTeamAppImpl) DeleteTag(teamId, tagId uint64) {
	p.tagTreeTeamRepo.DeleteBy(&entity.TagTreeTeam{TeamId: teamId, TagId: tagId})
}

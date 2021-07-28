package application

import (
	"mayfly-go/base/biz"
	"mayfly-go/base/model"
	"mayfly-go/server/devops/domain/entity"
	"mayfly-go/server/devops/domain/repository"
	"mayfly-go/server/devops/infrastructure/persistence"
)

type Project interface {
	// 分页获取项目信息列表
	GetPageList(condition *entity.Project, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	ListProjectByIds(ids []uint64, toEntity interface{}, orderBy ...string)

	SaveProject(project *entity.Project)

	// 根据项目id获取所有该项目下的环境信息列表
	ListEnvByProjectId(projectId uint64, listPtr interface{})

	// 保存项目环境信息
	SaveProjectEnv(projectEnv *entity.ProjectEnv)

	// 根据条件获取项目成员信息
	ListMember(condition *entity.ProjectMember, toEntity interface{}, orderBy ...string)

	SaveProjectMember(pm *entity.ProjectMember)

	// 根据条件获取项目成员信息
	GetMemberPage(condition *entity.ProjectMember, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	DeleteMember(projectId, accountId uint64)
}

type projectAppImpl struct {
	projectRepo       repository.Project
	projectEnvRepo    repository.ProjectEnv
	projectMemberRepo repository.ProjectMemeber
}

var ProjectApp Project = &projectAppImpl{
	projectRepo:       persistence.ProjectRepo,
	projectEnvRepo:    persistence.ProjectEnvRepo,
	projectMemberRepo: persistence.ProjectMemberRepo,
}

// 分页获取项目信息列表
func (p *projectAppImpl) GetPageList(condition *entity.Project, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return p.projectRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

func (p *projectAppImpl) ListProjectByIds(ids []uint64, toEntity interface{}, orderBy ...string) {
	p.projectRepo.GetByIdIn(ids, toEntity, orderBy...)
}

func (p *projectAppImpl) SaveProject(project *entity.Project) {
	if project.Id == 0 {
		p.projectRepo.Save(project)
	} else {
		// 防止误传导致项目名更新
		project.Name = ""
		p.projectRepo.Update(project)
	}
}

// 根据项目id获取所有该项目下的环境信息列表
func (p *projectAppImpl) ListEnvByProjectId(projectId uint64, listPtr interface{}) {
	p.projectEnvRepo.ListEnv(&entity.ProjectEnv{ProjectId: projectId}, listPtr)
}

// 保存项目环境信息
func (p *projectAppImpl) SaveProjectEnv(projectEnv *entity.ProjectEnv) {
	p.projectEnvRepo.Save(projectEnv)
}

// 根据条件获取项目成员信息
func (p *projectAppImpl) ListMember(condition *entity.ProjectMember, toEntity interface{}, orderBy ...string) {
	p.projectMemberRepo.ListMemeber(condition, toEntity, orderBy...)
}

func (p *projectAppImpl) SaveProjectMember(pm *entity.ProjectMember) {
	pms := new([]entity.ProjectMember)
	p.ListMember(&entity.ProjectMember{ProjectId: pm.ProjectId, AccountId: pm.AccountId}, pms)
	biz.IsTrue(len(*pms) == 0, "该成员已存在")
	p.projectMemberRepo.Save(pm)
}

func (p *projectAppImpl) GetMemberPage(condition *entity.ProjectMember, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return p.projectMemberRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

func (p *projectAppImpl) DeleteMember(projectId, accountId uint64) {
	p.projectMemberRepo.DeleteByPidMid(projectId, accountId)
}

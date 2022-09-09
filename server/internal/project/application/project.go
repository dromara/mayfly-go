package application

import (
	dbapp "mayfly-go/internal/db/application"
	dbentity "mayfly-go/internal/db/domain/entity"
	machineapp "mayfly-go/internal/machine/application"
	machineentity "mayfly-go/internal/machine/domain/entity"
	mongoapp "mayfly-go/internal/mongo/application"
	mongoentity "mayfly-go/internal/mongo/domain/entity"
	"mayfly-go/internal/project/domain/entity"
	"mayfly-go/internal/project/domain/repository"
	redisapp "mayfly-go/internal/redis/application"
	redisentity "mayfly-go/internal/redis/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type Project interface {
	// 分页获取项目信息列表
	GetPageList(condition *entity.Project, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	Count(condition *entity.Project) int64

	ListProjectByIds(ids []uint64, toEntity interface{}, orderBy ...string)

	SaveProject(project *entity.Project)

	DelProject(id uint64)

	DelProjectEnv(id uint64)

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

	// 账号是否有权限访问该项目关联的资源信息
	CanAccess(accountId, projectId uint64) error
}

func newProjectApp(projectRepo repository.Project,
	projectEnvRepo repository.ProjectEnv,
	projectMemberRepo repository.ProjectMemeber,
	machineApp machineapp.Machine,
	redisApp redisapp.Redis,
	dbApp dbapp.Db,
	mongoApp mongoapp.Mongo) Project {
	return &projectAppImpl{
		projectRepo:       projectRepo,
		projectEnvRepo:    projectEnvRepo,
		projectMemberRepo: projectMemberRepo,
		machineApp:        machineApp,
		redisApp:          redisApp,
		dbApp:             dbApp,
	}
}

type projectAppImpl struct {
	projectRepo       repository.Project
	projectEnvRepo    repository.ProjectEnv
	projectMemberRepo repository.ProjectMemeber
	machineApp        machineapp.Machine
	redisApp          redisapp.Redis
	mongoApp          mongoapp.Mongo
	dbApp             dbapp.Db
}

// 分页获取项目信息列表
func (p *projectAppImpl) GetPageList(condition *entity.Project, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return p.projectRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

func (p *projectAppImpl) Count(condition *entity.Project) int64 {
	return p.projectRepo.Count(condition)
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

func (p *projectAppImpl) DelProject(id uint64) {
	biz.IsTrue(p.machineApp.Count(&machineentity.Machine{ProjectId: id}) == 0, "请先删除该项目关联的机器信息")
	biz.IsTrue(p.redisApp.Count(&redisentity.Redis{ProjectId: id}) == 0, "请先删除该项目关联的redis信息")
	biz.IsTrue(p.dbApp.Count(&dbentity.Db{ProjectId: id}) == 0, "请先删除该项目关联的数据库信息")
	biz.IsTrue(p.mongoApp.Count(&mongoentity.Mongo{ProjectId: id}) == 0, "请先删除该项目关联的Mongo信息")
	p.projectRepo.Delete(id)
	p.projectEnvRepo.DeleteEnvs(id)
	p.projectMemberRepo.DeleteMems(id)
}

// 根据项目id获取所有该项目下的环境信息列表
func (p *projectAppImpl) ListEnvByProjectId(projectId uint64, listPtr interface{}) {
	p.projectEnvRepo.ListEnv(&entity.ProjectEnv{ProjectId: projectId}, listPtr)
}

// 保存项目环境信息
func (p *projectAppImpl) SaveProjectEnv(projectEnv *entity.ProjectEnv) {
	p.projectEnvRepo.Save(projectEnv)
}

// 删除项目环境信息
func (p *projectAppImpl) DelProjectEnv(id uint64) {
	biz.IsTrue(p.redisApp.Count(&redisentity.Redis{EnvId: id}) == 0, "请先删除该项目环境关联的redis信息")
	biz.IsTrue(p.dbApp.Count(&dbentity.Db{EnvId: id}) == 0, "请先删除该项目环境关联的数据库信息")
	biz.IsTrue(p.mongoApp.Count(&mongoentity.Mongo{ProjectId: id}) == 0, "请先删除该项目环境关联的Mongo信息")
	p.projectEnvRepo.DeleteEnv(id)
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

func (p *projectAppImpl) CanAccess(accountId, projectId uint64) error {
	if p.projectMemberRepo.IsExist(projectId, accountId) {
		return nil
	}
	return biz.NewBizErr("您无权操作该资源")
}

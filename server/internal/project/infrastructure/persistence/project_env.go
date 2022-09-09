package persistence

import (
	"mayfly-go/internal/project/domain/entity"
	"mayfly-go/internal/project/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type projectEnvRepoImpl struct{}

func newProjectEnvRepo() repository.ProjectEnv {
	return new(projectEnvRepoImpl)
}

func (p *projectEnvRepoImpl) ListEnv(condition *entity.ProjectEnv, toEntity interface{}, orderBy ...string) {
	model.ListByOrder(condition, toEntity, orderBy...)
}

func (p *projectEnvRepoImpl) Save(entity *entity.ProjectEnv) {
	biz.ErrIsNilAppendErr(model.Insert(entity), "保存环境失败：%s")
}

func (p *projectEnvRepoImpl) DeleteEnvs(projectId uint64) {
	model.DeleteByCondition(&entity.ProjectEnv{ProjectId: projectId})
}

func (p *projectEnvRepoImpl) DeleteEnv(envId uint64) {
	model.DeleteById(new(entity.ProjectEnv), envId)
}

package persistence

import (
	"mayfly-go/internal/devops/domain/entity"
	"mayfly-go/internal/devops/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type projectEnvRepo struct{}

var ProjectEnvRepo repository.ProjectEnv = &projectEnvRepo{}

func (p *projectEnvRepo) ListEnv(condition *entity.ProjectEnv, toEntity interface{}, orderBy ...string) {
	model.ListByOrder(condition, toEntity, orderBy...)
}

func (p *projectEnvRepo) Save(entity *entity.ProjectEnv) {
	biz.ErrIsNilAppendErr(model.Insert(entity), "保存环境失败：%s")
}

func (p *projectEnvRepo) DeleteEnvs(projectId uint64) {
	model.DeleteByCondition(&entity.ProjectEnv{ProjectId: projectId})
}

func (p *projectEnvRepo) DeleteEnv(envId uint64) {
	model.DeleteById(new(entity.ProjectEnv), envId)
}

package persistence

import (
	"mayfly-go/internal/project/domain/entity"
	"mayfly-go/internal/project/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type projectRepoImpl struct{}

func newProjectRepo() repository.Project {
	return new(projectRepoImpl)
}

func (p *projectRepoImpl) GetPageList(condition *entity.Project, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return model.GetPage(pageParam, condition, toEntity, orderBy...)
}

func (p *projectRepoImpl) Count(condition *entity.Project) int64 {
	return model.CountBy(condition)
}

func (p *projectRepoImpl) GetByIdIn(ids []uint64, toEntity interface{}, orderBy ...string) {
	model.GetByIdIn(new(entity.Project), toEntity, ids, orderBy...)
}

func (p *projectRepoImpl) Save(project *entity.Project) {
	biz.ErrIsNil(model.Insert(project), "保存项目失败")
}

func (p *projectRepoImpl) Update(project *entity.Project) {
	biz.ErrIsNil(model.UpdateById(project), "更新项目信息")
}

func (p *projectRepoImpl) Delete(id uint64) {
	model.DeleteById(new(entity.Project), id)
}

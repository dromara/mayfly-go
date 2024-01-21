package application

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/base"
)

type TagResource interface {
	base.App[*entity.TagResource]

	ListByQuery(condition *entity.TagResourceQuery, toEntity any)
}

type tagResourceAppImpl struct {
	base.AppImpl[*entity.TagResource, repository.TagResource]
}

// 注入TagResourceRepo
func (tr *tagResourceAppImpl) InjectTagResourceRepo(repo repository.TagResource) {
	tr.Repo = repo
}

func (tr *tagResourceAppImpl) ListByQuery(condition *entity.TagResourceQuery, toEntity any) {
	tr.Repo.SelectByCondition(condition, toEntity)
}

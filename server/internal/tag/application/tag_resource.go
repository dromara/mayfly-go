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

func newTagResourceApp(tagResourceRepo repository.TagResource) TagResource {
	tagResourceApp := &tagResourceAppImpl{}
	tagResourceApp.Repo = tagResourceRepo
	return tagResourceApp
}

type tagResourceAppImpl struct {
	base.AppImpl[*entity.TagResource, repository.TagResource]
}

func (tr *tagResourceAppImpl) ListByQuery(condition *entity.TagResourceQuery, toEntity any) {
	tr.Repo.SelectByCondition(condition, toEntity)
}

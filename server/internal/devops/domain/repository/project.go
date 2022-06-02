package repository

import (
	"mayfly-go/internal/devops/domain/entity"
	"mayfly-go/pkg/model"
)

type Project interface {
	GetPageList(condition *entity.Project, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	Count(condition *entity.Project) int64

	GetByIdIn(ids []uint64, toEntity interface{}, orderBy ...string)

	Save(p *entity.Project)

	Update(project *entity.Project)

	Delete(id uint64)
}

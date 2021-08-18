package repository

import (
	"mayfly-go/base/model"
	"mayfly-go/server/devops/domain/entity"
)

type Project interface {
	GetPageList(condition *entity.Project, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	GetByIdIn(ids []uint64, toEntity interface{}, orderBy ...string)

	Save(p *entity.Project)

	Update(project *entity.Project)

	Delete(id uint64)
}

package repository

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/model"
)

type Team interface {
	GetPageList(condition *entity.Team, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult

	Insert(p *entity.Team)

	UpdateById(p *entity.Team)

	Delete(id uint64)

	DeleteBy(p *entity.Team)
}

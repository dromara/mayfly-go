package repository

import (
	"mayfly-go/internal/tag/domain/entity"
)

type TagTree interface {
	// 根据条件查询
	SelectByCondition(condition *entity.TagTreeQuery, toEntity any, orderBy ...string)

	GetBy(condition *entity.TagTree, cols ...string) error

	// 根据主键查询，若不存在返回nil
	SelectById(id uint64) *entity.TagTree

	Insert(p *entity.TagTree)

	UpdateById(p *entity.TagTree)

	Delete(id uint64)
}

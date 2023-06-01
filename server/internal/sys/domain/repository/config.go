package repository

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/model"
)

type Config interface {
	GetPageList(condition *entity.Config, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult

	Insert(config *entity.Config)

	Update(config *entity.Config)

	GetConfig(config *entity.Config, cols ...string) error

	GetByCondition(condition *entity.Config, cols ...string) error
}

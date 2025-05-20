package repository

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type Config interface {
	base.Repo[*entity.Config]

	GetPageList(condition *entity.Config, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.Config], error)
}

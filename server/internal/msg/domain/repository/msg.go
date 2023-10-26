package repository

import (
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type Msg interface {
	base.Repo[*entity.Msg]

	GetPageList(condition *entity.Msg, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}

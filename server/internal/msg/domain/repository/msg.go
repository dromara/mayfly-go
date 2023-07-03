package repository

import (
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/pkg/model"
)

type Msg interface {
	GetPageList(condition *entity.Msg, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any]

	Insert(msg *entity.Msg)
}

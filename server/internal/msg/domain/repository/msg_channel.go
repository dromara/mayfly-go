package repository

import (
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type MsgChannel interface {
	base.Repo[*entity.MsgChannel]

	GetPageList(condition *entity.MsgChannel, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MsgChannel], error)
}

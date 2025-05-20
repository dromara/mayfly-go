package repository

import (
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type MsgTmpl interface {
	base.Repo[*entity.MsgTmpl]

	GetPageList(condition *entity.MsgTmpl, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MsgTmpl], error)
}

type MsgTmplChannel interface {
	base.Repo[*entity.MsgTmplChannel]
}

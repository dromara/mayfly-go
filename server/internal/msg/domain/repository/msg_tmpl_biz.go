package repository

import (
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/pkg/base"
)

type MsgTmplBiz interface {
	base.Repo[*entity.MsgTmplBiz]
}

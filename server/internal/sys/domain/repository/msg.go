package repository

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/model"
)

type Msg interface {
	GetPageList(condition *entity.Msg, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult

	Insert(msg *entity.Msg)
}

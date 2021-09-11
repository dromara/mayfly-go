package repository

import (
	"mayfly-go/base/model"
	"mayfly-go/server/sys/domain/entity"
)

type Msg interface {
	GetPageList(condition *entity.Msg, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	Insert(msg *entity.Msg)
}

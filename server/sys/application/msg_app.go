package application

import (
	"mayfly-go/base/model"
	"mayfly-go/server/sys/domain/entity"
	"mayfly-go/server/sys/domain/repository"
	"mayfly-go/server/sys/infrastructure/persistence"
)

type Msg interface {
	GetPageList(condition *entity.Msg, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	Create(msg *entity.Msg)
}

type msgAppImpl struct {
	msgRepo repository.Msg
}

var MsgApp Msg = &msgAppImpl{
	msgRepo: persistence.MsgDao,
}

func (a *msgAppImpl) GetPageList(condition *entity.Msg, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return a.msgRepo.GetPageList(condition, pageParam, toEntity)
}

func (a *msgAppImpl) Create(msg *entity.Msg) {
	a.msgRepo.Insert(msg)
}

package application

import (
	"mayfly-go/base/model"
	"mayfly-go/base/ws"
	"mayfly-go/server/sys/domain/entity"
	"mayfly-go/server/sys/domain/repository"
	"mayfly-go/server/sys/infrastructure/persistence"
	"time"
)

type Msg interface {
	GetPageList(condition *entity.Msg, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	Create(msg *entity.Msg)

	// 创建消息，并通过ws发送
	CreateAndSend(la *model.LoginAccount, msg *ws.Msg)
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

func (a *msgAppImpl) CreateAndSend(la *model.LoginAccount, wmsg *ws.Msg) {
	now := time.Now()
	msg := &entity.Msg{Type: 2, Msg: wmsg.Msg, RecipientId: int64(la.Id), CreateTime: &now, CreatorId: la.Id, Creator: la.Username}
	a.msgRepo.Insert(msg)
	ws.SendMsg(la.Id, wmsg)
}

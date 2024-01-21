package application

import (
	"context"
	"mayfly-go/internal/msg/application/dto"
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/msg/domain/repository"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/ws"
	"time"
)

type Msg interface {
	GetPageList(condition *entity.Msg, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	Create(ctx context.Context, msg *entity.Msg)

	// 创建消息，并通过ws发送
	CreateAndSend(la *model.LoginAccount, msg *dto.SysMsg)
}

type msgAppImpl struct {
	MsgRepo repository.Msg `inject:""`
}

func (a *msgAppImpl) GetPageList(condition *entity.Msg, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return a.MsgRepo.GetPageList(condition, pageParam, toEntity)
}

func (a *msgAppImpl) Create(ctx context.Context, msg *entity.Msg) {
	a.MsgRepo.Insert(ctx, msg)
}

func (a *msgAppImpl) CreateAndSend(la *model.LoginAccount, wmsg *dto.SysMsg) {
	now := time.Now()
	msg := &entity.Msg{Type: 2, Msg: wmsg.Msg, RecipientId: int64(la.Id), CreateTime: &now, CreatorId: la.Id, Creator: la.Username}
	a.MsgRepo.Insert(context.TODO(), msg)
	ws.SendJsonMsg(ws.UserId(la.Id), wmsg.ClientId, wmsg)
}

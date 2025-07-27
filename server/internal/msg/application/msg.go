package application

import (
	"cmp"
	"context"
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/msg/domain/repository"
	"mayfly-go/internal/msg/msgx"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/stringx"
)

type Msg interface {
	msgx.MsgSender

	base.App[*entity.Msg]

	GetPageList(condition *entity.Msg, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.Msg], error)
}

var _ (Msg) = (*msgAppImpl)(nil)

type msgAppImpl struct {
	base.AppImpl[*entity.Msg, repository.Msg]

	msgRepo repository.Msg `inject:"T"`
}

func (a *msgAppImpl) GetPageList(condition *entity.Msg, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.Msg], error) {
	return a.msgRepo.GetPageList(condition, pageParam)
}

func (a *msgAppImpl) Send(ctx context.Context, channel *msgx.Channel, msg *msgx.Msg) error {
	// 存在i18n msgId，content则使用msgId翻译
	if msgId := msg.TmplExtra.GetInt("msgId"); msgId != 0 {
		msg.Content = i18n.TC(ctx, i18n.MsgId(msgId))
	}
	content, err := stringx.TemplateParse(msg.Content, msg.Params)
	if err != nil {
		return err
	}

	for _, receiver := range msg.Receivers {
		msgEntity := &entity.Msg{
			Msg:         content,
			RecipientId: int64(receiver.Id),
			Type:        entity.MsgType(msg.TmplExtra.GetInt("type")),
			Subtype:     entity.MsgSubtype(msg.TmplExtra.GetStr("subtype")),
			Status:      cmp.Or(entity.MsgStatus(msg.TmplExtra.GetInt("status")), entity.MsgStatusRead),
		}
		msgEntity.Extra = msg.Params
		if err := a.Save(ctx, msgEntity); err != nil {
			return err
		}
	}

	return nil
}

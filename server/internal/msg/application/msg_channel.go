package application

import (
	"context"
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/msg/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/stringx"
)

type MsgChannel interface {
	base.App[*entity.MsgChannel]

	GetPageList(condition *entity.MsgChannel, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MsgChannel], error)

	SaveChannel(ctx context.Context, msgChannel *entity.MsgChannel) error

	DeleteChannel(ctx context.Context, id uint64) error
}

type msgChannelAppImpl struct {
	base.AppImpl[*entity.MsgChannel, repository.MsgChannel]

	msgTempApp MsgTmpl `inject:"T"`
}

var _ (MsgChannel) = (*msgChannelAppImpl)(nil)

func (m *msgChannelAppImpl) GetPageList(condition *entity.MsgChannel, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MsgChannel], error) {
	return m.Repo.GetPageList(condition, pageParam)
}

func (m *msgChannelAppImpl) SaveChannel(ctx context.Context, msgChannel *entity.MsgChannel) error {
	if msgChannel.Id == 0 {
		msgChannel.Code = stringx.Rand(8)
	}
	return m.Save(ctx, msgChannel)
}

func (m *msgChannelAppImpl) DeleteChannel(ctx context.Context, id uint64) error {
	return m.Tx(ctx, func(ctx context.Context) error {
		if err := m.DeleteById(ctx, id); err != nil {
			return err
		}
		// 删除渠道关联的模板
		if err := m.msgTempApp.DeleteTmplChannel(ctx, id); err != nil {
			return err
		}
		return nil
	})
}

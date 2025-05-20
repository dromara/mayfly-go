package application

import (
	"context"
	"mayfly-go/internal/msg/application/dto"
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/msg/domain/repository"
	"mayfly-go/internal/msg/msgx"
	sysapp "mayfly-go/internal/sys/application"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/utils/structx"
)

type MsgTmpl interface {
	base.App[*entity.MsgTmpl]

	GetPageList(condition *entity.MsgTmpl, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MsgTmpl], error)

	SaveTmpl(ctx context.Context, msgTmpl *dto.MsgTmplSave) error

	DeleteTmpl(ctx context.Context, id uint64) error

	GetTmplChannels(ctx context.Context, tmplId uint64) ([]*entity.MsgChannel, error)

	// Send 发送消息
	Send(ctx context.Context, tmplCode string, params map[string]any, receiverId ...uint64) error

	// DeleteTmplChannel 删除指定渠道关联的模板
	DeleteTmplChannel(ctx context.Context, channelId uint64) error
}

type msgTmplAppImpl struct {
	base.AppImpl[*entity.MsgTmpl, repository.MsgTmpl]

	msgTmplChannelRepo repository.MsgTmplChannel `inject:"T"`

	msgChannelApp MsgChannel     `inject:"T"`
	msgTmplBizApp MsgTmplBiz     `inject:"T"`
	accountApp    sysapp.Account `inject:"T"`
}

var _ (MsgTmpl) = (*msgTmplAppImpl)(nil)

func (m *msgTmplAppImpl) GetPageList(condition *entity.MsgTmpl, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MsgTmpl], error) {
	return m.Repo.GetPageList(condition, pageParam)
}

func (m *msgTmplAppImpl) SaveTmpl(ctx context.Context, msgTmpl *dto.MsgTmplSave) error {
	return m.Tx(ctx, func(ctx context.Context) error {
		mt := &entity.MsgTmpl{}
		structx.Copy(mt, msgTmpl)
		isCreate := mt.Id == 0
		if isCreate {
			mt.Code = stringx.Rand(8)
		}

		if err := m.Save(ctx, mt); err != nil {
			return err
		}

		oldTemplChannelIds := []uint64{}
		if !isCreate {
			oldTemplChannels, err := m.msgTmplChannelRepo.SelectByCond(&entity.MsgTmplChannel{TmplId: mt.Id}, "channel_id")
			if err != nil {
				return err
			}

			oldTemplChannelIds = collx.ArrayMap(oldTemplChannels, func(c *entity.MsgTmplChannel) uint64 {
				return c.ChannelId
			})
		}

		add, del, _ := collx.ArrayCompare(msgTmpl.ChannelIds, oldTemplChannelIds)
		if len(add) > 0 {
			tmplChannels := collx.ArrayMap(msgTmpl.ChannelIds, func(channelId uint64) *entity.MsgTmplChannel {
				return &entity.MsgTmplChannel{
					ChannelId: channelId,
					TmplId:    mt.Id,
				}
			})
			if err := m.msgTmplChannelRepo.BatchInsert(ctx, tmplChannels); err != nil {
				return err
			}
		}

		if len(del) > 0 {
			if err := m.msgTmplChannelRepo.DeleteByCond(ctx, model.NewCond().Eq("tmpl_id", mt.Id).In("channel_id", del)); err != nil {
				return err
			}
		}

		return nil
	})
}

func (m *msgTmplAppImpl) DeleteTmpl(ctx context.Context, id uint64) error {
	return m.Tx(ctx, func(ctx context.Context) error {
		if err := m.DeleteById(ctx, id); err != nil {
			return err
		}

		if err := m.msgTmplBizApp.DeleteByTmplId(ctx, id); err != nil {
			return err
		}

		return m.msgTmplChannelRepo.DeleteByCond(ctx, &entity.MsgTmplChannel{TmplId: id})
	})
}

func (m *msgTmplAppImpl) GetTmplChannels(ctx context.Context, tmplId uint64) ([]*entity.MsgChannel, error) {
	tmplChannels, err := m.msgTmplChannelRepo.SelectByCond(&entity.MsgTmplChannel{TmplId: tmplId}, "channel_id")
	if err != nil {
		return nil, err
	}
	if len(tmplChannels) == 0 {
		return []*entity.MsgChannel{}, nil
	}

	return m.msgChannelApp.GetByIds(collx.ArrayMap(tmplChannels, func(c *entity.MsgTmplChannel) uint64 {
		return c.ChannelId
	}))
}

func (m *msgTmplAppImpl) Send(ctx context.Context, tmplCode string, params map[string]any, receiverId ...uint64) error {
	tmpl := &entity.MsgTmpl{Code: tmplCode}
	err := m.GetByCond(tmpl)
	if err != nil {
		return errorx.NewBiz("message template does not exist")
	}
	if tmpl.Status != entity.TmplStatusEnable {
		return errorx.NewBiz("message template is disabled")
	}

	tmplChannels, err := m.msgTmplChannelRepo.SelectByCond(&entity.MsgTmplChannel{TmplId: tmpl.Id}, "channel_id")
	if err != nil {
		return err
	}
	if len(tmplChannels) == 0 {
		return errorx.NewBiz("message template is not associated with any channel")
	}

	channels, err := m.msgChannelApp.GetByIds(collx.ArrayMap(tmplChannels, func(c *entity.MsgTmplChannel) uint64 {
		return c.ChannelId
	}))

	if err != nil {
		return err
	}

	// content, err := stringx.TemplateParse(tmpl.Tmpl, params)
	// if err != nil {
	// 	return err
	// }

	// toAll := len(receiverId) == 0
	accounts, err := m.accountApp.GetByIds(receiverId)
	if err != nil {
		return err
	}

	msg := &msgx.Msg{
		Content:   tmpl.Tmpl,
		Params:    params,
		Title:     tmpl.Title,
		Type:      tmpl.MsgType,
		ExtraData: tmpl.ExtraData,
	}

	if len(accounts) > 0 {
		msg.Receivers = collx.ArrayMap(accounts, func(account *sysentity.Account) msgx.Receiver {
			return msgx.Receiver{
				ExtraData: account.ExtraData,
				Email:     account.Email,
				Mobile:    account.Mobile,
			}
		})
	}

	for _, channel := range channels {
		if channel.Status != entity.ChannelStatusEnable {
			logx.Warnf("channel is disabled => %s", channel.Code)
			continue
		}

		go func(channel *entity.MsgChannel) {
			if err := msgx.Send(&msgx.Channel{
				Type:      channel.Type,
				Name:      channel.Name,
				URL:       channel.Url,
				ExtraData: channel.ExtraData,
			}, msg); err != nil {
				logx.Errorf("send msg error => channel=%s, msg=%s, err -> %v", channel.Code, msg.Content, err)
			}
		}(channel)
	}

	return nil
}

func (m *msgTmplAppImpl) DeleteTmplChannel(ctx context.Context, channelId uint64) error {
	return m.msgTmplChannelRepo.DeleteByCond(ctx, &entity.MsgTmplChannel{ChannelId: channelId})
}

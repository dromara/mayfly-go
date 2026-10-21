package application

import (
	"context"
	"mayfly-go/internal/msg/application/dto"
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/msg/domain/repository"
	"mayfly-go/internal/msg/imsg"
	"mayfly-go/internal/msg/msgx"
	sysapp "mayfly-go/internal/sys/application"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gox"
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

	// CheckSendable 校验模板当前是否可发（存在 / 已启用 / 已关联至少一个渠道），
	// 不可发时返回按请求语言译好的业务错误（供调用方直接当 toast 文案用）。
	//
	// 存在动机（真机实测）：触发器/任务/机器计划任务的「结果通知模板」过去只在保存时校验
	// 「非空」，模板已停用或未关联渠道也能存进去，直到运行收尾才在服务端日志里留一条
	// ERROR——界面上「操作成功」，通知这条链路的最后一跳却永远不通，配错的人看不到任何信号。
	// 判据与 Send 共用 loadSendableTmpl，不会出现「保存时放行、发送时才拒」的漂移
	CheckSendable(ctx context.Context, tmplCode string) error

	// CheckSendableById 按模板 ID 校验可发性，判据与 CheckSendable / Send 完全一致。
	//
	// 为何除了按编码还要一个按 ID 的入口：模板-业务关联（流程定义的待办通知等）按 ID 绑定模板，
	// 它拿不到编码。两个入口必须汇到同一份判据上，否则就会出现「按编码查会被拒、
	// 按 ID 绑却能绑上」的绕过路径
	CheckSendableById(ctx context.Context, tmplId uint64) error

	// Send 发送消息
	Send(ctx context.Context, tmplCode string, params map[string]any, receiverId ...uint64) error

	SendMsg(ctx context.Context, mts *dto.MsgTmplSend) error

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

var _ MsgTmpl = (*msgTmplAppImpl)(nil)

var _ (MsgTmpl) = (*msgTmplAppImpl)(nil)

func (m *msgTmplAppImpl) GetPageList(condition *entity.MsgTmpl, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MsgTmpl], error) {
	return m.Repo.GetPageList(condition, pageParam)
}

func (m *msgTmplAppImpl) SaveTmpl(ctx context.Context, msgTmpl *dto.MsgTmplSave) error {
	// 事务前先取旧模板的渠道：它既是差异比较的基线，又必须留在事务**之外**——基座读入口不收
	// ctx，走的是另一条连接：事务内读既看不到本事务尚未提交的写入，也会在事务占满连接池时等不到
	// 空闲连接（部署把 max-open-conns 调小后就是长时间不返回）。
	// 这里读的是已提交的旧状态，与本事务要写入的内容无依赖，因此提前读不改变结果
	oldTemplChannelIds := []uint64{}
	if msgTmpl.Id != 0 {
		oldTemplChannels, err := m.msgTmplChannelRepo.SelectByCond(&entity.MsgTmplChannel{TmplId: msgTmpl.Id}, "channel_id")
		if err != nil {
			return err
		}
		oldTemplChannelIds = collx.ArrayMap(oldTemplChannels, func(c *entity.MsgTmplChannel) uint64 {
			return c.ChannelId
		})
	}

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

// CheckSendable 见 MsgTmpl 接口说明：与 Send 共用 loadSendableTmpl，保存期与发送期判据必然一致
func (m *msgTmplAppImpl) CheckSendable(ctx context.Context, tmplCode string) error {
	_, _, err := m.loadSendableTmpl(ctx, tmplCode)
	return err
}

// CheckSendableById 见 MsgTmpl 接口说明：按 ID 取模板后走同一份 checkSendable 判据
func (m *msgTmplAppImpl) CheckSendableById(ctx context.Context, tmplId uint64) error {
	tmpl, err := m.GetById(tmplId)
	if err != nil {
		return errorx.NewBizI(ctx, imsg.ErrMsgTmplIdNotExist, "id", tmplId)
	}
	_, err = m.checkSendable(ctx, tmpl)
	return err
}

// loadSendableTmpl 按编码取出「现在就能发」的模板及其关联渠道 id
func (m *msgTmplAppImpl) loadSendableTmpl(ctx context.Context, tmplCode string) (*entity.MsgTmpl, []uint64, error) {
	tmpl := &entity.MsgTmpl{Code: tmplCode}
	if err := m.GetByCond(tmpl); err != nil {
		return nil, nil, errorx.NewBizI(ctx, imsg.ErrMsgTmplNotExist, "code", tmplCode)
	}

	channelIds, err := m.checkSendable(ctx, tmpl)
	if err != nil {
		return nil, nil, err
	}
	return tmpl, channelIds, nil
}

// checkSendable 可发性判据的唯一实现：已启用 + 已关联至少一个渠道。
// Send / CheckSendable / CheckSendableById 全部经此，不可能出现「某个入口少查一项」的漂移。
// 文案走 imsg 按请求 ctx 的语言生成（会以 toast 直接呈现给配置人，硬编码英文字面量
// 会让中文界面冒出英文串）
func (m *msgTmplAppImpl) checkSendable(ctx context.Context, tmpl *entity.MsgTmpl) ([]uint64, error) {
	if tmpl.Status != entity.TmplStatusEnable {
		return nil, errorx.NewBizI(ctx, imsg.ErrMsgTmplDisabled, "code", tmpl.Code)
	}

	tmplChannels, err := m.msgTmplChannelRepo.SelectByCond(&entity.MsgTmplChannel{TmplId: tmpl.Id}, "channel_id")
	if err != nil {
		return nil, err
	}
	if len(tmplChannels) == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrMsgTmplNoChannel, "code", tmpl.Code)
	}

	return collx.ArrayMap(tmplChannels, func(c *entity.MsgTmplChannel) uint64 {
		return c.ChannelId
	}), nil
}

func (m *msgTmplAppImpl) Send(ctx context.Context, tmplCode string, params map[string]any, receiverId ...uint64) error {
	tmpl, channelIds, err := m.loadSendableTmpl(ctx, tmplCode)
	if err != nil {
		return err
	}

	channels, err := m.msgChannelApp.GetByIds(channelIds)
	if err != nil {
		return err
	}

	return m.SendMsg(ctx, &dto.MsgTmplSend{
		Tmpl:        tmpl,
		Channels:    channels,
		Params:      params,
		ReceiverIds: receiverId,
	})
}

func (m *msgTmplAppImpl) SendMsg(ctx context.Context, mts *dto.MsgTmplSend) error {
	tmpl := mts.Tmpl
	msg := &msgx.Msg{
		Content:   tmpl.Tmpl,
		Params:    mts.Params,
		Title:     tmpl.Title,
		Type:      tmpl.MsgType,
		TmplExtra: tmpl.Extra,
	}

	accounts, err := m.accountApp.GetByIds(mts.ReceiverIds)
	if err != nil {
		return err
	}

	if len(accounts) > 0 {
		msg.Receivers = collx.ArrayMap(accounts, func(account *sysentity.Account) msgx.Receiver {
			return msgx.Receiver{
				Id:     account.Id,
				Extra:  account.Extra,
				Email:  account.Email,
				Mobile: account.Mobile,
			}
		})
	}

	for _, channel := range mts.Channels {
		if channel.Status != entity.ChannelStatusEnable {
			logx.Warnf("channel is disabled => %s", channel.Code)
			continue
		}

		gox.Go(func() {
			ch := *channel
			if err := msgx.Send(ctx, &msgx.Channel{
				Type:  ch.Type,
				Name:  ch.Name,
				URL:   ch.Url,
				Extra: ch.Extra,
			}, msg); err != nil {
				logx.Errorf("send msg error => channel=%s, msg=%s, err -> %v", ch.Code, msg.Content, err)
			}
		})
	}

	return nil
}

func (m *msgTmplAppImpl) DeleteTmplChannel(ctx context.Context, channelId uint64) error {
	return m.msgTmplChannelRepo.DeleteByCond(ctx, &entity.MsgTmplChannel{ChannelId: channelId})
}

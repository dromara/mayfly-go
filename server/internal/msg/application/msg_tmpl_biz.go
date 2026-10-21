package application

import (
	"context"
	"mayfly-go/internal/msg/application/dto"
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/msg/domain/repository"
	"mayfly-go/internal/msg/imsg"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
)

type MsgTmplBiz interface {
	base.App[*entity.MsgTmplBiz]

	// SaveBizTmpl 保存消息模板关联业务信息
	SaveBizTmpl(ctx context.Context, bizTmpl *dto.MsgTmplBizSave) error

	// DeleteByBiz 根据业务删除消息模板业务关联
	DeleteByBiz(ctx context.Context, bizType string, bizId uint64) error

	// DeleteByTmplId 根据模板ID删除消息模板业务关联
	DeleteByTmplId(ctx context.Context, tmplId uint64) error

	// Send 发送消息
	Send(ctx context.Context, sendParam *dto.BizMsgTmplSend) error
}

type msgTmplBizAppImpl struct {
	base.AppImpl[*entity.MsgTmplBiz, repository.MsgTmplBiz]

	msgTmplApp MsgTmpl `inject:"T"`
}

var _ MsgTmplBiz = (*msgTmplBizAppImpl)(nil)

var _ (MsgTmplBiz) = (*msgTmplBizAppImpl)(nil)

func (m *msgTmplBizAppImpl) SaveBizTmpl(ctx context.Context, bizTmpl *dto.MsgTmplBizSave) error {
	msgTmplId := bizTmpl.TmplId
	bizId := bizTmpl.BizId
	bizType := bizTmpl.BizType
	if bizId == 0 {
		return errorx.NewBizI(ctx, imsg.ErrMsgTmplBizIdRequired)
	}
	if bizType == "" {
		return errorx.NewBizI(ctx, imsg.ErrMsgTmplBizTypeRequired)
	}

	msgTmplBiz := &entity.MsgTmplBiz{
		BizId:   bizId,
		BizType: bizType,
	}
	bound := m.GetByCond(msgTmplBiz) == nil

	// 绑定未变：不重复写也不校验（否则改业务侧任何无关字段都会被一条早已存在的坏绑定拦下）
	if bound && msgTmplBiz.TmplId == msgTmplId {
		return nil
	}
	// 模板 ID 为 0 表示解绑（本来就没绑时是空操作）
	if msgTmplId == 0 {
		if bound {
			return m.DeleteByBiz(ctx, bizType, bizId)
		}
		return nil
	}

	// 走到这里必然要写入一个非 0 的新模板，先确认它发得出去。
	// 校验点只有这一个（而不是新建/变更两条分支各写一遍）：「任何会落库的模板绑定都被校验过」
	// 因此是结构上的必然，而不是靠调用方自觉
	if err := m.checkTmplSendable(ctx, msgTmplId); err != nil {
		return err
	}

	if bound {
		update := &entity.MsgTmplBiz{
			TmplId: msgTmplId,
		}
		update.Id = msgTmplBiz.Id
		return m.UpdateById(ctx, update)
	}

	msgTmplBiz.TmplId = msgTmplId
	return m.Save(ctx, msgTmplBiz)
}

// checkTmplSendable 绑定前确认模板当前发得出去（存在 / 已启用 / 已关联渠道）。
//
// 与 agentops 任务、触发器、机器计划任务的保存校验同一取向：能在保存期拦住的配置错误，
// 不留到发送期只写日志。这里的发送期尤其看不见——业务模板消息经 EventBus 异步派发，
// 订阅方见 msg/init 对 EventTopicBizMsgTmplSend 的注册；失败只回到 eventbus 的日志里，
// 界面上流程定义早已「保存成功」，而待办人永远收不到通知。
// 实测形态：t_msg_tmpl_biz 里 flow:task:notify 绑的模板在 t_msg_tmpl_channel 里没有任何关联行
func (m *msgTmplBizAppImpl) checkTmplSendable(ctx context.Context, tmplId uint64) error {
	// 单测绕过 IOC 装配时 msgTmplApp 为 nil：跳过校验而不是 panic，
	// 否则一个「只是没法校验」的依赖会把整条业务保存主链拖死
	if m.msgTmplApp == nil {
		return nil
	}
	return m.msgTmplApp.CheckSendableById(ctx, tmplId)
}

func (m *msgTmplBizAppImpl) DeleteByBiz(ctx context.Context, bizType string, bizId uint64) error {
	return m.DeleteByCond(ctx, &entity.MsgTmplBiz{BizId: bizId, BizType: bizType})
}

func (m *msgTmplBizAppImpl) DeleteByTmplId(ctx context.Context, tmplId uint64) error {
	return m.DeleteByCond(ctx, &entity.MsgTmplBiz{TmplId: tmplId})
}

func (m *msgTmplBizAppImpl) Send(ctx context.Context, sendParam *dto.BizMsgTmplSend) error {
	// 获取业务关联的消息模板
	msgTmplBiz := &entity.MsgTmplBiz{
		BizId:   sendParam.BizId,
		BizType: sendParam.BizType,
	}
	if err := m.GetByCond(msgTmplBiz); err != nil {
		return errorx.NewBizI(ctx, imsg.ErrMsgTmplBizNotBound)
	}

	mstTmpl, err := m.msgTmplApp.GetById(msgTmplBiz.TmplId)
	if err != nil {
		return errorx.NewBizI(ctx, imsg.ErrMsgTmplIdNotExist, "id", msgTmplBiz.TmplId)
	}

	return m.msgTmplApp.Send(ctx, mstTmpl.Code, sendParam.Params, sendParam.ReceiverIds...)
}

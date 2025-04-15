package application

import (
	"context"
	"mayfly-go/internal/msg/application/dto"
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/msg/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
)

type MsgTmplBiz interface {
	base.App[*entity.MsgTmplBiz]

	// SaveBizTmpl 保存消息模板关联业务信息
	SaveBizTmpl(ctx context.Context, bizTmpl dto.MsgTmplBizSave) error

	// DeleteByBiz 根据业务删除消息模板业务关联
	DeleteByBiz(ctx context.Context, bizType string, bizId uint64) error

	// DeleteByTmplId 根据模板ID删除消息模板业务关联
	DeleteByTmplId(ctx context.Context, tmplId uint64) error

	// Send 发送消息
	Send(ctx context.Context, sendParam dto.BizMsgTmplSend) error
}

type msgTmplBizAppImpl struct {
	base.AppImpl[*entity.MsgTmplBiz, repository.MsgTmplBiz]

	msgTmplApp MsgTmpl `inject:"T"`
}

var _ (MsgTmplBiz) = (*msgTmplBizAppImpl)(nil)

func (m *msgTmplBizAppImpl) SaveBizTmpl(ctx context.Context, bizTmpl dto.MsgTmplBizSave) error {
	msgTmplId := bizTmpl.TmplId
	bizId := bizTmpl.BizId
	bizType := bizTmpl.BizType
	if bizId == 0 {
		return errorx.NewBiz("business ID cannot be empty")
	}
	if bizType == "" {
		return errorx.NewBiz("business type cannot be empty")
	}

	msgTmplBiz := &entity.MsgTmplBiz{
		BizId:   bizId,
		BizType: bizType,
	}
	// exist
	if err := m.GetByCond(msgTmplBiz); err == nil {
		// tmplId不变，直接返回即可
		if msgTmplBiz.TmplId == msgTmplId {
			return nil
		}

		// 如果模板ID为0，表示删除业务关联
		if msgTmplId == 0 {
			return m.DeleteByBiz(ctx, bizTmpl.BizType, bizTmpl.BizId)
		}

		update := &entity.MsgTmplBiz{
			TmplId: msgTmplId,
		}
		update.Id = msgTmplBiz.Id
		return m.UpdateById(ctx, update)
	}

	if msgTmplId == 0 {
		return nil
	}

	msgTmplBiz.TmplId = msgTmplId
	return m.Save(ctx, msgTmplBiz)
}

func (m *msgTmplBizAppImpl) DeleteByBiz(ctx context.Context, bizType string, bizId uint64) error {
	return m.DeleteByCond(ctx, &entity.MsgTmplBiz{BizId: bizId, BizType: bizType})
}

func (m *msgTmplBizAppImpl) DeleteByTmplId(ctx context.Context, tmplId uint64) error {
	return m.DeleteByCond(ctx, &entity.MsgTmplBiz{TmplId: tmplId})
}

func (m *msgTmplBizAppImpl) Send(ctx context.Context, sendParam dto.BizMsgTmplSend) error {
	// 获取业务关联的消息模板
	msgTmplBiz := &entity.MsgTmplBiz{
		BizId:   sendParam.BizId,
		BizType: sendParam.BizType,
	}
	if err := m.GetByCond(msgTmplBiz); err != nil {
		return errorx.NewBiz("message tmplate association business information does not exist")
	}

	mstTmpl, err := m.msgTmplApp.GetById(msgTmplBiz.TmplId)
	if err != nil {
		return errorx.NewBiz("message template does not exist")
	}

	return m.msgTmplApp.Send(ctx, mstTmpl.Code, sendParam.Params, sendParam.ReceiverIds...)
}

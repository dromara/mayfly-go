package api

import (
	"mayfly-go/internal/msg/application"
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

type Msg struct {
	msgApp application.Msg `inject:"T"`
}

func (m *Msg) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("/self", m.GetMsgs),
		req.NewGet("/self/unread/count", m.GetUnreadCount),
		req.NewGet("/self/read", m.ReadMsg),
	}

	return req.NewConfs("/msgs", reqs[:]...)
}

// GetMsgs 获取账号接收的消息列表
func (m *Msg) GetMsgs(rc *req.Ctx) {
	condition := &entity.Msg{
		RecipientId: int64(rc.GetLoginAccount().Id),
	}
	res, err := m.msgApp.GetPageList(condition, rc.GetPageParam())
	biz.ErrIsNil(err)
	rc.ResData = res
}

// GetUnreadCount 获取账号接收的未读消息数量
func (m *Msg) GetUnreadCount(rc *req.Ctx) {
	condition := &entity.Msg{
		RecipientId: int64(rc.GetLoginAccount().Id),
		Status:      entity.MsgStatusUnRead,
	}
	rc.ResData = m.msgApp.CountByCond(condition)
}

// ReadMsg 将账号接收的未读消息标记为已读
func (m *Msg) ReadMsg(rc *req.Ctx) {
	cond := &entity.Msg{
		RecipientId: int64(rc.GetLoginAccount().Id),
		Status:      entity.MsgStatusUnRead,
	}
	cond.Id = uint64(rc.QueryInt("id"))

	biz.ErrIsNil(m.msgApp.UpdateByCond(rc.MetaCtx, &entity.Msg{
		Status: entity.MsgStatusRead,
	}, cond))
}

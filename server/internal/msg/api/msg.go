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
	}

	return req.NewConfs("/msgs", reqs[:]...)
}

// 获取账号接收的消息列表
func (m *Msg) GetMsgs(rc *req.Ctx) {
	condition := &entity.Msg{
		RecipientId: int64(rc.GetLoginAccount().Id),
	}
	res, err := m.msgApp.GetPageList(condition, rc.GetPageParam())
	biz.ErrIsNil(err)
	rc.ResData = res
}

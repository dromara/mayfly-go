package api

import (
	"mayfly-go/internal/msg/application"
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
)

type Msg struct {
	MsgApp application.Msg `inject:""`
}

// 获取账号接收的消息列表
func (m *Msg) GetMsgs(rc *req.Ctx) {
	condition := &entity.Msg{
		RecipientId: int64(rc.GetLoginAccount().Id),
	}
	res, err := m.MsgApp.GetPageList(condition, ginx.GetPageParam(rc.GinCtx), new([]entity.Msg))
	biz.ErrIsNil(err)
	rc.ResData = res
}

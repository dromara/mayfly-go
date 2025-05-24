package api

import (
	"mayfly-go/internal/msg/api/form"
	"mayfly-go/internal/msg/application"
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/msg/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"strings"

	"github.com/may-fly/cast"
)

type MsgChannel struct {
	msgChannelApp application.MsgChannel `inject:"T"`
}

func (m *MsgChannel) ReqConfs() *req.Confs {
	basePermCode := "msg:channel:base"

	reqs := [...]*req.Conf{
		req.NewGet("", m.GetMsgChannels).RequiredPermissionCode(basePermCode),
		req.NewPost("", m.SaveMsgChannels).Log(req.NewLogSaveI(imsg.LogMsgChannelSave)).RequiredPermissionCode("msg:channel:save"),
		req.NewDelete("", m.DelMsgChannels).Log(req.NewLogSaveI(imsg.LogMsgChannelDelete)).RequiredPermissionCode("msg:channel:del"),
	}

	return req.NewConfs("/msg/channels", reqs[:]...)
}

func (m *MsgChannel) GetMsgChannels(rc *req.Ctx) {
	condition := &entity.MsgChannel{}
	res, err := m.msgChannelApp.GetPageList(condition, rc.GetPageParam())
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (m *MsgChannel) SaveMsgChannels(rc *req.Ctx) {
	form, channel := req.BindJsonAndCopyTo[*form.MsgChannel, *entity.MsgChannel](rc)
	rc.ReqParam = form
	err := m.msgChannelApp.SaveChannel(rc.MetaCtx, channel)
	biz.ErrIsNil(err)
}

func (m *MsgChannel) DelMsgChannels(rc *req.Ctx) {
	idsStr := rc.Query("id")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		biz.ErrIsNil(m.msgChannelApp.DeleteChannel(rc.MetaCtx, cast.ToUint64(v)))
	}
}

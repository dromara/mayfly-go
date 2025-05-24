package api

import (
	"mayfly-go/internal/msg/api/form"
	"mayfly-go/internal/msg/application"
	"mayfly-go/internal/msg/application/dto"
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/msg/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"strings"

	"github.com/may-fly/cast"
)

type MsgTmpl struct {
	msgTmplApp application.MsgTmpl `inject:"T"`
}

func (m *MsgTmpl) ReqConfs() *req.Confs {
	basePermCode := "msg:tmpl:base"

	reqs := [...]*req.Conf{
		req.NewGet("", m.GetMsgTmpls).RequiredPermissionCode(basePermCode),
		req.NewGet(":id/channels", m.GetMsgTmplChannels).RequiredPermissionCode(basePermCode),
		req.NewPost("", m.SaveMsgTmpl).Log(req.NewLogSaveI(imsg.LogMsgTmplSave)).RequiredPermissionCode("msg:tmpl:save"),
		req.NewDelete("", m.DelMsgTmpls).Log(req.NewLogSaveI(imsg.LogMsgTmplDelete)).RequiredPermissionCode("msg:tmpl:del"),
		req.NewPost(":code/send", m.SendMsg).Log(req.NewLogSaveI(imsg.LogMsgTmplSave)).RequiredPermissionCode("msg:tmpl:send"),
	}

	return req.NewConfs("/msg/tmpls", reqs[:]...)
}

func (m *MsgTmpl) GetMsgTmpls(rc *req.Ctx) {
	condition := &entity.MsgTmpl{
		Code: rc.Query("code"),
	}
	condition.Id = cast.ToUint64(rc.QueryInt("id"))
	res, err := m.msgTmplApp.GetPageList(condition, rc.GetPageParam())
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (m *MsgTmpl) GetMsgTmplChannels(rc *req.Ctx) {
	channels, err := m.msgTmplApp.GetTmplChannels(rc.MetaCtx, cast.ToUint64(rc.PathParamInt("id")))
	biz.ErrIsNil(err)
	rc.ResData = collx.ArrayMap(channels, func(val *entity.MsgChannel) collx.M {
		return collx.M{
			"id":   val.Id,
			"name": val.Name,
			"type": val.Type,
			"code": val.Code,
		}
	})
}

func (m *MsgTmpl) SaveMsgTmpl(rc *req.Ctx) {
	form, channel := req.BindJsonAndCopyTo[*form.MsgTmpl, *dto.MsgTmplSave](rc)
	rc.ReqParam = form
	biz.ErrIsNil(m.msgTmplApp.SaveTmpl(rc.MetaCtx, channel))
}

func (m *MsgTmpl) DelMsgTmpls(rc *req.Ctx) {
	idsStr := rc.Query("id")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		biz.ErrIsNil(m.msgTmplApp.DeleteTmpl(rc.MetaCtx, cast.ToUint64(v)))
	}
}

func (m *MsgTmpl) SendMsg(rc *req.Ctx) {
	code := rc.PathParam("code")
	form := req.BindJsonAndValid[*form.SendMsg](rc)

	rc.ReqParam = form

	params, err := jsonx.ToMap(form.Parmas)
	biz.ErrIsNil(err)
	biz.ErrIsNil(m.msgTmplApp.Send(rc.MetaCtx, code, params, form.ReceiverIds...))
}

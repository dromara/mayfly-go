package api

import (
	"mayfly-go/internal/alert/application"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

type AlertEvent struct {
	eventApp    application.AlertEvent    `inject:"T"`
	overviewApp application.AlertOverview `inject:"T"`
}

func (a *AlertEvent) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("", a.List),
		req.NewPut(":id/ack", a.Ack).Log(req.NewLogSaveI(imsg.LogAlertEventAck)),
		req.NewPut(":id/close", a.Close).Log(req.NewLogSaveI(imsg.LogAlertEventClose)),
		req.NewDelete(":id", a.Delete).Log(req.NewLogSaveI(imsg.LogAlertEventDelete)),
		req.NewGet("overview", a.Overview),
	}
	return req.NewConfs("alert-events", reqs[:]...)
}

func (a *AlertEvent) List(rc *req.Ctx) {
	condition := rc.BindQuery[entity.AlertEventQuery]()
	res, err := a.eventApp.GetAlertEventList(condition)
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (a *AlertEvent) Ack(rc *req.Ctx) {
	eventId := uint64(rc.PathParamInt("id"))
	userId := rc.GetLoginAccount().Id
	biz.ErrIsNil(a.eventApp.Ack(rc.MetaCtx, eventId, int64(userId)))
}

func (a *AlertEvent) Close(rc *req.Ctx) {
	eventId := uint64(rc.PathParamInt("id"))
	biz.ErrIsNil(a.eventApp.Close(rc.MetaCtx, eventId))
}

func (a *AlertEvent) Delete(rc *req.Ctx) {
	eventId := uint64(rc.PathParamInt("id"))
	biz.ErrIsNil(a.eventApp.DeleteEvent(rc.MetaCtx, eventId))
}

func (a *AlertEvent) Overview(rc *req.Ctx) {
	data, err := a.overviewApp.GetOverview(rc.MetaCtx)
	biz.ErrIsNil(err)
	rc.ResData = data
}

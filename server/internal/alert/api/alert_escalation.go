package api

import (
	"mayfly-go/internal/alert/application"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type AlertEscalation struct {
	escalationApp application.AlertEscalation `inject:"T"`
}

func (a *AlertEscalation) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("", a.List),
		req.NewGet(":id", a.GetById),
		req.NewPost("", a.Save).Log(req.NewLogSaveI(imsg.LogAlertEscalationSave)),
		req.NewPut(":id", a.Update).Log(req.NewLogSaveI(imsg.LogAlertEscalationSave)),
		req.NewPut(":id/:status", a.ChangeStatus).Log(req.NewLogSaveI(imsg.LogAlertEscalationChangeStatus)),
		req.NewDelete(":id", a.Delete).Log(req.NewLogSaveI(imsg.LogAlertEscalationDelete)),
	}
	return req.NewConfs("alert-escalations", reqs[:]...)
}

func (a *AlertEscalation) List(rc *req.Ctx) {
	condition := rc.BindQuery[entity.AlertEscalationQuery]()
	res, err := a.escalationApp.GetAlertEscalationList(condition)
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (a *AlertEscalation) GetById(rc *req.Ctx) {
	id := uint64(rc.PathParamInt("id"))
	esc, err := a.escalationApp.GetById(id)
	biz.ErrIsNil(err)
	rc.ResData = esc
}

func (a *AlertEscalation) Save(rc *req.Ctx) {
	escalation := rc.BindJson[entity.AlertEscalation]()
	biz.ErrIsNil(a.escalationApp.SaveAlertEscalation(rc.MetaCtx, escalation))
	rc.ResData = escalation.Id
}

func (a *AlertEscalation) Update(rc *req.Ctx) {
	escalation := rc.BindJson[entity.AlertEscalation]()
	escalation.Id = uint64(rc.PathParamInt("id"))
	biz.ErrIsNil(a.escalationApp.SaveAlertEscalation(rc.MetaCtx, escalation))
}

func (a *AlertEscalation) ChangeStatus(rc *req.Ctx) {
	id := uint64(rc.PathParamInt("id"))
	status := int8(rc.PathParamInt("status"))
	biz.IsTrueBy(status == entity.AlertEscalationStatusEnable || status == entity.AlertEscalationStatusDisable,
		errorx.NewBizI(rc.MetaCtx, imsg.ErrEscalationStatusInvalid))
	rc.ReqParam = collx.Kvs("id", id, "status", status)
	esc, err := a.escalationApp.GetById(id)
	biz.ErrIsNil(err)
	esc.Status = status
	biz.ErrIsNil(a.escalationApp.SaveAlertEscalation(rc.MetaCtx, esc))
}

func (a *AlertEscalation) Delete(rc *req.Ctx) {
	idsStr := rc.PathParam("id")
	rc.ReqParam = idsStr
	for _, id := range parseCommaIds(idsStr) {
		biz.ErrIsNil(a.escalationApp.DeleteEscalation(rc.MetaCtx, id))
	}
}

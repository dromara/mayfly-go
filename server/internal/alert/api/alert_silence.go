package api

import (
	"mayfly-go/internal/alert/application"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type AlertSilence struct {
	silenceApp application.AlertSilence `inject:"T"`
}

func (a *AlertSilence) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("", a.List),
		req.NewGet(":id", a.GetById),
		req.NewPost("", a.Save).Log(req.NewLogSaveI(imsg.LogAlertSilenceSave)),
		req.NewPut(":id", a.Update).Log(req.NewLogSaveI(imsg.LogAlertSilenceSave)),
		req.NewPut(":id/:status", a.ChangeStatus).Log(req.NewLogSaveI(imsg.LogAlertSilenceChangeStatus)),
		req.NewDelete(":id", a.Delete).Log(req.NewLogSaveI(imsg.LogAlertSilenceDelete)),
	}
	return req.NewConfs("alert-silences", reqs[:]...)
}

func (a *AlertSilence) List(rc *req.Ctx) {
	condition := rc.BindQuery[entity.AlertSilenceQuery]()
	res, err := a.silenceApp.GetAlertSilenceList(condition)
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (a *AlertSilence) GetById(rc *req.Ctx) {
	id := uint64(rc.PathParamInt("id"))
	silence, err := a.silenceApp.GetById(id)
	biz.ErrIsNil(err)
	rc.ResData = silence
}

func (a *AlertSilence) Save(rc *req.Ctx) {
	silence := rc.BindJson[entity.AlertSilence]()
	biz.ErrIsNil(a.silenceApp.SaveAlertSilence(rc.MetaCtx, silence))
	rc.ResData = silence.Id
}

func (a *AlertSilence) Update(rc *req.Ctx) {
	silence := rc.BindJson[entity.AlertSilence]()
	silence.Id = uint64(rc.PathParamInt("id"))
	biz.ErrIsNil(a.silenceApp.SaveAlertSilence(rc.MetaCtx, silence))
}

func (a *AlertSilence) ChangeStatus(rc *req.Ctx) {
	id := uint64(rc.PathParamInt("id"))
	status := int8(rc.PathParamInt("status"))
	biz.ErrIsNil(a.silenceApp.ChangeStatus(rc.MetaCtx, id, status))
	rc.ReqParam = collx.Kvs("id", id, "status", status)
}

func (a *AlertSilence) Delete(rc *req.Ctx) {
	idsStr := rc.PathParam("id")
	rc.ReqParam = idsStr
	for _, id := range parseCommaIds(idsStr) {
		biz.ErrIsNil(a.silenceApp.DeleteSilence(rc.MetaCtx, id))
	}
}

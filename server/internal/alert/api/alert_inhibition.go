package api

import (
	"mayfly-go/internal/alert/application"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type AlertInhibition struct {
	inhibitionApp application.AlertInhibition `inject:"T"`
}

func (a *AlertInhibition) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("", a.List),
		req.NewGet(":id", a.GetById),
		req.NewPost("", a.Save).Log(req.NewLogSaveI(imsg.LogAlertInhibitionSave)),
		req.NewPut(":id", a.Update).Log(req.NewLogSaveI(imsg.LogAlertInhibitionSave)),
		req.NewPut(":id/:status", a.ChangeStatus).Log(req.NewLogSaveI(imsg.LogAlertInhibitionChangeStatus)),
		req.NewDelete(":id", a.Delete).Log(req.NewLogSaveI(imsg.LogAlertInhibitionDelete)),
	}
	return req.NewConfs("alert-inhibitions", reqs[:]...)
}

func (a *AlertInhibition) List(rc *req.Ctx) {
	condition := rc.BindQuery[entity.AlertInhibitionQuery]()
	res, err := a.inhibitionApp.GetAlertInhibitionList(condition)
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (a *AlertInhibition) GetById(rc *req.Ctx) {
	id := uint64(rc.PathParamInt("id"))
	inhibition, err := a.inhibitionApp.GetById(id)
	biz.ErrIsNil(err)
	rc.ResData = inhibition
}

func (a *AlertInhibition) Save(rc *req.Ctx) {
	inhibition := rc.BindJson[entity.AlertInhibition]()
	biz.ErrIsNil(a.inhibitionApp.SaveAlertInhibition(rc.MetaCtx, inhibition))
	rc.ResData = inhibition.Id
}

func (a *AlertInhibition) Update(rc *req.Ctx) {
	inhibition := rc.BindJson[entity.AlertInhibition]()
	inhibition.Id = uint64(rc.PathParamInt("id"))
	biz.ErrIsNil(a.inhibitionApp.SaveAlertInhibition(rc.MetaCtx, inhibition))
}

func (a *AlertInhibition) ChangeStatus(rc *req.Ctx) {
	id := uint64(rc.PathParamInt("id"))
	status := int8(rc.PathParamInt("status"))
	biz.ErrIsNil(a.inhibitionApp.ChangeStatus(rc.MetaCtx, id, status))
	rc.ReqParam = collx.Kvs("id", id, "status", status)
}

func (a *AlertInhibition) Delete(rc *req.Ctx) {
	idsStr := rc.PathParam("id")
	rc.ReqParam = idsStr
	for _, id := range parseCommaIds(idsStr) {
		biz.ErrIsNil(a.inhibitionApp.DeleteInhibition(rc.MetaCtx, id))
	}
}

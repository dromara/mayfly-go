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

type AlertNotifyPolicy struct {
	notifyPolicyApp application.AlertNotifyPolicy `inject:"T"`
}

func (a *AlertNotifyPolicy) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("", a.List),
		req.NewGet(":id", a.GetById),
		req.NewPost("", a.Save).Log(req.NewLogSaveI(imsg.LogAlertNotifyPolicySave)),
		req.NewPut(":id", a.Update).Log(req.NewLogSaveI(imsg.LogAlertNotifyPolicySave)),
		req.NewPut(":id/:status", a.ChangeStatus).Log(req.NewLogSaveI(imsg.LogAlertNotifyPolicyChangeStatus)),
		req.NewDelete(":id", a.Delete).Log(req.NewLogSaveI(imsg.LogAlertNotifyPolicyDelete)),
	}
	return req.NewConfs("alert-notify-policies", reqs[:]...)
}

func (a *AlertNotifyPolicy) List(rc *req.Ctx) {
	condition := rc.BindQuery[entity.AlertNotifyPolicyQuery]()
	res, err := a.notifyPolicyApp.GetAlertNotifyPolicyList(condition)
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (a *AlertNotifyPolicy) GetById(rc *req.Ctx) {
	id := uint64(rc.PathParamInt("id"))
	policy, err := a.notifyPolicyApp.GetById(id)
	biz.ErrIsNil(err)
	rc.ResData = policy
}

func (a *AlertNotifyPolicy) Save(rc *req.Ctx) {
	policy := rc.BindJson[entity.AlertNotifyPolicy]()
	biz.ErrIsNil(a.notifyPolicyApp.SaveAlertNotifyPolicy(rc.MetaCtx, policy))
	rc.ResData = policy.Id
}

func (a *AlertNotifyPolicy) Update(rc *req.Ctx) {
	policy := rc.BindJson[entity.AlertNotifyPolicy]()
	policy.Id = uint64(rc.PathParamInt("id"))
	biz.ErrIsNil(a.notifyPolicyApp.SaveAlertNotifyPolicy(rc.MetaCtx, policy))
}

func (a *AlertNotifyPolicy) ChangeStatus(rc *req.Ctx) {
	id := uint64(rc.PathParamInt("id"))
	status := int8(rc.PathParamInt("status"))
	biz.IsTrueBy(status == entity.AlertNotifyPolicyStatusEnable || status == entity.AlertNotifyPolicyStatusDisable,
		errorx.NewBizI(rc.MetaCtx, imsg.ErrNotifyPolicyStatusInvalid))
	rc.ReqParam = collx.Kvs("id", id, "status", status)
	biz.ErrIsNil(a.notifyPolicyApp.ChangeStatus(rc.MetaCtx, id, status))
}

func (a *AlertNotifyPolicy) Delete(rc *req.Ctx) {
	idsStr := rc.PathParam("id")
	rc.ReqParam = idsStr
	for _, id := range parseCommaIds(idsStr) {
		biz.ErrIsNil(a.notifyPolicyApp.DeleteNotifyPolicy(rc.MetaCtx, id))
	}
}

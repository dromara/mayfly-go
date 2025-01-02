package api

import (
	"mayfly-go/internal/tag/application"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

type ResourceOpLog struct {
	resourceOpLogApp application.ResourceOpLog `inject:"T"`
}

func (r *ResourceOpLog) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("/account", r.PageAccountOpLog),
	}

	return req.NewConfs("/resource-op-logs", reqs[:]...)
}

func (r *ResourceOpLog) PageAccountOpLog(rc *req.Ctx) {
	cond := new(entity.ResourceOpLog)
	cond.ResourceCode = rc.Query("resourceCode")
	cond.ResourceType = int8(rc.QueryInt("resourceType"))
	cond.CreatorId = rc.GetLoginAccount().Id

	rols, err := r.resourceOpLogApp.PageByCond(cond, rc.GetPageParam())
	biz.ErrIsNil(err)
	rc.ResData = rols
}

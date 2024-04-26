package api

import (
	"mayfly-go/internal/tag/application"
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
)

type ResourceOpLog struct {
	ResourceOpLogApp application.ResourceOpLog `inject:""`
}

func (r *ResourceOpLog) PageAccountOpLog(rc *req.Ctx) {
	cond := new(entity.ResourceOpLog)
	cond.ResourceCode = rc.Query("resourceCode")
	cond.ResourceType = int8(rc.QueryInt("resourceType"))
	cond.CreatorId = rc.GetLoginAccount().Id

	var rols []*entity.ResourceOpLog
	res, err := r.ResourceOpLogApp.PageQuery(cond, rc.GetPageParam(), &rols)
	biz.ErrIsNil(err)
	rc.ResData = res
}

package api

import (
	"encoding/json"
	"mayfly-go/internal/sys/api/form"
	"mayfly-go/internal/sys/api/vo"
	"mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type Resource struct {
	ResourceApp application.Resource `inject:""`
}

func (r *Resource) GetAllResourceTree(rc *req.Ctx) {
	var resources vo.ResourceManageVOList
	r.ResourceApp.ListByCondOrder(new(entity.Resource), &resources, "weight asc")
	rc.ResData = resources.ToTrees(0)
}

func (r *Resource) GetById(rc *req.Ctx) {
	res, err := r.ResourceApp.GetById(new(entity.Resource), uint64(rc.F.PathParamInt("id")))
	biz.ErrIsNil(err, "该资源不存在")
	rc.ResData = res
}

func (r *Resource) SaveResource(rc *req.Ctx) {
	form := new(form.ResourceForm)
	entity := req.BindJsonAndCopyTo(rc, form, new(entity.Resource))

	rc.ReqParam = form

	// 将meta转为json字符串存储
	bytes, _ := json.Marshal(form.Meta)
	entity.Meta = string(bytes)

	biz.ErrIsNil(r.ResourceApp.Save(rc.MetaCtx, entity))
}

func (r *Resource) DelResource(rc *req.Ctx) {
	biz.ErrIsNil(r.ResourceApp.Delete(rc.MetaCtx, uint64(rc.F.PathParamInt("id"))))
}

func (r *Resource) ChangeStatus(rc *req.Ctx) {
	rid := uint64(rc.F.PathParamInt("id"))
	status := int8(rc.F.PathParamInt("status"))
	rc.ReqParam = collx.Kvs("id", rid, "status", status)
	biz.ErrIsNil(r.ResourceApp.ChangeStatus(rc.MetaCtx, rid, status))
}

func (r *Resource) Sort(rc *req.Ctx) {
	var rs []form.ResourceForm
	rc.F.BindJSON(&rs)
	rc.ReqParam = rs

	for _, v := range rs {
		sortE := &entity.Resource{Pid: v.Pid, Weight: v.Weight}
		sortE.Id = uint64(v.Id)
		r.ResourceApp.Sort(rc.MetaCtx, sortE)
	}
}

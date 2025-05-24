package api

import (
	"encoding/json"
	"mayfly-go/internal/sys/api/form"
	"mayfly-go/internal/sys/api/vo"
	"mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type Resource struct {
	resourceApp application.Resource `inject:"T"`
}

func (r *Resource) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("", r.GetAllResourceTree),

		req.NewGet(":id", r.GetById),

		req.NewGet(":id/roles", r.GetResourceRoles),

		req.NewPost("", r.SaveResource).Log(req.NewLogSaveI(imsg.LogResourceSave)).RequiredPermissionCode("resource:add"),

		req.NewPut(":id/:status", r.ChangeStatus).Log(req.NewLogSaveI(imsg.LogChangeResourceStatus)).RequiredPermissionCode("resource:changeStatus"),

		req.NewPost("sort", r.Sort).Log(req.NewLogSaveI(imsg.LogSortResource)),

		req.NewDelete(":id", r.DelResource).Log(req.NewLogSaveI(imsg.LogResourceDelete)).RequiredPermissionCode("resource:delete"),
	}

	return req.NewConfs("sys/resources", reqs[:]...)
}

func (r *Resource) GetAllResourceTree(rc *req.Ctx) {
	var resources vo.ResourceManageVOList
	r.resourceApp.ListByCondToAny(model.NewCond().OrderByAsc("weight"), &resources)
	rc.ResData = resources.ToTrees(0)
}

func (r *Resource) GetById(rc *req.Ctx) {
	res, err := r.resourceApp.GetById(uint64(rc.PathParamInt("id")))
	biz.ErrIsNil(err, "The resource does not exist")
	rc.ResData = res
}

func (r *Resource) SaveResource(rc *req.Ctx) {
	form, entity := req.BindJsonAndCopyTo[*form.ResourceForm, *entity.Resource](rc)

	rc.ReqParam = form

	// 将meta转为json字符串存储
	bytes, _ := json.Marshal(form.Meta)
	entity.Meta = string(bytes)

	biz.ErrIsNil(r.resourceApp.Save(rc.MetaCtx, entity))
}

func (r *Resource) DelResource(rc *req.Ctx) {
	biz.ErrIsNil(r.resourceApp.Delete(rc.MetaCtx, uint64(rc.PathParamInt("id"))))
}

func (r *Resource) ChangeStatus(rc *req.Ctx) {
	rid := uint64(rc.PathParamInt("id"))
	status := int8(rc.PathParamInt("status"))
	rc.ReqParam = collx.Kvs("id", rid, "status", status)
	biz.ErrIsNil(r.resourceApp.ChangeStatus(rc.MetaCtx, rid, status))
}

func (r *Resource) Sort(rc *req.Ctx) {
	var rs []form.ResourceForm
	rc.BindJSON(&rs)
	rc.ReqParam = rs

	for _, v := range rs {
		sortE := &entity.Resource{Pid: v.Pid, Weight: v.Weight}
		sortE.Id = uint64(v.Id)
		r.resourceApp.Sort(rc.MetaCtx, sortE)
	}
}

// GetResourceRoles
func (r *Resource) GetResourceRoles(rc *req.Ctx) {
	rrs, err := r.resourceApp.GetResourceRoles(uint64(rc.PathParamInt("id")))
	biz.ErrIsNil(err)
	rc.ResData = rrs
}

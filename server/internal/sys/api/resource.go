package api

import (
	"encoding/json"
	"fmt"
	"mayfly-go/internal/sys/api/form"
	"mayfly-go/internal/sys/api/vo"
	"mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/ginx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils"
)

type Resource struct {
	ResourceApp application.Resource
}

func (r *Resource) GetAllResourceTree(rc *req.Ctx) {
	var resources vo.ResourceManageVOList
	r.ResourceApp.GetResourceList(new(entity.Resource), &resources, "weight asc")
	rc.ResData = resources.ToTrees(0)
}

func (r *Resource) GetById(rc *req.Ctx) {
	rc.ResData = r.ResourceApp.GetById(uint64(ginx.PathParamInt(rc.GinCtx, "id")))
}

func (r *Resource) SaveResource(rc *req.Ctx) {
	g := rc.GinCtx
	form := new(form.ResourceForm)
	ginx.BindJsonAndValid(g, form)
	rc.ReqParam = form

	entity := new(entity.Resource)
	utils.Copy(entity, form)
	// 将meta转为json字符串存储
	bytes, _ := json.Marshal(form.Meta)
	entity.Meta = string(bytes)

	entity.SetBaseInfo(rc.LoginAccount)
	r.ResourceApp.Save(entity)
}

func (r *Resource) DelResource(rc *req.Ctx) {
	r.ResourceApp.Delete(uint64(ginx.PathParamInt(rc.GinCtx, "id")))
}

func (r *Resource) ChangeStatus(rc *req.Ctx) {
	rid := uint64(ginx.PathParamInt(rc.GinCtx, "id"))
	status := int8(ginx.PathParamInt(rc.GinCtx, "status"))
	rc.ReqParam = fmt.Sprintf("id = %d, status = %d", rid, status)
	r.ResourceApp.ChangeStatus(rid, status)
}

func (r *Resource) Sort(rc *req.Ctx) {
	var rs []form.ResourceForm
	rc.GinCtx.ShouldBindJSON(&rs)
	rc.ReqParam = rs

	for _, v := range rs {
		sortE := &entity.Resource{Pid: v.Pid, Weight: v.Weight}
		sortE.Id = uint64(v.Id)
		r.ResourceApp.Sort(sortE)
	}
}

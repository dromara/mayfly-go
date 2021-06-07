package apis

import (
	"encoding/json"
	"mayfly-go/base/ctx"
	"mayfly-go/base/ginx"
	"mayfly-go/base/utils"
	"mayfly-go/server/sys/apis/form"
	"mayfly-go/server/sys/apis/vo"
	"mayfly-go/server/sys/application"
	"mayfly-go/server/sys/domain/entity"
)

type Resource struct {
	ResourceApp application.IResource
}

func (r *Resource) GetAllResourceTree(rc *ctx.ReqCtx) {
	var resources vo.ResourceManageVOList
	r.ResourceApp.GetResourceList(new(entity.Resource), &resources, "weight asc")
	rc.ResData = resources.ToTrees(0)
}

func (r *Resource) GetById(rc *ctx.ReqCtx) {
	rc.ResData = r.ResourceApp.GetById(uint64(ginx.PathParamInt(rc.GinCtx, "id")))
}

func (r *Resource) SaveResource(rc *ctx.ReqCtx) {
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

func (r *Resource) DelResource(rc *ctx.ReqCtx) {
	r.ResourceApp.Delete(uint64(ginx.PathParamInt(rc.GinCtx, "id")))
}

func (r *Resource) ChangeStatus(rc *ctx.ReqCtx) {
	re := &entity.Resource{}
	re.Id = uint64(ginx.PathParamInt(rc.GinCtx, "id"))
	re.Status = int8(ginx.PathParamInt(rc.GinCtx, "status"))
	r.ResourceApp.Save(re)
}

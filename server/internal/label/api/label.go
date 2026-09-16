package api

import (
	"mayfly-go/internal/label/application"
	"mayfly-go/internal/label/domain/entity"
	"mayfly-go/internal/label/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type Label struct {
	labelApp        application.Label        `inject:"T"`
	labelBindingApp application.LabelBinding `inject:"T"`
}

func (l *Label) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("", l.List),
		req.NewGet("keys", l.ListKeys),
		req.NewGet("values", l.ListValues),
		req.NewGet("autocomplete", l.Autocomplete),
		req.NewPost("", l.Save).Log(req.NewLogSaveI(imsg.LogLabelSave)),
		req.NewPut(":id", l.Update).Log(req.NewLogSaveI(imsg.LogLabelSave)),
		req.NewDelete(":id", l.Delete).Log(req.NewLogSaveI(imsg.LogLabelDelete)),

		// 绑定相关
		req.NewGet("bindings/:targetType/:targetId", l.ListBindings),
		req.NewPost("bindings", l.SaveBindings),
		req.NewDelete("bindings/:labelId/:targetType/:targetId", l.DeleteBinding),
		// 批量填充标签（类似 TagTree.FillTagInfo）
		req.NewPost("bindings/fill", l.FillBindings),
	}
	return req.NewConfs("labels", reqs[:]...)
}

func (l *Label) List(rc *req.Ctx) {
	condition := rc.BindQuery[entity.LabelQuery]()
	res, err := l.labelApp.GetLabelList(condition)
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (l *Label) ListKeys(rc *req.Ctx) {
	keys, err := l.labelApp.ListKeys()
	biz.ErrIsNil(err)
	rc.ResData = keys
}

func (l *Label) ListValues(rc *req.Ctx) {
	labelKey := rc.Query("labelKey")
	values, err := l.labelApp.ListValuesByKey(labelKey)
	biz.ErrIsNil(err)
	rc.ResData = values
}

func (l *Label) Autocomplete(rc *req.Ctx) {
	labelKey := rc.Query("key")
	items, err := l.labelApp.GetAutocomplete(labelKey)
	biz.ErrIsNil(err)
	rc.ResData = items
}

func (l *Label) Save(rc *req.Ctx) {
	label := rc.BindJson[entity.Label]()
	biz.ErrIsNil(l.labelApp.SaveLabel(rc.MetaCtx, label))
	rc.ResData = label.Id
}

func (l *Label) Update(rc *req.Ctx) {
	label := rc.BindJson[entity.Label]()
	label.Id = uint64(rc.PathParamInt("id"))
	biz.ErrIsNil(l.labelApp.SaveLabel(rc.MetaCtx, label))
}

func (l *Label) Delete(rc *req.Ctx) {
	id := uint64(rc.PathParamInt("id"))
	rc.ReqParam = id
	biz.ErrIsNil(l.labelApp.DeleteLabel(rc.MetaCtx, id))
}

// 绑定相关

func (l *Label) ListBindings(rc *req.Ctx) {
	targetType := rc.PathParam("targetType")
	targetId := uint64(rc.PathParamInt("targetId"))
	bindings, err := l.labelBindingApp.ListByTarget(rc.MetaCtx, targetType, targetId)
	biz.ErrIsNil(err)
	rc.ResData = bindings
}

func (l *Label) SaveBindings(rc *req.Ctx) {
	bindings := rc.BindJson[[]entity.LabelBinding]()
	// 转换为指针切片
	ptrBindings := make([]*entity.LabelBinding, len(*bindings))
	for i := range *bindings {
		ptrBindings[i] = &(*bindings)[i]
	}
	if len(ptrBindings) > 0 {
		rc.ReqParam = collx.Kvs("targetType", ptrBindings[0].TargetType, "targetId", ptrBindings[0].TargetId)
	}
	biz.ErrIsNil(l.labelBindingApp.SaveBindings(rc.MetaCtx, ptrBindings))
}

func (l *Label) DeleteBinding(rc *req.Ctx) {
	labelId := uint64(rc.PathParamInt("labelId"))
	targetType := rc.PathParam("targetType")
	targetId := uint64(rc.PathParamInt("targetId"))
	rc.ReqParam = collx.Kvs("labelId", labelId, "targetType", targetType, "targetId", targetId)
	biz.ErrIsNil(l.labelBindingApp.DeleteBinding(rc.MetaCtx, labelId, targetType, targetId))
}

// FillBindings 批量填充标签绑定信息（类似 TagTree.FillTagInfo）
// 请求体：{"targetType": "alert_rule", "targetIds": [1, 2, 3]}
// 返回：map[targetId]LabelBindingVO[]
func (l *Label) FillBindings(rc *req.Ctx) {
	type fillReq struct {
		TargetType string   `json:"targetType"`
		TargetIds  []uint64 `json:"targetIds"`
	}
	reqBody := rc.BindJson[fillReq]()

	result, err := l.labelBindingApp.ListByTargets(rc.MetaCtx, reqBody.TargetType, reqBody.TargetIds)
	biz.ErrIsNil(err)
	rc.ResData = result
}

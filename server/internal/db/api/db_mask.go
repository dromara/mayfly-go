package api

import (
	"mayfly-go/internal/db/application/mask"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"

	"github.com/spf13/cast"
)

type DbMask struct {
	maskApp mask.MaskApp `inject:"T"`
}

func (d *DbMask) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 分页获取脱敏规则
		req.NewGet("/mask-rules", d.Rules),
		// 保存脱敏规则
		req.NewPost("/mask-rules", d.SaveRule).Log(req.NewLogSaveI(imsg.LogDbMaskRuleSave)),
		req.NewPut("/mask-rules", d.SaveRule).Log(req.NewLogSaveI(imsg.LogDbMaskRuleSave)),
		// 删除脱敏规则
		req.NewDelete("/mask-rules/:id", d.DeleteRule).Log(req.NewLogSaveI(imsg.LogDbMaskRuleDelete)),

		// 分页获取脱敏列标签
		req.NewGet("/mask-columns", d.Columns),
		// 保存脱敏列标签
		req.NewPost("/mask-columns", d.SaveColumn).Log(req.NewLogSaveI(imsg.LogDbMaskTagSave)),
		req.NewPut("/mask-columns", d.SaveColumn).Log(req.NewLogSaveI(imsg.LogDbMaskTagSave)),
		// 删除脱敏列标签
		req.NewDelete("/mask-columns/:id", d.DeleteColumn).Log(req.NewLogSaveI(imsg.LogDbMaskTagDelete)),
	}

	return req.NewConfs("/dbs", reqs[:]...)
}

// Rules 分页获取脱敏规则
// @router /api/dbs/mask-rules [get]
func (d *DbMask) Rules(rc *req.Ctx) {
	queryCond := rc.BindQuery[entity.MaskRuleQuery]()
	res, err := d.maskApp.GetRulePageList(queryCond)
	biz.ErrIsNil(err)
	rc.ResData = res
}

// SaveRule 保存脱敏规则
// @router /api/dbs/mask-rules [post]
func (d *DbMask) SaveRule(rc *req.Ctx) {
	rule := rc.BindJson[entity.DbMaskRule]()
	rc.ReqParam = rule
	biz.ErrIsNil(d.maskApp.SaveRule(rc.MetaCtx, rule))
}

// DeleteRule 删除脱敏规则
// @router /api/dbs/mask-rules/:id [delete]
func (d *DbMask) DeleteRule(rc *req.Ctx) {
	id := uint64(rc.PathParamInt("id"))
	biz.IsTrue(id > 0, "id error")
	rc.ReqParam = id
	biz.ErrIsNil(d.maskApp.DeleteRule(rc.MetaCtx, id))
}

// Columns 分页获取脱敏列标签
// @router /api/dbs/mask-columns [get]
func (d *DbMask) Columns(rc *req.Ctx) {
	queryCond := rc.BindQuery[entity.MaskColumnQuery]()
	res, err := d.maskApp.GetTagPageList(queryCond)
	biz.ErrIsNil(err)
	rc.ResData = res
}

// SaveColumn 保存脱敏列标签
// @router /api/dbs/mask-columns [post]
func (d *DbMask) SaveColumn(rc *req.Ctx) {
	tag := rc.BindJson[entity.DbMaskColumn]()
	rc.ReqParam = tag
	biz.ErrIsNil(d.maskApp.SaveTag(rc.MetaCtx, tag))
}

// DeleteColumn 删除脱敏列标签
// @router /api/dbs/mask-columns/:id [delete]
func (d *DbMask) DeleteColumn(rc *req.Ctx) {
	id := cast.ToUint64(rc.PathParamInt("id"))
	biz.IsTrue(id > 0, "id error")
	rc.ReqParam = id
	biz.ErrIsNil(d.maskApp.DeleteTag(rc.MetaCtx, id))
}

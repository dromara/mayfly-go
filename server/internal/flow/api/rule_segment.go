package api

import (
	"mayfly-go/internal/flow/api/form"
	"mayfly-go/internal/flow/application"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/imsg"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"strings"

	"github.com/spf13/cast"
)

// RuleSegment 可复用条件组：把一段判断（如「DBA 成员」「工作时段之外」）单独维护，
// 供触发策略与流程图里的条件按标识引用
type RuleSegment struct {
	ruleSegmentApp application.RuleSegment `inject:"T"`
}

func (r *RuleSegment) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("", r.GetPageList),

		req.NewPost("", r.Save).Log(req.NewLogSaveI(imsg.LogRuleSegmentSave)).RequiredPermissionCode("flow:ruleSegment:save"),

		req.NewDelete(":id", r.Delete).Log(req.NewLogSaveI(imsg.LogRuleSegmentDelete)).RequiredPermissionCode("flow:ruleSegment:del"),
	}

	return req.NewConfs("/flow/rule-segments", reqs[:]...)
}

func (r *RuleSegment) GetPageList(rc *req.Ctx) {
	queryCond := rc.BindQuery[entity.RuleSegmentQuery]()
	res, err := r.ruleSegmentApp.GetPageList(queryCond)
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (r *RuleSegment) Save(rc *req.Ctx) {
	formData := rc.BindJson[form.RuleSegment]()
	rc.ReqParam = formData

	segment := &entity.RuleSegment{Ref: formData.Ref, Name: formData.Name, BizType: formData.BizType, Remark: formData.Remark, RuleNode: formData.RuleNode}
	segment.Id = formData.Id
	biz.ErrIsNil(r.ruleSegmentApp.SaveRuleSegment(rc.MetaCtx, segment))
	rc.ResData = segment.Id
}

func (r *RuleSegment) Delete(rc *req.Ctx) {
	idsStr := rc.PathParam("id")
	rc.ReqParam = idsStr
	for _, id := range strings.Split(idsStr, ",") {
		// 跳过空段：批量删除按逗号拆分，末尾空串会变成 id=0 的查询并抛出无意义的 record not found
		if id == "" {
			continue
		}
		// 直接透出业务错误：被引用时的提示本身已是本地化文案，
		// 再套一层英文前缀会让用户看到半中半英的句子
		biz.ErrIsNil(r.ruleSegmentApp.DeleteRuleSegment(rc.MetaCtx, cast.ToUint64(id)))
	}
}

package api

import (
	"mayfly-go/internal/alert/application"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/service"
	"mayfly-go/internal/alert/imsg"
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"strconv"

	"github.com/spf13/cast"
)

type AlertRule struct {
	ruleApp application.AlertRule `inject:"T"`
}

func (a *AlertRule) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("", a.List),
		// Metrics 为静态路由，必须注册在 ":id" 之前，避免被通配参数吞掉
		req.NewGet("metrics", a.Metrics),
		req.NewGet(":id", a.GetById),
		req.NewPost("", a.Save).Log(req.NewLogSaveI(imsg.LogAlertRuleSave)),
		req.NewPut(":id", a.Update).Log(req.NewLogSaveI(imsg.LogAlertRuleSave)),
		req.NewPut(":id/:status", a.ChangeStatus).Log(req.NewLogSaveI(imsg.LogAlertRuleChangeStatus)),
		req.NewDelete(":id", a.Delete).Log(req.NewLogSaveI(imsg.LogAlertRuleDelete)),
	}
	return req.NewConfs("alert-rules", reqs[:]...)
}

func (a *AlertRule) List(rc *req.Ctx) {
	condition := rc.BindQuery[entity.AlertRuleQuery]()
	res, err := a.ruleApp.GetAlertRuleList(condition)
	biz.ErrIsNil(err)
	rc.ResData = res
}

// ResourceMetrics 某资源类型支持的指标集合
type ResourceMetrics struct {
	ResourceType int8 `json:"resourceType"`
	// ResourceKey 资源类型的稳定字符串键（machine/db/...，真源见 consts.ResourceTypeKey）。
	// 数值型供告警规则自身联动；键字符串供跨模块消费（如 agentops 触发器匹配条件存的就是
	// 事件归一化后的键串）。同一接口一次下发两种形态，前端不必自抄一份数值↔键映射表
	ResourceKey string                     `json:"resourceKey"`
	Metrics     []service.MetricDefinition `json:"metrics"`
}

// Metrics 返回已注册评估器的资源类型及其支持的指标集合。
// 前端指标下拉与资源类型联动必须以此为唯一数据源，否则会出现"选了数据库类型却展示机器指标"
func (a *AlertRule) Metrics(rc *req.Ctx) {
	resourceTypes := rc.Query("resourceType")
	var types []int8
	if resourceTypes != "" {
		value, err := strconv.ParseInt(resourceTypes, 10, 8)
		biz.ErrIsNilAppendErr(err, "invalid resourceType: %s")
		types = []int8{int8(value)}
	} else {
		types = service.SupportedResourceTypes()
	}

	res := make([]*ResourceMetrics, 0, len(types))
	for _, rt := range types {
		key, _ := consts.ResourceTypeKey(rt)
		res = append(res, &ResourceMetrics{ResourceType: rt, ResourceKey: key, Metrics: service.MetricsOf(rt)})
	}
	rc.ResData = res
}

func (a *AlertRule) GetById(rc *req.Ctx) {
	id := cast.ToUint64(rc.PathParam("id"))
	rule, err := a.ruleApp.GetById(id)
	biz.ErrIsNil(err)
	rc.ResData = rule
}

// alertRuleForm 告警规则保存请求体。
//
// priority 必须用指针承接：P0 的枚举值就是 0，与 Go 零值同形，直接绑实体无法区分
// 「用户选了 P0」和「用户没选」。字段名与嵌入的 entity.AlertRule.Priority 同名，按 encoding/json
// 的深度优先规则由本层字段遮蔽，因此能拿到「是否出现于请求体」这个信息
type alertRuleForm struct {
	entity.AlertRule
	Priority *entity.AlertPriority `json:"priority"`
}

// toEntity 归一为领域实体：未提交 priority 时回退为 P2（与前端表单默认值、历史列默认值一致），
// 显式提交 0 则按 P0 处理
func (f *alertRuleForm) toEntity() *entity.AlertRule {
	rule := f.AlertRule
	if f.Priority != nil {
		rule.Priority = *f.Priority
	} else {
		rule.Priority = entity.AlertPriorityMedium
	}
	// Labels 为 JSON 列，空值需归一为合法 JSON
	if rule.Labels == "" {
		rule.Labels = "{}"
	}
	return &rule
}

func (a *AlertRule) Save(rc *req.Ctx) {
	rule := rc.BindJson[alertRuleForm]().toEntity()
	biz.ErrIsNil(a.ruleApp.SaveAlertRule(rc.MetaCtx, rule))
	rc.ResData = rule.Id
}

func (a *AlertRule) Update(rc *req.Ctx) {
	rule := rc.BindJson[alertRuleForm]().toEntity()
	rule.Id = cast.ToUint64(rc.PathParam("id"))
	biz.ErrIsNil(a.ruleApp.SaveAlertRule(rc.MetaCtx, rule))
}

func (a *AlertRule) ChangeStatus(rc *req.Ctx) {
	id := cast.ToUint64(rc.PathParam("id"))
	status := int8(rc.PathParamInt("status"))
	biz.IsTrueBy(status == entity.AlertRuleStatusEnable || status == entity.AlertRuleStatusDisable,
		errorx.NewBizI(rc.MetaCtx, imsg.ErrRuleStatusInvalid))
	rc.ReqParam = collx.Kvs("id", id, "status", status)
	// 必须调用应用层 ChangeStatus（内含禁用时级联关闭活跃事件逻辑），
	// 直接 UpdateById 会绕过级联，导致活跃事件变成孤儿、概览统计虚高
	biz.ErrIsNil(a.ruleApp.ChangeStatus(rc.MetaCtx, id, status))
}

func (a *AlertRule) Delete(rc *req.Ctx) {
	idsStr := rc.PathParam("id")
	rc.ReqParam = idsStr
	for _, id := range parseCommaIds(idsStr) {
		biz.ErrIsNil(a.ruleApp.DeleteAlertRule(rc.MetaCtx, id))
	}
}

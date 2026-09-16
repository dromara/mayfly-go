package application

import (
	"context"
	"encoding/json"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/repository"
	"mayfly-go/internal/alert/domain/service"
	"mayfly-go/internal/alert/imsg"
	labelapp "mayfly-go/internal/label/application"
	labelentity "mayfly-go/internal/label/domain/entity"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"strconv"
	"strings"
)

// 规则数值字段的合法区间
const (
	minEvalInterval  = 10    // 评估间隔下限(秒)
	maxEvalInterval  = 86400 // 评估间隔上限(秒)
	minLimitCount    = 1     // 触发/恢复次数下限
	maxLimitCount    = 1000  // 触发/恢复次数上限
	defaultEvalIntvl = 60    // 默认评估间隔(秒)
)

// AlertRule 告警规则应用层接口
type AlertRule interface {
	base.App[*entity.AlertRule]

	// SaveAlertRule 保存告警规则（保存前做全量业务校验并归一化默认值）
	SaveAlertRule(ctx context.Context, rule *entity.AlertRule) error

	// ChangeStatus 修改规则状态
	ChangeStatus(ctx context.Context, id uint64, status int8) error

	// DeleteAlertRule 删除告警规则（规则不存在时报错）
	DeleteAlertRule(ctx context.Context, id uint64) error

	// GetAlertRuleList 分页获取告警规则列表
	GetAlertRuleList(condition *entity.AlertRuleQuery, orderBy ...string) (*model.PageResult[*entity.AlertRule], error)

	// ListEnabled 获取所有启用的告警规则
	ListEnabled() ([]*entity.AlertRule, error)

	// ---- 概览统计方法 ----

	// GetRuleStats 返回规则总数、启用数、禁用数
	GetRuleStats() (total, enabled, disabled int64, err error)
	// CountGroupByPriority 按优先级分组统计
	CountGroupByPriority() (map[string]int64, error)
	// CountGroupByResourceType 按资源类型分组统计
	CountGroupByResourceType() (map[int8]int64, error)
}

type alertRuleAppImpl struct {
	base.AppImpl[*entity.AlertRule, repository.AlertRule]
	eventApp         AlertEvent            `inject:"T"`
	tagTreeRelateApp tagapp.TagTreeRelate  `inject:"T"`
	labelBindingApp  labelapp.LabelBinding `inject:"T"`
}

var _ AlertRule = (*alertRuleAppImpl)(nil)

func (a *alertRuleAppImpl) SaveAlertRule(ctx context.Context, rule *entity.AlertRule) error {
	if err := validateAlertRule(ctx, rule); err != nil {
		return err
	}

	if rule.Id == 0 {
		// 新增：先插入获取 ID，再建立标签关联
		if err := a.Insert(ctx, rule); err != nil {
			return err
		}
	} else {
		// 更新：先更新实体，成功后再清理旧关联并重建（避免更新失败时关联已丢失）
		if err := a.UpdateById(ctx, rule); err != nil {
			return err
		}
		// 更新成功后清理旧标签路径关联
		if err := a.tagTreeRelateApp.DeleteByCond(ctx, model.NewCond().Eq("relate_type", tagentity.TagRelateTypeAlertRule).Eq("relate_id", rule.Id)); err != nil {
			return err
		}
	}

	// 标签路径模式：建立标签关联记录（与计划任务保持一致）
	if rule.ScopeType == entity.AlertScopeTagPath {
		var paths []string
		if err := json.Unmarshal([]byte(rule.ScopeValue), &paths); err == nil && len(paths) > 0 {
			logx.Debugf("[alert] SaveAlertRule: rule[%d] calling RelateTag with %d paths: %v", rule.Id, len(paths), paths)
			if err := a.tagTreeRelateApp.RelateTag(ctx, tagentity.TagRelateTypeAlertRule, rule.Id, paths...); err != nil {
				logx.Errorf("[alert] SaveAlertRule: rule[%d] RelateTag error: %s", rule.Id, err.Error())
				return err
			}
			logx.Debugf("[alert] SaveAlertRule: rule[%d] RelateTag success", rule.Id)
		} else {
			logx.Warnf("[alert] SaveAlertRule: rule[%d] no valid paths in scopeValue=%s, err=%v", rule.Id, rule.ScopeValue, err)
		}
	}

	// 保存标签绑定到 t_label_binding 表
	if rule.Labels != "" && rule.Labels != "{}" {
		var labels map[string]string
		if err := json.Unmarshal([]byte(rule.Labels), &labels); err == nil && len(labels) > 0 {
			logx.Debugf("[alert] SaveAlertRule: rule[%d] saving %d label bindings", rule.Id, len(labels))
			if err := a.labelBindingApp.SaveBindingsByLabels(ctx, labelentity.LabelTargetAlertRule, rule.Id, labels); err != nil {
				logx.Errorf("[alert] SaveAlertRule: rule[%d] SaveBindingsByLabels error: %s", rule.Id, err.Error())
				return err
			}
			logx.Debugf("[alert] SaveAlertRule: rule[%d] label bindings saved", rule.Id)
		}
	} else {
		// 清空标签绑定
		if err := a.labelBindingApp.DeleteByTarget(ctx, labelentity.LabelTargetAlertRule, rule.Id); err != nil {
			logx.Errorf("[alert] SaveAlertRule: rule[%d] DeleteByTarget error: %s", rule.Id, err.Error())
			return err
		}
	}

	return nil
}

func (a *alertRuleAppImpl) DeleteAlertRule(ctx context.Context, id uint64) error {
	if _, err := a.GetById(id); err != nil {
		return errorx.NewBizI(ctx, imsg.ErrRuleNotFound, "id", id)
	}
	// 级联关闭活跃事件，防止删除规则后事件变成孤儿
	if err := a.eventApp.CloseByRuleId(ctx, id); err != nil {
		return err
	}
	// 清理标签关联记录
	if err := a.tagTreeRelateApp.DeleteByCond(ctx, model.NewCond().Eq("relate_type", tagentity.TagRelateTypeAlertRule).Eq("relate_id", id)); err != nil {
		return err
	}
	// 清理标签绑定记录
	if err := a.labelBindingApp.DeleteByTarget(ctx, labelentity.LabelTargetAlertRule, id); err != nil {
		return err
	}
	return a.DeleteById(ctx, id)
}

func (a *alertRuleAppImpl) ChangeStatus(ctx context.Context, id uint64, status int8) error {
	rule := new(entity.AlertRule)
	rule.Id = id
	rule.Status = status
	if err := a.UpdateById(ctx, rule); err != nil {
		return err
	}
	// 禁用规则时级联关闭活跃事件，防止概览统计虚高
	if status == entity.AlertRuleStatusDisable {
		return a.eventApp.CloseByRuleId(ctx, id)
	}
	return nil
}

func (a *alertRuleAppImpl) GetAlertRuleList(condition *entity.AlertRuleQuery, orderBy ...string) (*model.PageResult[*entity.AlertRule], error) {
	return a.GetRepo().GetAlertRuleList(condition, orderBy...)
}

func (a *alertRuleAppImpl) ListEnabled() ([]*entity.AlertRule, error) {
	return a.GetRepo().ListEnabled()
}

func (a *alertRuleAppImpl) GetRuleStats() (total, enabled, disabled int64, err error) {
	return a.GetRepo().GetRuleStats()
}

func (a *alertRuleAppImpl) CountGroupByPriority() (map[string]int64, error) {
	return a.GetRepo().CountGroupByPriority()
}

func (a *alertRuleAppImpl) CountGroupByResourceType() (map[int8]int64, error) {
	return a.GetRepo().CountGroupByResourceType()
}

// validateAlertRule 全量校验告警规则并归一化默认值。
//
// 校验必须覆盖所有"会让规则静默失效"的配置组合：非法比较方式在求值时恒为 false、
// 未知指标无法取值、指向不存在资源的范围永远展开为空、无评估器的资源类型不会被任何
// 评估周期处理。这些若不在保存时拒绝，规则会一直存在却永不告警，用户完全无感知。
func validateAlertRule(ctx context.Context, rule *entity.AlertRule) error {
	if strings.TrimSpace(rule.Name) == "" {
		return errorx.NewBizI(ctx, imsg.ErrRuleNameRequired)
	}
	if rule.Status != entity.AlertRuleStatusEnable && rule.Status != entity.AlertRuleStatusDisable {
		return errorx.NewBizI(ctx, imsg.ErrRuleStatusInvalid)
	}
	if !rule.Priority.IsValid() {
		return errorx.NewBizI(ctx, imsg.ErrRulePriorityInvalid)
	}
	if rule.ResourceType == 0 {
		return errorx.NewBizI(ctx, imsg.ErrRuleResourceTypeRequired)
	}
	if !service.SupportsResourceType(rule.ResourceType) {
		return errorx.NewBizI(ctx, imsg.ErrRuleResourceTypeUnsupported, "type", rule.ResourceType)
	}

	if err := validateRuleScope(ctx, rule); err != nil {
		return err
	}
	if err := validateRuleCondition(ctx, rule); err != nil {
		return err
	}
	return validateRuleAdvanced(ctx, rule)
}

// validateRuleScope 校验告警规则的关联范围：范围展开为空等价于规则永不生效，必须拒绝
func validateRuleScope(ctx context.Context, rule *entity.AlertRule) error {
	if !entity.IsValidRuleScopeType(rule.ScopeType) {
		return errorx.NewBizI(ctx, imsg.ErrRuleScopeTypeInvalid)
	}
	rule.ScopeValue = strings.TrimSpace(rule.ScopeValue)
	return validateScopeValue(ctx, rule.ResourceType, rule.ScopeType, rule.ScopeValue)
}

// validateScopeValue 校验并归一化范围值，告警规则与静默规则共用以保证语义一致
func validateScopeValue(ctx context.Context, resourceType int8, scopeType entity.AlertScopeType, scopeValue string) error {
	scopeValue = strings.TrimSpace(scopeValue)
	if scopeValue == "" {
		return errorx.NewBizI(ctx, imsg.ErrRuleScopeValueRequired)
	}

	switch scopeType {
	case entity.AlertScopeResource:
		id, err := strconv.ParseUint(scopeValue, 10, 64)
		if err != nil || id == 0 {
			return errorx.NewBizI(ctx, imsg.ErrRuleScopeValueInvalid, "value", scopeValue)
		}
		if !service.ResourceExists(resourceType, id) {
			return errorx.NewBizI(ctx, imsg.ErrRuleScopeValueInvalid, "value", scopeValue)
		}
	case entity.AlertScopeTagPath:
		var paths []string
		if err := json.Unmarshal([]byte(scopeValue), &paths); err != nil {
			return errorx.NewBizI(ctx, imsg.ErrRuleScopeValueInvalid, "value", scopeValue)
		}
		if len(paths) == 0 {
			return errorx.NewBizI(ctx, imsg.ErrRuleScopeValueInvalid, "value", scopeValue)
		}
		for _, path := range paths {
			if strings.TrimSpace(path) == "" {
				return errorx.NewBizI(ctx, imsg.ErrRuleScopeValueInvalid, "value", scopeValue)
			}
		}
	default:
		return errorx.NewBizI(ctx, imsg.ErrRuleScopeTypeInvalid)
	}
	return nil
}

// validateRuleCondition 校验告警条件组合，并把默认值归一化后回写
func validateRuleCondition(ctx context.Context, rule *entity.AlertRule) error {
	condition := rule.Condition
	if condition == nil || len(condition.Items) == 0 {
		return errorx.NewBizI(ctx, imsg.ErrRuleConditionRequired)
	}
	// 空组合方式按 AND 归一化，与求值侧默认行为保持一致，避免前后端理解不一致
	if condition.Operator == "" {
		condition.Operator = entity.ConditionOperatorAnd
	}
	if !entity.IsValidConditionOperator(condition.Operator) {
		return errorx.NewBizI(ctx, imsg.ErrRuleOperatorInvalid)
	}

	for i := range condition.Items {
		item := &condition.Items[i]
		index := i + 1

		metric := strings.TrimSpace(item.Metric)
		if metric == "" {
			return errorx.NewBizI(ctx, imsg.ErrRuleConditionMetricRequired, "index", index)
		}
		item.Metric = metric

		def, ok := service.FindMetric(rule.ResourceType, metric)
		if !ok {
			return errorx.NewBizI(ctx, imsg.ErrRuleConditionMetricUnknown, "index", index, "metric", metric)
		}

		if item.Compare == "" {
			return errorx.NewBizI(ctx, imsg.ErrRuleConditionCompareRequired, "index", index)
		}
		if !entity.IsValidCompare(item.Compare) {
			return errorx.NewBizI(ctx, imsg.ErrRuleConditionCompareInvalid,
				"index", index, "compare", item.Compare)
		}

		if item.Duration < 0 {
			return errorx.NewBizI(ctx, imsg.ErrRuleConditionDurationInvalid, "index", index)
		}

		// 阈值越界会导致条件恒真/恒假，属于必然失配的非法配置
		if def.HasRange && (item.Value < def.Min || item.Value > def.Max) {
			return errorx.NewBizI(ctx, imsg.ErrRuleConditionValueInvalid,
				"index", index, "value", item.Value, "metric", metric, "min", def.Min, "max", def.Max)
		}
	}
	return nil
}

// validateRuleAdvanced 校验高级配置并归一化零值为默认值
func validateRuleAdvanced(ctx context.Context, rule *entity.AlertRule) error {
	if rule.EvalInterval == 0 {
		rule.EvalInterval = defaultEvalIntvl
	} else if rule.EvalInterval < minEvalInterval || rule.EvalInterval > maxEvalInterval {
		return errorx.NewBizI(ctx, imsg.ErrRuleEvalIntervalInvalid)
	}

	for _, count := range []int{normalizeCount(&rule.TriggerCount), normalizeCount(&rule.RecoveryCount)} {
		if count < minLimitCount || count > maxLimitCount {
			return errorx.NewBizI(ctx, imsg.ErrRuleCountInvalid)
		}
	}

	if rule.NotifyConfig != nil {
		nc := rule.NotifyConfig
		if nc.GroupWait < 0 || nc.GroupInterval < 0 || nc.RepeatInterval < 0 {
			return errorx.NewBizI(ctx, imsg.ErrRuleCountInvalid)
		}
	}
	return nil
}

// normalizeCount 零值回退为 1（不填即不防抖），返回归一化后的值
func normalizeCount(count *int) int {
	if *count <= 0 {
		*count = 1
	}
	return *count
}

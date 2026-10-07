package application

import (
	"context"

	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/service"
	machineapp "mayfly-go/internal/machine/application"
	pkgconsts "mayfly-go/internal/pkg/consts"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/logx"
)

// MachineTriage 用「已启用的告警规则」为机器健康总览判定越线与优先级。
//
// 为什么在告警侧做：阈值、作用域与比较语义都归告警规则所有，机器模块只消费判定结果。
// 机器侧另写一套判定会让总览与告警随时间漂移（同一台机器一边红一边绿），所以这里直接复用
// 引擎用的同一套作用域展开（expandResourceScope）与同一套评估器（service.GetEvaluator）。
//
// 与引擎周期评估的差别是刻意的：这里只问「此刻水位是否越线」，不做 Duration 持续判定、
// 不看静默/抑制、也不产生告警事件——总览要的是巡检视图，不是事件流。
type MachineTriage interface {
	// TriageMachineHealth 就地把规则判定结果回填到每行的 Priority / Triaged / Hits。
	//
	// 规则清单查询失败返回 error（宁可不给视图，也不要给出「全部正常」的假结论）。
	TriageMachineHealth(ctx context.Context, rows []*machineapp.MachineHealth) error
}

type machineTriageAppImpl struct {
	ruleApp          AlertRule            `inject:"T"`
	tagTreeRelateApp tagapp.TagTreeRelate `inject:"T"`
	tagTreeApp       tagapp.TagTree       `inject:"T"`
}

var _ MachineTriage = (*machineTriageAppImpl)(nil)

func (t *machineTriageAppImpl) TriageMachineHealth(ctx context.Context, rows []*machineapp.MachineHealth) error {
	if len(rows) == 0 {
		return nil
	}

	evaluator, ok := service.GetEvaluator(pkgconsts.ResourceTypeMachine)
	if !ok {
		// 机器评估器缺失说明注册链断了：把每行标成未判定，让界面显示「未判定」而非「正常」
		markUntagged(rows)
		logx.Warnf("[alert] machine triage: no evaluator registered for machine resource type")
		return nil
	}

	rules, err := t.ruleApp.ListEnabled()
	if err != nil {
		return err
	}

	rowById := make(map[uint64]*machineapp.MachineHealth, len(rows))
	for _, row := range rows {
		rowById[row.MachineId] = row
	}

	for _, rule := range rules {
		if rule.ResourceType != pkgconsts.ResourceTypeMachine || rule.Condition == nil || len(rule.Condition.Items) == 0 {
			continue
		}
		resourceIds := expandResourceScope(ctx, t.tagTreeRelateApp, t.tagTreeApp, rule.ResourceType, rule.Id, rule.ScopeType, rule.ScopeValue, func(codes []string) []uint64 {
			return evaluator.ResourceIdsByCodes(codes)
		})
		for _, rid := range resourceIds {
			// 规则覆盖到的机器可能不在本次总览里（被权限过滤或非 SSH），跳过即可
			row, covered := rowById[rid]
			if !covered {
				continue
			}
			evaluateRuleOn(ctx, rule, rid, evaluator, row)
		}
	}
	return nil
}

func evaluateRuleOn(ctx context.Context, rule *entity.AlertRule, resourceId uint64, evaluator service.AlertEvaluator, row *machineapp.MachineHealth) {
	result, err := evaluator.Evaluate(ctx, &service.EvalParam{Rule: rule, ResourceId: resourceId})
	if err != nil {
		// 单台评估失败只让该行变成「未判定」，不影响其他机器的视图
		logx.Warnf("[alert] machine triage: evaluate rule[%d] machine[%d] error: %s", rule.Id, resourceId, err.Error())
		row.Triaged = false
		return
	}
	// 与告警引擎同一套 fail-closed：取不到当前值时结论不可靠，不能按「未越线」展示
	if !result.Evaluated {
		row.Triaged = false
		return
	}
	if !result.Triggered {
		return
	}
	applyHits(row, rule, result)
}

// applyHits 记录命中的条件项明细，并把该行优先级压到最差（数值越小越严重）
func applyHits(row *machineapp.MachineHealth, rule *entity.AlertRule, result *service.EvalResult) {
	for _, item := range result.Items {
		if !item.Satisfied {
			continue
		}
		row.Hits = append(row.Hits, &machineapp.HealthHit{
			RuleId:    rule.Id,
			RuleName:  rule.Name,
			Metric:    item.Metric,
			Compare:   item.Compare,
			Threshold: item.Threshold,
			Current:   item.CurrentValue,
			Priority:  int8(rule.Priority),
		})
		if row.Priority == machineapp.NoHealthHit || int8(rule.Priority) > row.Priority {
			row.Priority = int8(rule.Priority)
		}
	}
}

// markUntagged 把全部行标记为未判定
func markUntagged(rows []*machineapp.MachineHealth) {
	for _, row := range rows {
		row.Triaged = false
	}
}

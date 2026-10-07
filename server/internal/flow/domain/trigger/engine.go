package trigger

import (
	"context"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/imsg"
)

// 结论来源标识，前端据此映射展示文案
const (
	// SourceEngine 结论来自引擎自身（策略不可解析、检查项缺失注册等配置异常）
	SourceEngine = "engine"
	// SourceCustomWhen 结论来自自定义触发条件
	SourceCustomWhen = "custom.when"
	// SourceCustomUnless 结论来自自定义豁免条件
	SourceCustomUnless = "custom.unless"
)

// 引擎侧结论的展示文案 key，文案由前端 i18n 承载
const (
	TitlePolicyMissing     = "flow.policy.missing"
	TitlePolicyVersion     = "flow.policy.versionUnsupported"
	TitleCheckUnregistered = "flow.policy.checkUnregistered"
	TitlePolicyUnreadable  = "flow.policy.unreadable"
	TitleCustomWhen        = "flow.policy.customWhen"
	TitleCustomUnless      = "flow.policy.customUnless"
	// TitleFallbackUnhonourable 兜底级别在当前场景落不了地（如机器命令配了「需审批」）
	TitleFallbackUnhonourable = "flow.policy.fallbackUnhonourable"
)

// upgradePolicy 把历史版本的策略结构升级到当前版本，返回 false 表示模型版本无法识别。
// 模型结构演进时在此登记「版本 -> 升级函数」，求值与校验只面对当前版本结构
func upgradePolicy(policy *entity.TriggerPolicy) bool {
	// 未带版本号的历史数据与当前版本结构一致；只判不写，保证求值对入参只读
	return policy.Version <= entity.TriggerPolicyVersion
}

// Evaluate 求值触发策略并给出处置结论。
//
// 语义：
//   - unless 命中直接豁免，优先级高于一切触发结论
//   - 该场景配置了规则时只有命中才产生处置，未命中即放行，避免「只拦写操作」的策略把 SELECT 也拦下
//   - 该场景一条规则都没配置时按 DefaultSeverity 兜底（绑定了流程但尚未配置规则），此时宁可一律审批；
//     兜底级别在该场景落不了地时收敛到最严格的可用级别（见 honourableSeverity）
//   - 多条规则命中取最严格的处置级别；策略不可解析时 fail-closed 抬到需审批
func Evaluate(ctx context.Context, policy *entity.TriggerPolicy, tc *Context) *Decision {
	decision := &Decision{Severity: entity.SeverityDisabled}

	if policy == nil {
		return failClosed(decision, tc, TitlePolicyMissing)
	}
	if !upgradePolicy(policy) {
		return failClosed(decision, tc, TitlePolicyVersion)
	}
	if !policy.HasRuleFor(tc.BizType) {
		// 兜底级别也必须过能力位这道关：没有审批通道的场景（如机器命令）配不出也无法落地「需审批」，
		// 原样返回会让只判「禁止执行」的调用方直接放行 —— 配了策略却静默不拦，比不配更危险
		severity, clamped := honourableSeverity(tc.BizType, policy.DefaultSeverity)
		decision.Severity = severity
		if clamped {
			decision.add(SourceEngine, severity, TitleFallbackUnhonourable, nil).Summary = imsg.TriggerReasonFallbackUnhonourable
		}
		decision.Unknown = tc.Unknown
		return decision
	}

	custom := policy.CustomOf(tc.BizType)
	if custom != nil && evalUnless(ctx, custom, tc, decision) {
		decision.Exempted = true
		decision.Severity = entity.SeverityDisabled
		decision.Unknown = tc.Unknown
		return decision
	}

	evalChecks(ctx, policy, tc, decision)
	if custom != nil {
		evalCustomWhen(ctx, custom, tc, decision)
	}
	decision.Matched = len(decision.Findings) > 0
	decision.Unknown = tc.Unknown
	return decision
}

// evalUnless 返回 true 表示命中豁免条件
func evalUnless(ctx context.Context, custom *entity.CustomCondition, tc *Context, decision *Decision) bool {
	if custom.Unless == nil {
		return false
	}
	matched, err := eval(custom.Unless, tc)
	if err != nil {
		failClosed(decision, tc, TitlePolicyUnreadable)
		return false
	}
	if !matched {
		return false
	}
	decision.add(SourceCustomUnless, entity.SeverityDisabled, TitleCustomUnless, nil)
	return true
}

func evalChecks(ctx context.Context, policy *entity.TriggerPolicy, tc *Context, decision *Decision) {
	for _, config := range policy.Checks {
		// null 元素无法归属场景，与 HasRuleFor 一样按「该场景没配这条」处理：
		// 保存期 validate 会拒掉这种策略，这里只保证历史脏数据不会把业务执行路径打崩
		if config == nil || config.BizType != tc.BizType {
			continue
		}
		def, registered := CheckOf(config.Key)
		if !registered {
			// 检查项被下线或从未注册，风险无法判定，按最严格方向处置并留痕
			failClosed(decision, tc, TitleCheckUnregistered)
			continue
		}

		matched, err := def.Evaluate(ctx, tc, config.Params)
		if err != nil {
			failClosed(decision, tc, TitlePolicyUnreadable)
			continue
		}
		if matched {
			decision.add(def.Key, config.Severity, def.TitleKey, nil).Summary = def.Summary
		}
	}
}

func evalCustomWhen(ctx context.Context, custom *entity.CustomCondition, tc *Context, decision *Decision) {
	if custom.When == nil {
		return
	}
	matched, err := eval(custom.When, tc)
	if err != nil {
		failClosed(decision, tc, TitlePolicyUnreadable)
		return
	}
	if matched {
		decision.add(SourceCustomWhen, custom.Severity, TitleCustomWhen, nil).Summary = imsg.TriggerReasonCustomCondition
	}
}

// failClosedSeverity 给出该场景下 fail-closed 应该落在哪个级别。
//
// 不能写死「需审批」：没有审批回调的场景（如机器命令）配不出也无法落地该级别，
// 兜到它反而等于放行。也不能直接取最严级别：那样会让可审批场景在无法判定时硬拒，
// 把「交给人工确认」这个正常出口跳过了。因此优先「需审批」，不具备时才退到最严可用级别
func failClosedSeverity(bizType string) entity.Severity {
	available := AvailableSeverities(bizType)
	strictest := available[0]
	for _, severity := range available {
		if severity == entity.SeverityRequired {
			return entity.SeverityRequired
		}
		if severity.StrongerThan(strictest) {
			strictest = severity
		}
	}
	return strictest
}

// FailClosedSeverity 该场景在「无法判定」或「兜底级别落不了地」时实际会落在哪个级别。
//
// 注意它不是字面上的「最严级别」：可审批场景的落点是「需审批」（交给人确认，而不是硬拒），
// 只有没有审批通道的场景才退到最严的可落地级别。回显必须由引擎给出：
// 这条规则在前端再算一遍就是第二份真源，算漏一次就会显示成「管理员看到的级别≠实际生效的级别」
func FailClosedSeverity(bizType string) entity.Severity {
	return failClosedSeverity(bizType)
}

// honourableSeverity 把兜底级别收敛到该场景真正能落地的级别，第二个返回值表示是否发生收敛。
//
// 「不处置」永远可落地（它就是放行）；「需审批」要求该场景注册了审批通过后的执行回调，
// 没有回调时这个兜底不会被任何调用方消费，必须抬到最严格的可用级别而不是原样返回
func honourableSeverity(bizType string, severity entity.Severity) (entity.Severity, bool) {
	if severity == entity.SeverityDisabled {
		return severity, false
	}
	for _, available := range AvailableSeverities(bizType) {
		if severity == available {
			return severity, false
		}
	}
	return failClosedSeverity(bizType), true
}

// failClosed 记录一条引擎侧结论并把处置级别抬到该场景最严格的可用级别
func failClosed(decision *Decision, tc *Context, titleKey string) *Decision {
	decision.add(SourceEngine, failClosedSeverity(tc.BizType), titleKey, nil)
	decision.Unknown = tc.Unknown
	return decision
}

func (d *Decision) add(source string, severity entity.Severity, titleKey string, detail map[string]any) *Finding {
	finding := &Finding{Source: source, Severity: severity, Title: titleKey, Detail: detail}
	d.Findings = append(d.Findings, finding)
	if severity.StrongerThan(d.Severity) {
		d.Severity = severity
	}
	return finding
}

package trigger

import (
	"context"
	"strings"
	"testing"

	"mayfly-go/internal/flow/domain/entity"
)

// TestFallbackSeverityFollowsApprovalChannel 兜底级别必须受场景能力位约束。
//
// 这条链此前是断的：规则级校验拦了「没有审批通道就不能配需审批」，兜底级漏拦，
// 于是机器命令这类无审批通道的场景能存下「未配置规则时：需审批」，
// 而调用方只判「禁止执行」——策略看着配好了，实际一条也不拦
func TestFallbackSeverityFollowsApprovalChannel(t *testing.T) {
	const silentBiz = "test_fallback_no_channel_flow"
	RegisterBiz(BizMeta{
		BizType: silentBiz,
		Fields:  []TriggerField{{Key: "cmd", TitleKey: "flow.field.cmd", Group: "risk", Type: TypeString}},
	})

	policy := &entity.TriggerPolicy{
		Version:         entity.TriggerPolicyVersion,
		DefaultSeverity: entity.SeverityRequired,
	}
	tc := NewContext(context.Background(), silentBiz, 1, nil, nil)
	decision := Evaluate(context.Background(), policy, tc)

	if !decision.IsForbidden() {
		t.Fatalf("a fallback that cannot be honoured must escalate to the strictest available action, got %v", decision.Severity)
	}
	if len(decision.Findings) != 1 || decision.Findings[0].Title != TitleFallbackUnhonourable {
		t.Fatalf("the escalation must be visible as a finding, got %+v", decision.Findings)
	}
	// 被拦下的操作者看到的括号里必须有内容，否则只会得到一个不知所以的拒绝
	if reason := decision.Reason(context.Background()); !strings.Contains(reason, "无法落地") {
		t.Fatalf("the block reason must explain the clamp, got %q", reason)
	}
}

func TestFallbackSeverityStaysWhenHonourable(t *testing.T) {
	const approvableBiz = "test_fallback_channel_flow"
	RegisterBiz(BizMeta{
		BizType:    approvableBiz,
		Approvable: true,
		Fields:     []TriggerField{{Key: "cmd", TitleKey: "flow.field.cmd", Group: "risk", Type: TypeString}},
	})

	for _, severity := range []entity.Severity{entity.SeverityRequired, entity.SeverityWarning, entity.SeverityForbidden} {
		if got, clamped := honourableSeverity(approvableBiz, severity); got != severity || clamped {
			t.Errorf("severity %v must stay as configured for an approvable scenario, got %v (clamped=%v)", severity, got, clamped)
		}
	}

	// 「不处置」本来就是放行，任何场景都算可落地，不能被收敛成拦截
	if got, clamped := honourableSeverity("test_fallback_no_channel_flow", entity.SeverityDisabled); got != entity.SeverityDisabled || clamped {
		t.Fatalf("the pass-through fallback must never be escalated, got %v (clamped=%v)", got, clamped)
	}
}

// TestFallbackAppliesPerScenario 兜底按场景生效，不受其它场景的规则影响。
//
// 一份策略同时治理数据库与机器命令时，只给数据库配了规则：
// 若兜底按「整份策略有没有规则」判断，机器侧永远进不到兜底分支，
// 绑定流程等于没绑——这是真机上复现过的静默放行
func TestFallbackAppliesPerScenario(t *testing.T) {
	const dbBiz = "test_mixed_db_flow"
	const silentBiz = "test_mixed_silent_flow"
	RegisterBiz(BizMeta{
		BizType: dbBiz, Approvable: true,
		Fields: []TriggerField{{Key: "sql", TitleKey: "flow.field.sql", Group: "risk", Type: TypeString}},
		Checks: []CheckDef{{
			Key: "mixed.dml", TitleKey: "flow.check.x", BizTypes: []string{dbBiz}, Default: entity.SeverityRequired,
			Evaluate: func(ctx context.Context, tc *Context, params map[string]any) (bool, error) {
				return false, nil
			},
		}},
	})
	RegisterBiz(BizMeta{
		BizType: silentBiz,
		Fields:  []TriggerField{{Key: "cmd", TitleKey: "flow.field.cmd", Group: "risk", Type: TypeString}},
	})

	policy := &entity.TriggerPolicy{
		Version:         entity.TriggerPolicyVersion,
		DefaultSeverity: entity.SeverityRequired,
		Checks: []*entity.CheckConfig{{
			Key: "mixed.dml", BizType: dbBiz, Severity: entity.SeverityRequired,
		}},
	}

	// 数据库侧：配了规则且没命中 → 放行（保留「只拦写操作不该拦 SELECT」的既有语义）
	dbDecision := Evaluate(context.Background(), policy, NewContext(context.Background(), dbBiz, 1, nil, nil))
	if dbDecision.Severity != entity.SeverityDisabled || len(dbDecision.Findings) > 0 {
		t.Fatalf("an unmatched configured scenario must pass through, got %v / %+v", dbDecision.Severity, dbDecision.Findings)
	}

	// 机器侧：一条规则都没配 → 走兜底；兜底在该场景落不了地 → 收敛为禁止执行
	silentDecision := Evaluate(context.Background(), policy, NewContext(context.Background(), silentBiz, 1, nil, nil))
	if !silentDecision.IsForbidden() {
		t.Fatalf("a scenario without any rule must still fall back to the default action, got %v", silentDecision.Severity)
	}
	if len(silentDecision.Findings) != 1 || silentDecision.Findings[0].Title != TitleFallbackUnhonourable {
		t.Fatalf("the machine-side fallback must be explained, got %+v", silentDecision.Findings)
	}
}

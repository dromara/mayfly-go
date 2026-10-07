package trigger

import (
	"context"
	"testing"

	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/pkg/consts"
)

const (
	governedTestBiz   = "test_govern_match_flow"
	governedTestCheck = "test.govern-check"
	shallowTestCheck  = "test.shallow-check"
)

func registerGovernedScenario(t *testing.T) {
	t.Helper()
	RegisterBiz(BizMeta{
		BizType: governedTestBiz,
		// 治理粒度在「实例/凭证/库」这一层
		GovernPaths: [][]int8{{consts.ResourceTypeDbInstance, consts.ResourceTypeAuthCert, consts.ResourceTypeDbName}},
		Checks: []CheckDef{{
			Key: governedTestCheck, TitleKey: "flow.check.dangerousCmd", BizTypes: []string{governedTestBiz},
			Default: entity.SeverityWarning,
			// 注册契约要求每个检查项都带求值函数（TestRegistryContract 会扫全部注册项）
			Evaluate: func(context.Context, *Context, map[string]any) (bool, error) { return false, nil },
		}},
	})
}

// TestPolicyScenariosMustBeGovernedByBoundPaths 配了规则的场景必须真能被绑定资源命中。
//
// 策略编辑器会列出全部已注册场景：管理员给某个场景配好规则，流程定义却只绑了别的资源，
// 运行时永远解析不到那个场景 —— 不拦住就是「看着管住了，实际一条都没管」
func TestPolicyScenariosMustBeGovernedByBoundPaths(t *testing.T) {
	registerGovernedScenario(t)
	policy := &entity.TriggerPolicy{
		Version: entity.TriggerPolicyVersion,
		Checks:  []*entity.CheckConfig{{BizType: governedTestBiz, Key: governedTestCheck, Severity: entity.SeverityWarning}},
	}

	// 绑定层级比治理层级高（实例）：祖先展开会覆盖到库，属于能命中
	if err := ValidatePolicy(policy, WithBoundPaths([][]int8{{consts.ResourceTypeDbInstance}})); err != nil {
		t.Fatalf("binding an ancestor of the governed path must be accepted: %v", err)
	}
	// 绑定层级与治理层级一致
	if err := ValidatePolicy(policy, WithBoundPaths([][]int8{{consts.ResourceTypeDbInstance, consts.ResourceTypeAuthCert, consts.ResourceTypeDbName}})); err != nil {
		t.Fatalf("binding the governed path itself must be accepted: %v", err)
	}
	// 绑定层级比治理层级更深（治理路径是绑定路径的前缀）同样要算命中：
	// 漏掉这一侧会让「浅层治理 + 深层绑定」的组合被误判成永不命中，而它恰恰是祖先展开的常见形态
	shallow := "test_shallow_govern_flow"
	RegisterBiz(BizMeta{
		BizType:     shallow,
		GovernPaths: [][]int8{{consts.ResourceTypeDbInstance}},
		Checks: []CheckDef{{
			Key: shallowTestCheck, TitleKey: "flow.check.dangerousCmd", BizTypes: []string{shallow}, Default: entity.SeverityWarning,
			Evaluate: func(context.Context, *Context, map[string]any) (bool, error) { return false, nil },
		}},
	})
	deepBound := &entity.TriggerPolicy{
		Version: entity.TriggerPolicyVersion,
		Checks:  []*entity.CheckConfig{{BizType: shallow, Key: shallowTestCheck, Severity: entity.SeverityWarning}},
	}
	if err := ValidatePolicy(deepBound, WithBoundPaths([][]int8{{consts.ResourceTypeDbInstance, consts.ResourceTypeAuthCert, consts.ResourceTypeDbName}})); err != nil {
		t.Fatalf("binding deeper than the governed path must be accepted: %v", err)
	}

	// 绑的是完全另一条资源线（机器）：该场景永远命中不上
	if err := ValidatePolicy(policy, WithBoundPaths([][]int8{{consts.ResourceTypeMachine}})); err == nil {
		t.Fatal("a scenario governing none of the bound resources must be rejected, it can never match")
	}

	// 未绑定资源时跳过：保存草稿与尚未选资源的表单不该被这条判据打死
	if err := ValidatePolicy(policy); err != nil {
		t.Fatalf("without bound paths the governance check must be skipped: %v", err)
	}
}

// TestConditionOnlyScenarioIsNotRequiredToDeclareGovernPaths 只提供字段字典的流程内条件场景不治理资源，
// 反查与命中性校验都不该把它算进去，否则流程内部条件会被当成「配了却永不命中」
func TestConditionOnlyScenarioIsNotRequiredToDeclareGovernPaths(t *testing.T) {
	RegisterBiz(BizMeta{BizType: "test_condition_only_flow", ConditionOnly: true})
	if got := GovernedBizTypes([][]int8{{consts.ResourceTypeMachine}}); len(got) != 0 {
		for _, bizType := range got {
			if bizType == "test_condition_only_flow" {
				t.Fatalf("a condition-only registration must not be reported as governing resources, got %v", got)
			}
		}
	}
}

// TestGovernanceDeclarationProblems 治理路径的声明自检。
//
// 段值写错不会有任何运行时报错，只会让场景「选不到 / 配了不命中」，所以必须在启动期就报出来；
// 而 ConditionOnly 的流程内条件场景本就不治理资源，不该被这条判据误伤
func TestGovernanceDeclarationProblems(t *testing.T) {
	if problems := GovernanceDeclarationProblems(BizMeta{BizType: "x", ConditionOnly: true}); len(problems) != 0 {
		t.Fatalf("a condition-only registration governs nothing by design, got %v", problems)
	}
	if problems := GovernanceDeclarationProblems(BizMeta{BizType: "x"}); len(problems) == 0 {
		t.Fatal("a scenario without any governance path can never be bound, it must be reported")
	}
	problems := GovernanceDeclarationProblems(BizMeta{BizType: "x", GovernPaths: [][]int8{{consts.ResourceTypeDbInstance, 127}}})
	if len(problems) != 1 {
		t.Fatalf("exactly one problem expected for the unknown resource type, got %v", problems)
	}
	if problems := GovernanceDeclarationProblems(BizMeta{BizType: "x", GovernPaths: [][]int8{{consts.ResourceTypeMachine}}}); len(problems) != 0 {
		t.Fatalf("a valid declaration must stay silent, got %v", problems)
	}
}

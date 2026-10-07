package application

import (
	"context"
	"strings"
	"testing"

	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/trigger"
)

func machinePolicy(severity flowentity.Severity) *flowentity.TriggerPolicy {
	return &flowentity.TriggerPolicy{
		Version: flowentity.TriggerPolicyVersion,
		Checks: []*flowentity.CheckConfig{
			{Key: machineCheckUnsafe, BizType: MachineRunCmdFlowBizType, Severity: severity},
		},
	}
}

func evalMachine(policy *flowentity.TriggerPolicy, cmd string) *trigger.Decision {
	tc := trigger.NewContext(context.Background(), MachineRunCmdFlowBizType, 1, map[string]string{machineFieldCmdText: cmd}, nil)
	return trigger.Evaluate(context.Background(), policy, tc)
}

// 不可静态审计的 shell 结构必须命中：重定向与命令替换会让「看到的命令」不等于「实际执行的命令」
func TestMachineUnsafePatternCheck(t *testing.T) {
	cases := []struct {
		cmd     string
		matched bool
	}{
		{"echo $(whoami)", true},
		{"cat /etc/passwd > /tmp/out", true},
		{"sh -c `id`", true},
		{"echo hi ${HOME}", true},
		{"ls -la", false},
		{"ps aux", false},
		// 危险命令本身不是「不可审计结构」，它由命令黑名单负责，两者不能混为一谈
		{"rm -rf /tmp/x", false},
		{"shutdown -h now", false},
	}

	for _, tc := range cases {
		decision := evalMachine(machinePolicy(flowentity.SeverityForbidden), tc.cmd)
		got := decision.Severity == flowentity.SeverityForbidden
		if got != tc.matched {
			t.Fatalf("%q: expected matched=%v, got severity=%v findings=%+v", tc.cmd, tc.matched, decision.Severity, decision.Findings)
		}
	}
}

// 未配置任何规则时不得凭空拦截：策略只反映管理员显式配置的级别
func TestMachinePolicyWithoutRulesUsesFallback(t *testing.T) {
	policy := flowentity.NewTriggerPolicy(flowentity.SeverityDisabled)
	if decision := evalMachine(policy, "rm -rf /tmp/x"); decision.Severity != flowentity.SeverityDisabled {
		t.Fatalf("an empty policy must not block anything, got %v", decision.Severity)
	}

	// 「绑定了流程但一条规则都没配」按兜底级别处置。
	// 但兜底配「需审批」时机器场景拿不到这个结论：CheckMachineCmd 只消费「禁止执行」，
	// 原样返回 Required 等于命令照常执行且不留痕迹（看着配了策略、实际一条不拦）。
	// 求值期把它收敛到该场景能落地的最严级别，所以这里必须是「禁止执行」而不是「需审批」
	policy.DefaultSeverity = flowentity.SeverityRequired
	decision := evalMachine(policy, "ls -la")
	if decision.Severity != flowentity.SeverityForbidden {
		t.Fatalf("a fallback that cannot be honoured must escalate to the strictest available action, got %v", decision.Severity)
	}
	if !decision.IsForbidden() {
		t.Fatalf("the escalated machine fallback must actually block the command, got %+v", decision)
	}
}

// 黑名单检查项按机器标签取命令配置：同一条命令在不同机器上结论可以不同，
// 这是「按资源就近生效」的语义，不能退化成全局开关
func TestMachineBlacklistUsesResourceScopedRules(t *testing.T) {
	def, ok := trigger.CheckOf(machineCheckBlacklisted)
	if !ok {
		t.Fatalf("the blacklist check must be registered")
	}
	// 命令配置取不到时必须冒错而不是返回「未命中」：
	// 静默判不命中等于「查不到黑名单就放行」，是安全规则的 fail-open
	tc := trigger.NewContext(context.Background(), MachineRunCmdFlowBizType, 1, map[string]string{machineFieldCmdText: "rm -rf /"}, nil)
	if _, err := def.Evaluate(context.Background(), tc, nil); err == nil {
		t.Fatalf("an unavailable command configuration must surface as an error so the engine fails closed")
	}

	// 引擎侧确认：无法判定要抬到该场景最严的可用级别（机器场景没有审批通道，即「禁止执行」）
	policy := &flowentity.TriggerPolicy{
		Version: flowentity.TriggerPolicyVersion,
		Checks:  []*flowentity.CheckConfig{{Key: machineCheckBlacklisted, BizType: MachineRunCmdFlowBizType, Severity: flowentity.SeverityWarning}},
	}
	decision := trigger.Evaluate(context.Background(), policy, tc)
	if decision.Severity != flowentity.SeverityForbidden {
		t.Fatalf("an unevaluable rule must fail closed to the strictest available severity, got %v", decision.Severity)
	}
}

// 豁免条件必须能放行：给运维留口子是命令策略的刚需，否则只能整条关掉的粗粒度二选一
func TestMachineUnlessExempts(t *testing.T) {
	policy := &flowentity.TriggerPolicy{
		Version: flowentity.TriggerPolicyVersion,
		Checks:  []*flowentity.CheckConfig{{Key: machineCheckUnsafe, BizType: MachineRunCmdFlowBizType, Severity: flowentity.SeverityForbidden}},
		Customs: []*flowentity.CustomCondition{{
			BizType:  MachineRunCmdFlowBizType,
			Severity: flowentity.SeverityWarning,
			Unless: &flowentity.RuleNode{
				Kind: flowentity.NodeKindCondition, Field: machineFieldCmd, Op: trigger.OpEq, Value: "rm",
			},
		}},
	}

	if decision := evalMachine(policy, "rm -rf /tmp/x"); !decision.Exempted {
		t.Fatalf("rm must be exempted by the unless condition, got %+v", decision.Findings)
	}
	if decision := evalMachine(policy, "shutdown -h now"); decision.Exempted {
		t.Fatalf("only the exempted command may pass, got %+v", decision.Findings)
	}
}

// 命令名与命令段名由原文派生，条件才能按命令粒度而非整串文本配置
func TestMachineDerivedFields(t *testing.T) {
	tc := trigger.NewContext(context.Background(), MachineRunCmdFlowBizType, 1, map[string]string{machineFieldCmdText: "cat /etc/passwd | grep root"}, nil)

	names, found, err := tc.FieldValue(mustField(t, machineFieldCmdNames))
	if err != nil || !found {
		t.Fatalf("the command names must be derivable, got %v %v", found, err)
	}
	list, ok := names.([]string)
	if !ok || len(list) < 2 || list[0] != "cat" || list[1] != "grep" {
		t.Fatalf("both pipeline segments must be reported, got %v", names)
	}

	single := trigger.NewContext(context.Background(), MachineRunCmdFlowBizType, 1, map[string]string{machineFieldCmdText: "rm -rf /tmp/x"}, nil)
	name, found, err := single.FieldValue(mustField(t, machineFieldCmd))
	if err != nil || !found || name != "rm" {
		t.Fatalf("the leading command name must be derived, got %v %v %v", name, found, err)
	}
}

func mustField(t *testing.T, key string) trigger.TriggerField {
	t.Helper()
	meta, ok := trigger.BizMetaOf(MachineRunCmdFlowBizType)
	if !ok {
		t.Fatalf("the machine scenario must be registered")
	}
	for _, field := range meta.Fields {
		if field.Key == key {
			return field
		}
	}
	t.Fatalf("the field %q must be registered for the machine scenario", key)
	return trigger.TriggerField{}
}

// 默认规则包必须能通过保存校验，否则升级后流程定义一保存就被拒
func TestDefaultMachineRulePackPassesValidation(t *testing.T) {
	policy := &flowentity.TriggerPolicy{
		Version:         flowentity.TriggerPolicyVersion,
		DefaultSeverity: flowentity.SeverityDisabled,
		Checks: []*flowentity.CheckConfig{
			{Key: machineCheckBlacklisted, BizType: MachineRunCmdFlowBizType, Severity: flowentity.SeverityForbidden},
			{Key: machineCheckUnsafe, BizType: MachineRunCmdFlowBizType, Severity: flowentity.SeverityForbidden},
		},
	}
	if err := trigger.ValidatePolicy(policy); err != nil {
		t.Fatalf("the default machine rule pack must be valid, got %v", err)
	}
}

// 机器场景没有审批回调，因此「需审批」不可配置：工单批完没人把命令执行回去
func TestMachineScenarioHasNoApprovalChannel(t *testing.T) {
	for _, severity := range trigger.AvailableSeverities(MachineRunCmdFlowBizType) {
		if severity == flowentity.SeverityRequired {
			t.Fatalf("the machine scenario must not offer the required severity without an approval handler")
		}
	}

	policy := &flowentity.TriggerPolicy{
		Version: flowentity.TriggerPolicyVersion,
		Checks:  []*flowentity.CheckConfig{{Key: machineCheckUnsafe, BizType: MachineRunCmdFlowBizType, Severity: flowentity.SeverityRequired}},
	}
	if err := trigger.ValidatePolicy(policy); err == nil {
		t.Fatalf("configuring required on a scenario without an approval handler must be rejected")
	}
}

// 命令过滤表达式在保存时就要拒掉坏正则：
// 一旦落库，运行期只能选择跳过该规则（安全规则静默失效）或按命中处理（误拦正常命令）
func TestValidateCmdPatternsRejectsBrokenPattern(t *testing.T) {
	app := &machineCmdConfAppImpl{}
	if err := app.ValidateCmdPatterns([]string{"rm\\s+-rf", "shutdown"}); err != nil {
		t.Fatalf("valid patterns must pass, got %v", err)
	}
	err := app.ValidateCmdPatterns([]string{"([unclosed"})
	if err == nil {
		t.Fatalf("an uncompilable pattern must be rejected at save time")
	}
	if !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("the rejection must report the offending pattern, got %v", err)
	}
}

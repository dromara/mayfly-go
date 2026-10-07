package application

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/trigger"
	flowimsg "mayfly-go/internal/flow/imsg"
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"

	"github.com/stretchr/testify/require"
)

// governed 走与保存校验完全相同的一段链路：标签路径 → 资源类型路径 → 治理场景
func governed(codePaths []string) []string {
	return trigger.GovernedBizTypes(boundResourcePaths(codePaths))
}

func flowVars(values map[string]any) collx.M {
	return collx.M(values)
}

func matchFlowCondition(t *testing.T, node *entity.RuleNode, vars collx.M) bool {
	t.Helper()
	matched, err := IsUserTaskComplete(context.Background(), node, vars)
	if err != nil {
		t.Fatalf("the flow condition must be decidable, got %v", err)
	}
	return matched
}

// 或签与会签都表达成「计数满足阈值」，与旧的模板字符串写法同义，
// 但条件树是可见的：管理员在设计器里能直接看到比较的是哪个字段、和什么值比
func TestUserTaskCompletionModes(t *testing.T) {
	orSign := &entity.RuleNode{Kind: entity.NodeKindCondition, Field: flowFieldNrOfCompleted, Op: trigger.OpGte, Value: 1}
	andSign := &entity.RuleNode{Kind: entity.NodeKindCondition, Field: flowFieldNrOfCompletedRate, Op: trigger.OpGte, Value: 1}
	majority := &entity.RuleNode{Kind: entity.NodeKindCondition, Field: flowFieldNrOfCompletedRate, Op: trigger.OpGte, Value: 0.5}

	cases := []struct {
		name        string
		node        *entity.RuleNode
		all         any
		completed   any
		wantComplet bool
	}{
		{"或签-无人审批", orSign, 3, 0, false},
		{"或签-一人审批", orSign, 3, 1, true},
		{"会签-差一人", andSign, 3, 2, false},
		{"会签-全部通过", andSign, 3, 3, true},
		{"过半-两人中的一人", majority, 2, 1, true},
		{"过半-三人中的一人", majority, 3, 1, false},
		{"过半-三人中的两人", majority, 3, 2, true},
	}

	for _, tc := range cases {
		vars := flowVars(map[string]any{flowFieldNrOfAll: tc.all, flowFieldNrOfCompleted: tc.completed})
		if got := matchFlowCondition(t, tc.node, vars); got != tc.wantComplet {
			t.Fatalf("%s: expected completed=%v, got %v", tc.name, tc.wantComplet, got)
		}
	}
}

// 计数经 JSON 存取后整数会变成 float64，直接写任务变量时又是 int，
// 两种形态必须都能比，否则「流程实例恢复后判不出完成」会随机出现
func TestUserTaskCompletionNormalisesNumberShapes(t *testing.T) {
	node := &entity.RuleNode{Kind: entity.NodeKindCondition, Field: flowFieldNrOfAll, Op: trigger.OpGte, Value: 3}
	for _, all := range []any{3, int64(3), float64(3), "3"} {
		vars := flowVars(map[string]any{flowFieldNrOfAll: all, flowFieldNrOfCompleted: float64(0)})
		if !matchFlowCondition(t, node, vars) {
			t.Fatalf("the counter %v (%T) must compare as a number", all, all)
		}
	}
}

// 总数缺失或为 0 时不能算出比例 0：那会被读成「还没有人通过」，
// 而真实情况是「无从判断」，条件应当不成立而不是给出一个看起来合理的数
func TestCompletionRateStaysUnknownWithoutATotal(t *testing.T) {
	node := &entity.RuleNode{Kind: entity.NodeKindCondition, Field: flowFieldNrOfCompletedRate, Op: trigger.OpLt, Value: 1}

	for name, vars := range map[string]collx.M{
		"没有总数":  flowVars(map[string]any{flowFieldNrOfCompleted: 1}),
		"总数为 0": flowVars(map[string]any{flowFieldNrOfAll: 0, flowFieldNrOfCompleted: 0}),
	} {
		if matchFlowCondition(t, node, vars) {
			t.Fatalf("%s: an undecidable rate must not satisfy a comparison", name)
		}
	}
}

// 审批结果在库中是数字状态码，条件里用符号：
// 状态码一旦调整，历史流程定义里的条件就会判错，而符号取值不依赖编码
func TestApprovalResultMatchesBySymbol(t *testing.T) {
	accepted := &entity.RuleNode{Kind: entity.NodeKindCondition, Field: flowFieldApprovalResult, Op: trigger.OpEq, Value: ApprovalResultCompleted}

	vars := flowVars(map[string]any{flowFieldApprovalResult: entity.ProcinstTaskStatusCompleted})
	if !matchFlowCondition(t, accepted, vars) {
		t.Fatalf("the numeric task status must resolve to the %q option", ApprovalResultCompleted)
	}

	// 同一个条件也要能匹配已经写成符号的变量，避免两类调用点各写一种形态
	if !matchFlowCondition(t, accepted, flowVars(map[string]any{flowFieldApprovalResult: ApprovalResultCompleted})) {
		t.Fatalf("a symbolic approval result must match as well")
	}

	rejected := &entity.RuleNode{Kind: entity.NodeKindCondition, Field: flowFieldApprovalResult, Op: trigger.OpEq, Value: ApprovalResultBack}
	if matchFlowCondition(t, rejected, vars) {
		t.Fatalf("an approval must not match the returned branch")
	}
}

// 认不出的状态码按「无法判定」处理：猜一个结果会让流程走进某一条分支，
// 而分支走错的代价是跳过本该存在的审批
func TestUnknownApprovalResultIsNotMatched(t *testing.T) {
	for _, value := range []string{ApprovalResultCompleted, ApprovalResultBack, ApprovalResultReject, ApprovalResultCanceled, ApprovalResultProcessing} {
		node := &entity.RuleNode{Kind: entity.NodeKindCondition, Field: flowFieldApprovalResult, Op: trigger.OpEq, Value: value}
		if matchFlowCondition(t, node, flowVars(map[string]any{flowFieldApprovalResult: entity.ProcinstTaskStatus(99)})) {
			t.Fatalf("an unmappable status code must not match %q", value)
		}
	}
}

// 未配置条件即默认流转：这是流程图里绝大多数连线的形态，
// 若按「条件不成立」处理会让所有没写条件的连线都走不通
func TestEdgeWithoutConditionAlwaysMatches(t *testing.T) {
	node, err := newExecutionCtxWithFlow(t, `{"nodes":[{"name":"A","key":"a","type":"start"},{"name":"B","key":"b","type":"end"}],"edges":[{"name":"默认","key":"e1","sourceNodeKey":"a","targetNodeKey":"b"}]}`).GetNextNode(flowVars(map[string]any{}))
	if err != nil {
		t.Fatalf("an edge without a condition must be taken, got %v", err)
	}
	if node == nil || node.Key != "b" {
		t.Fatalf("the default edge must lead to the target node, got %+v", node)
	}
}

// 连线条件解析失败必须冒错并定位到那条连线：
// 静默按不成立处理会让流程停在原地，运维只能从「为什么没走下去」倒推配置问题
func TestBrokenEdgeConditionReportsTheOffendingEdge(t *testing.T) {
	ec := newExecutionCtxWithFlow(t, `{"nodes":[{"name":"A","key":"a","type":"start"},{"name":"B","key":"b","type":"end"}],"edges":[{"name":"金额较大","key":"e1","sourceNodeKey":"a","targetNodeKey":"b","extra":{"condition":"{{ eq .nrOfAll 3 }}"}}]}`)

	_, err := ec.GetNextNode(flowVars(map[string]any{flowFieldNrOfAll: 3}))
	if err == nil {
		t.Fatalf("a leftover template string must be reported instead of silently ignored")
	}
	if got := err.Error(); !strings.Contains(got, "金额较大") {
		t.Fatalf("the error must name the edge so the designer can locate it, got %q", got)
	}
}

// 条件引用的字段确实没提供时按不成立处理，流程原地等待而不是走进分支：
// 无后继节点是可恢复的（补上变量再推进），走错分支不可恢复
func TestEdgeConditionOnMissingVariableIsNotMatched(t *testing.T) {
	flow := `{"nodes":[{"name":"A","key":"a","type":"start"},{"name":"B","key":"b","type":"end"}],"edges":[{"name":"通过","key":"e1","sourceNodeKey":"a","targetNodeKey":"b","extra":{"condition":{"kind":"condition","field":"approvalResult","op":"eq","value":"completed"}}}]}`

	// 审批结果还没产生：带条件的连线不能走
	node, err := newExecutionCtxWithFlow(t, flow).GetNextNode(flowVars(map[string]any{}))
	if err != nil {
		t.Fatalf("a missing variable must not fail the evaluation, got %v", err)
	}
	if node != nil {
		t.Fatalf("no branch may be taken while the approval result is unknown, got %+v", node)
	}

	// 同一个条件在审批通过后必须能走出那一条
	node, err = newExecutionCtxWithFlow(t, flow).GetNextNode(flowVars(map[string]any{flowFieldApprovalResult: entity.ProcinstTaskStatusCompleted}))
	if err != nil {
		t.Fatalf("the completed approval must be decidable, got %v", err)
	}
	if node == nil || node.Key != "b" {
		t.Fatalf("the approved branch must be taken, got %+v", node)
	}
}

// newExecutionCtxWithFlow 构造一条只带着流程图的执行流，用于直接验证连线条件的判定路径
func newExecutionCtxWithFlow(t *testing.T, flowDef string) *ExecutionCtx {
	t.Helper()
	procinst := &entity.Procinst{ProcdefId: 1, FlowDef: flowDef}
	execution := &entity.Execution{NodeKey: "a", Vars: collx.M{}}
	return NewExecutionCtx(context.Background(), procinst, execution)
}

// 审批模式预置必须能通过自己的字段字典校验：预置与字段字典同处注册，
// 一旦字段改名而预置没跟上，管理员点一下「会签」就会得到一条无法保存的条件
func TestApprovalModePresetsAreValid(t *testing.T) {
	presets := flowConditionPresets()
	if len(presets) == 0 {
		t.Fatalf("the approval modes must be registered as presets")
	}
	for _, preset := range presets {
		if preset.TitleKey == "" || preset.DescriptionKey == "" {
			t.Fatalf("the preset %q must carry both a title and a description", preset.Key)
		}
		if !trigger.HasCondition(preset.RuleNode) {
			t.Fatalf("the preset %q resolves to no constraint at all", preset.Key)
		}
		if err := trigger.ValidateCondition(FlowInstanceBizType, preset.RuleNode); err != nil {
			t.Fatalf("the preset %q must be a valid condition, got %v", preset.Key, err)
		}
	}
}

// 三种审批模式必须互不相同，否则界面上会是三个按下去结果一样的按钮
func TestApprovalModePresetsAreDistinct(t *testing.T) {
	seen := map[string]string{}
	for _, preset := range flowConditionPresets() {
		signature := jsonx.ToStr(preset.RuleNode)
		if other, exist := seen[signature]; exist {
			t.Fatalf("the presets %q and %q produce the same condition", other, preset.Key)
		}
		seen[signature] = preset.Key
	}
}

// 判定用的取值必须与预置引用的字段同源：预置写的是 nrOfCompletedRate，
// 求值就要能从「已完成 / 总数」派生出它，否则套用好按钮却永远判不出完成
func TestPresetsEvaluateAgainstRealCounters(t *testing.T) {
	andSign := presetNode(t, "andSign")
	vars := flowVars(map[string]any{flowFieldNrOfAll: 3, flowFieldNrOfCompleted: 3})
	if !matchFlowCondition(t, andSign, vars) {
		t.Fatalf("three of three approvals must satisfy the and-sign preset")
	}
	if matchFlowCondition(t, andSign, flowVars(map[string]any{flowFieldNrOfAll: 3, flowFieldNrOfCompleted: 2})) {
		t.Fatalf("two of three approvals must not satisfy the and-sign preset")
	}

	majority := presetNode(t, "majoritySign")
	if !matchFlowCondition(t, majority, flowVars(map[string]any{flowFieldNrOfAll: 3, flowFieldNrOfCompleted: 2})) {
		t.Fatalf("two of three approvals must satisfy the majority preset")
	}

	orSign := presetNode(t, "orSign")
	if !matchFlowCondition(t, orSign, flowVars(map[string]any{flowFieldNrOfAll: 3, flowFieldNrOfCompleted: 1})) {
		t.Fatalf("a single approval must satisfy the or-sign preset")
	}
}

func presetNode(t *testing.T, key string) *entity.RuleNode {
	t.Helper()
	for _, preset := range flowConditionPresets() {
		if preset.Key == key {
			return preset.RuleNode
		}
	}
	t.Fatalf("the preset %q is not registered", key)
	return nil
}

// 只改名、换标签的保存不产生策略变更：时间线里塞满「什么都没改」的条目，
// 真正的改动反而被淹没
func TestPolicyTextChangedDetectsRealEdits(t *testing.T) {
	base := &entity.TriggerPolicy{
		Version:         entity.TriggerPolicyVersion,
		DefaultSeverity: entity.SeverityRequired,
		Checks:          []*entity.CheckConfig{{Key: "db.dml-without-where", BizType: "db_sql_exec_flow", Severity: entity.SeverityRequired}},
	}
	same := jsonx.ToStr(base)
	reparsed := new(entity.TriggerPolicy)
	if err := json.Unmarshal([]byte(same), reparsed); err != nil {
		t.Fatalf("the serialised policy must be readable again: %v", err)
	}
	if policyTextChanged(base, reparsed) {
		t.Fatalf("an identical policy read back from storage must not count as a change")
	}

	relaxed := new(entity.TriggerPolicy)
	if err := json.Unmarshal([]byte(same), relaxed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	relaxed.Checks[0].Severity = entity.SeverityWarning
	if !policyTextChanged(base, relaxed) {
		t.Fatalf("changing a rule severity must be recorded")
	}

	if !policyTextChanged(base, nil) {
		t.Fatalf("removing the whole policy is the biggest change of all")
	}
	if policyTextChanged(nil, nil) {
		t.Fatalf("two absent policies are not a change")
	}
}

// 试算与断言必须和保存共用同一份条件组解析能力：
// 只在校验时挂解析器的话，引用条件组的策略在试算页会必然报「无法展开引用」，功能等于不可用
func TestSimulateTriggerResolvesSegmentReferences(t *testing.T) {
	segmentApp, _, _ := setupRuleSegmentTest(t)
	require.NoError(t, segmentApp.SaveRuleSegment(context.Background(), newSegment("orders_only", "只碰订单表", segTestBizType, conditionLeaf("sql", "contains", "orders"))))

	// 把试算用到的校验路径与真实服务对齐：这里直接验证带引用的策略能通过校验并展开求值
	policy := &entity.TriggerPolicy{
		Version:         entity.TriggerPolicyVersion,
		DefaultSeverity: entity.SeverityDisabled,
		Customs: []*entity.CustomCondition{{
			BizType:  segTestBizType,
			Severity: entity.SeverityForbidden,
			When:     segmentLeaf("orders_only"),
		}},
	}
	if err := trigger.ValidatePolicy(policy, trigger.WithSegmentResolver(segmentApp.Resolver(context.Background()))); err != nil {
		t.Fatalf("a policy referencing an existing condition group must pass validation, got %v", err)
	}

	tc := trigger.NewContext(context.Background(), segTestBizType, 0, map[string]string{"sql": "select * from orders"}, nil).
		WithSegments(segmentApp.Resolver(context.Background()))
	decision := trigger.Evaluate(context.Background(), policy, tc)
	if decision.Severity != entity.SeverityForbidden {
		t.Fatalf("the referenced group must drive the decision, got %v", decision.Severity)
	}

	// 悬空引用在校验期就要拦住，而不是等到执行报「无法判定」
	dangling := &entity.TriggerPolicy{Version: entity.TriggerPolicyVersion, DefaultSeverity: entity.SeverityDisabled, Customs: []*entity.CustomCondition{{
		BizType: segTestBizType, Severity: entity.SeverityForbidden, When: segmentLeaf("ghost"),
	}}}
	if err := trigger.ValidatePolicy(dangling, trigger.WithSegmentResolver(segmentApp.Resolver(context.Background()))); err == nil {
		t.Fatalf("a dangling condition group reference must be rejected at save/simulate time")
	}
}

// TestValidateConditionRejectsHalfFilledRows 条件树「有壳子没内容」必须保存前就拒。
//
// 只选了字段没填比较值这类半成品，运行期求值会得到与配置者预期相反的结果；
// 后端放行而界面显示成一空行，管理员会以为条件已经生效
func TestValidateConditionRejectsHalfFilledRows(t *testing.T) {
	groupOf := func(items ...*entity.RuleNode) *entity.RuleNode {
		return &entity.RuleNode{Kind: entity.NodeKindGroup, Logic: entity.LogicAll, Items: items}
	}

	cases := []struct {
		name  string
		node  *entity.RuleNode
		valid bool
	}{
		{name: "未配置条件即默认流转", node: nil, valid: true},
		{
			name:  "完整条件",
			node:  groupOf(&entity.RuleNode{Kind: entity.NodeKindCondition, Field: "nrOfCompleted", Op: "gte", Value: 1.0}),
			valid: true,
		},
		{
			name:  "选了字段没填比较值",
			node:  groupOf(&entity.RuleNode{Kind: entity.NodeKindCondition, Field: "nrOfCompleted", Op: "gte"}),
			valid: false,
		},
		{
			name:  "比较值为空串",
			node:  groupOf(&entity.RuleNode{Kind: entity.NodeKindCondition, Field: "nrOfCompleted", Op: "gte", Value: ""}),
			valid: false,
		},
		{
			name:  "占位行没选字段",
			node:  groupOf(&entity.RuleNode{Kind: entity.NodeKindCondition}),
			valid: false,
		},
		{name: "空分组", node: groupOf(), valid: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := trigger.ValidateCondition(FlowInstanceBizType, tc.node)
			if tc.valid {
				require.NoError(t, err)
				return
			}
			require.Error(t, err, "半成品条件必须被拒, got nil")
			require.NotNil(t, trigger.AsError(err), "拒绝原因必须可本地化")
		})
	}
}

// TestUndecidableRateIsRecordedAsUnknown 派生字段算不出来时必须留痕。
//
// 完成率在「会签总数为 0」时算不出取值。如果把它当成「取到了 nil」，条件只是判假，
// Unknown 与日志都没有痕迹：管理员看到的是「审批都通过了，流程却停在原地」，
// 而且换一台机器、换一个实例都可能重现不了
func TestUndecidableRateIsRecordedAsUnknown(t *testing.T) {
	andSign := presetNode(t, "andSign")

	tc := newFlowConditionContext(context.Background(), flowVars(map[string]any{flowFieldNrOfAll: 0, flowFieldNrOfCompleted: 0}), 1)
	matched, err := trigger.MatchCondition(andSign, tc)
	require.NoError(t, err, "算不出来不是求值失败，不能打断流转")
	require.False(t, matched, "无从判定不能当成条件成立")
	require.Contains(t, tc.Unknown, flowFieldNrOfCompletedRate, "算不出来的字段必须进 Unknown 留痕")

	// 取到值时不应留痕：否则试算面板会把正常判定报成「未能判定」
	resolved := newFlowConditionContext(context.Background(), flowVars(map[string]any{flowFieldNrOfAll: 2, flowFieldNrOfCompleted: 2}), 1)
	require.True(t, matchFlowCondition(t, andSign, flowVars(map[string]any{flowFieldNrOfAll: 2, flowFieldNrOfCompleted: 2})))
	require.NotContains(t, resolved.Unknown, flowFieldNrOfCompletedRate)
}

// TestGovernedScenariosFromCodePaths 兜底级别的场景判定要认资源类型，不认标签。
//
// 流程定义绑的是标签路径（`默认/机器|code/`、`默认/2|code/`），
// 只有从路径里的资源类型段推场景，才能发现「这台资源根本没有审批通道」
func TestGovernedScenariosFromCodePaths(t *testing.T) {
	// 真实场景注册在 db/redis/machine 各自的包里，不进入本包的测试二进制，
	// 所以这里用自注册的场景验证「资源类型 → 场景」这条推导本身
	RegisterTriggerBiz(trigger.BizMeta{
		BizType:     "gov_db_flow",
		Approvable:  true,
		GovernPaths: [][]int8{{consts.ResourceTypeDbInstance, consts.ResourceTypeAuthCert, consts.ResourceTypeDbName}},
		Fields:      []trigger.TriggerField{{Key: "sql", TitleKey: "flow.field.sql", Group: "risk", Type: trigger.TypeString}},
	})
	RegisterTriggerBiz(trigger.BizMeta{
		BizType:     "gov_machine_flow",
		GovernPaths: [][]int8{{consts.ResourceTypeMachine}},
		Fields:      []trigger.TriggerField{{Key: "cmd", TitleKey: "flow.field.cmd", Group: "risk", Type: trigger.TypeString}},
	})

	// 机器实例段（类型 1）只推出机器场景：它没有审批通道，兜底配「需审批」就该被拒
	require.Equal(t, []string{"gov_machine_flow"}, governed([]string{"默认/1|UMaq3B2KHkKg/"}))

	// 同一场景被多种资源类型命中时不重复给出
	require.Equal(t, []string{"gov_db_flow"}, governed([]string{"默认/2|YXJzgY38XO/", "默认/22|T6txf4tTxU/"}))

	require.ElementsMatch(t, []string{"gov_db_flow", "gov_machine_flow"}, governed([]string{"默认/2|YXJzgY38XO/", "默认/1|UMaq3B2KHkKg/"}))

	// 纯标签路径推不出任何场景：此时跳过兜底级别校验，而不是误报成「配置无效」
	require.Empty(t, governed([]string{"mybatis/"}))
}

// TestFallbackSeverityRejectedForMachineOnlyProcdef 兜底级别在不可审批的流程定义上必须存不进去。
//
// 求值期的收敛只是最后一道防线：能在这里拒掉，管理员当场就知道该改级别，
// 而不是等运维敲命令被硬拦后回头查是谁配的策略
func TestFallbackSeverityRejectedForMachineOnlyProcdef(t *testing.T) {
	policy := &entity.TriggerPolicy{
		Version:         entity.TriggerPolicyVersion,
		DefaultSeverity: entity.SeverityRequired,
	}

	err := trigger.ValidatePolicy(policy, trigger.WithFallbackScenarios([]string{"gov_machine_flow"}))
	require.Error(t, err, "机器场景没有审批通道，兜底「需审批」落不了地")
	policyErr := trigger.AsError(err)
	require.NotNil(t, policyErr)
	require.Equal(t, i18n.MsgId(flowimsg.ReasonSeverityNoChannel), policyErr.MsgId)
	require.Contains(t, policyErr.Source, "defaultSeverity")

	// 可审批场景不受影响；「直接放行」这种兜底也永远合法
	require.NoError(t, trigger.ValidatePolicy(policy, trigger.WithFallbackScenarios([]string{"gov_db_flow"})))
	policy.DefaultSeverity = entity.SeverityDisabled
	require.NoError(t, trigger.ValidatePolicy(policy, trigger.WithFallbackScenarios([]string{"gov_machine_flow"})))
}

// TestFallbackSeverityAllowedForMixedResources 一份策略同时治理可审批与不可审批资源时，
// 兜底「需审批」必须能存下去。
//
// 不能因为机器侧落不了地就整份拒存：管理员对数据库侧的「未命中规则也要审批」是合法意图，
// 拒存会让人直接去掉兜底（反而让数据库失去保护）或拆流程定义（门槛更高）；
// 机器侧由求值期按场景收敛为禁止执行，两边语义各自正确
func TestFallbackSeverityAllowedForMixedResources(t *testing.T) {
	policy := &entity.TriggerPolicy{
		Version:         entity.TriggerPolicyVersion,
		DefaultSeverity: entity.SeverityRequired,
	}
	mixed := []string{"gov_db_flow", "gov_machine_flow"}
	require.NoError(t, trigger.ValidatePolicy(policy, trigger.WithFallbackScenarios(mixed)))

	// 但求值机器场景时不能真拿「需审批」返回（无人消费即等于放行），要收敛成可落地的最严级别
	machineTc := trigger.NewContext(context.Background(), "gov_machine_flow", 1, nil, nil)
	machineDecision := trigger.Evaluate(context.Background(), policy, machineTc)
	if machineDecision.Severity != entity.SeverityForbidden {
		t.Fatalf("the machine side of a mixed policy must be clamped, got %v", machineDecision.Severity)
	}

	// 数据库场景保持管理员配的那个级别，不被连带收敛
	dbTc := trigger.NewContext(context.Background(), "gov_db_flow", 1, nil, nil)
	dbDecision := trigger.Evaluate(context.Background(), policy, dbTc)
	if dbDecision.Severity != entity.SeverityRequired || len(dbDecision.Findings) != 0 {
		t.Fatalf("an approvable scenario must keep the configured fallback untouched, got %v / %+v", dbDecision.Severity, dbDecision.Findings)
	}
}

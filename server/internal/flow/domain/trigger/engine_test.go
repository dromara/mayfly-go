package trigger

import (
	"context"
	"errors"
	"strings"
	"testing"

	"mayfly-go/internal/flow/domain/entity"
)

// 测试场景：字段字典覆盖四种值类型，并包含一个必然取值失败的派生字段
const testBizType = "test_trigger_flow"

func init() {
	RegisterBiz(BizMeta{
		BizType:    testBizType,
		Approvable: true,
		Fields: []TriggerField{
			{Key: "stmtType", TitleKey: "flow.field.stmtType", Group: "statement", Type: TypeEnum, Options: []FieldOption{{Value: "select"}, {Value: "update"}, {Value: "delete"}, {Value: "ddl"}}},
			{Key: "sql", TitleKey: "flow.field.sql", Group: "statement", Type: TypeString},
			{Key: "tableCount", TitleKey: "flow.field.tableCount", Group: "target", Type: TypeNumber},
			{Key: "dangerous", TitleKey: "flow.field.dangerous", Group: "risk", Type: TypeBool},
			{Key: "tables", TitleKey: "flow.field.tables", Group: "target", Type: TypeStringList},
			{
				Key: "broken", TitleKey: "flow.field.broken", Group: "risk", Type: TypeString,
				Resolve: func(*Context) (any, error) { return nil, errors.New("resolver failed") },
			},
		},
		Checks: []CheckDef{
			{
				Key: "test.write-stmt", TitleKey: "flow.check.writeStmt", BizTypes: []string{testBizType},
				Default: entity.SeverityRequired,
				Params: []CheckParam{{
					Key: "types", TitleKey: "flow.checkParam.types", Type: TypeEnum, Required: true,
					Options: []FieldOption{{Value: "select"}, {Value: "update"}, {Value: "delete"}, {Value: "ddl"}},
				}},
				Evaluate: func(ctx context.Context, tc *Context, params map[string]any) (bool, error) {
					stmtType, _ := tc.RawValue("stmtType")
					return OneOf(stmtType, ParamStrings(params, "types")), nil
				},
			},
			{
				Key: "test.heavy-scan", TitleKey: "flow.check.heavyScan", BizTypes: []string{testBizType},
				Default: entity.SeverityWarning,
				Evaluate: func(ctx context.Context, tc *Context, params map[string]any) (bool, error) {
					sql, _ := tc.RawValue("sql")
					return strings.Contains(sql, "SELECT *"), nil
				},
			},
			{
				Key: "test.broken-check", TitleKey: "flow.check.broken", BizTypes: []string{testBizType},
				Default: entity.SeverityRequired,
				Evaluate: func(ctx context.Context, tc *Context, params map[string]any) (bool, error) {
					return false, errors.New("check failed")
				},
			},
		},
	})
}

func newTestContext(raw map[string]string) *Context {
	return NewContext(context.Background(), testBizType, 1, raw, nil)
}

// newTestContextWith 列表型字段的取值是真正的切片，只能经 Attributes 传入（Raw 恒为字符串）
func newTestContextWith(raw map[string]string, attributes map[string]any) *Context {
	return newTestContext(raw).WithAttributes(attributes)
}

// listCtx 构造只带列表型取值的上下文
func listCtx(attributes map[string]any) *Context {
	return newTestContextWith(map[string]string{}, attributes)
}

func condition(field string, op entity.OpName, value any) *entity.RuleNode {
	return &entity.RuleNode{Kind: entity.NodeKindCondition, Field: field, Op: op, Value: value}
}

func group(logic entity.Logic, items ...*entity.RuleNode) *entity.RuleNode {
	return &entity.RuleNode{Kind: entity.NodeKindGroup, Logic: logic, Items: items}
}

func TestEvalOperators(t *testing.T) {
	base := map[string]string{"stmtType": "update", "sql": "UPDATE t SET a = 1 WHERE id = 2", "tableCount": "3", "dangerous": "true", "tables": ""}

	cases := []struct {
		name    string
		ctx     *Context
		node    *entity.RuleNode
		want    bool
		wantErr bool
	}{
		{"enum eq 命中", newTestContext(base), condition("stmtType", OpEq, "update"), true, false},
		{"enum eq 忽略大小写", newTestContext(base), condition("stmtType", OpEq, "UPDATE"), true, false},
		{"enum ne", newTestContext(base), condition("stmtType", OpNe, "delete"), true, false},
		{"enum in", newTestContext(base), condition("stmtType", OpIn, []any{"insert", "update"}), true, false},
		{"enum in 未命中", newTestContext(base), condition("stmtType", OpIn, []any{"delete"}), false, false},
		{"enum notIn", newTestContext(base), condition("stmtType", OpNotIn, []any{"delete", "ddl"}), true, false},
		{"enum in 空集合恒假", newTestContext(base), condition("stmtType", OpIn, []any{}), false, false},
		{"string contains 忽略大小写", newTestContext(base), condition("sql", OpContains, "set a"), true, false},
		{"string notContains", newTestContext(base), condition("sql", OpNotContain, "truncate"), true, false},
		{"string startsWith", newTestContext(base), condition("sql", OpStartsWith, "update"), true, false},
		{"string regex", newTestContext(base), condition("sql", OpRegex, `where\s+id\s*=\s*\d+`), true, false},
		{"string regex 非法模式返回错误", newTestContext(base), condition("sql", OpRegex, "([unclosed"), false, true},
		{"number gt", newTestContext(base), condition("tableCount", OpGt, float64(2)), true, false},
		{"number 字符串与数值互通", newTestContext(base), condition("tableCount", OpEq, 3), true, false},
		{"number between", newTestContext(base), condition("tableCount", OpBetween, []any{1, 5}), true, false},
		{"number between 端点缺失返回错误", newTestContext(base), condition("tableCount", OpBetween, []any{1}), false, true},
		{"bool eq true", newTestContext(base), condition("dangerous", OpEq, true), true, false},
		{"bool 只与 bool 相等", newTestContext(base), condition("dangerous", OpEq, "true"), true, false},
		{"list anyOf", listCtx(map[string]any{"tables": []string{"orders", "user"}}), condition("tables", OpAnyOf, []any{"accounts", "orders"}), true, false},
		{"list anyOf 未命中", listCtx(map[string]any{"tables": []string{"orders", "user"}}), condition("tables", OpAnyOf, []any{"accounts"}), false, false},
		{"list allOf", listCtx(map[string]any{"tables": []string{"orders", "user"}}), condition("tables", OpAllOf, []any{"orders", "user"}), true, false},
		{"list noneOf", listCtx(map[string]any{"tables": []string{"orders", "user"}}), condition("tables", OpNoneOf, []any{"accounts"}), true, false},
		{"空列表不命中量词", listCtx(map[string]any{"tables": []string{}}), condition("tables", OpAnyOf, []any{"orders"}), false, false},
		{"empty", listCtx(map[string]any{"tables": []string{}}), condition("tables", OpEmpty, nil), true, false},
		{"notEmpty", newTestContext(base), condition("sql", OpNotEmpty, nil), true, false},
		{"数值 0 不算 empty", newTestContext(map[string]string{"tableCount": "0"}), condition("tableCount", OpEmpty, nil), false, false},
		{"未注册字段判假并记入 Unknown", newTestContext(base), condition("ghost", OpEq, "x"), false, false},
		{"派生字段求值失败必须报错而非判假", newTestContext(base), condition("broken", OpEq, "x"), false, true},
		{"字段缺失判假", newTestContext(map[string]string{}), condition("stmtType", OpEq, "update"), false, false},
		{"操作符与字段类型不匹配返回错误", newTestContext(base), condition("stmtType", OpContains, "up"), false, true},
		{"未注册操作符返回错误", newTestContext(base), condition("stmtType", "between2", "up"), false, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := tc.ctx
			got, err := eval(tc.node, ctx)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %v but got %v", tc.want, got)
			}
		})
	}
}

func TestEvalUnknownFieldsAreReported(t *testing.T) {
	// 只缺值（ghost）判假不报错；派生失败（broken）必须冒泡给引擎做 fail-closed
	ctx := newTestContext(map[string]string{"stmtType": "update"})
	matched, err := eval(group(entity.LogicAll, condition("ghost", OpEq, "x")), ctx)
	if err != nil || matched {
		t.Fatalf("a field the caller did not provide must be treated as not matched, got %v %v", matched, err)
	}
	if _, err := eval(group(entity.LogicAll, condition("broken", OpEq, "x")), ctx); err == nil {
		t.Fatalf("a failing field resolver must surface as an error so the engine fails closed")
	}
	joined := strings.Join(ctx.Unknown, ",")
	for _, want := range []string{"ghost"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("unknown fields %q should contain %q", joined, want)
		}
	}
}

func TestEvalGroupLogics(t *testing.T) {
	ctx := newTestContext(map[string]string{"stmtType": "update", "dangerous": "false"})

	if matched, _ := eval(group(entity.LogicAll, condition("stmtType", OpEq, "update"), condition("dangerous", OpEq, false)), ctx); !matched {
		t.Fatalf("all should match when every item matches")
	}
	if matched, _ := eval(group(entity.LogicAny, condition("stmtType", OpEq, "delete"), condition("dangerous", OpEq, false)), ctx); !matched {
		t.Fatalf("any should match when one item matches")
	}
	if matched, _ := eval(group(entity.LogicNone, condition("stmtType", OpEq, "delete"), condition("sql", OpContains, "truncate")), ctx); !matched {
		t.Fatalf("none should match when no item matches")
	}
	// 空分组不表达约束，避免「全不」在空集上恒真导致意外放行
	if matched, _ := eval(&entity.RuleNode{Kind: entity.NodeKindGroup, Logic: entity.LogicNone}, ctx); matched {
		t.Fatalf("an empty group must never match")
	}
	if _, err := eval(&entity.RuleNode{Kind: entity.NodeKindGroup, Logic: "xor", Items: []*entity.RuleNode{{Kind: entity.NodeKindCondition, Field: "stmtType", Op: OpEq, Value: "update"}}}, ctx); err == nil {
		t.Fatalf("an unknown group logic must fail")
	}
}

func TestEvaluateScenarioSemantics(t *testing.T) {
	writeCheck := &entity.CheckConfig{Key: "test.write-stmt", BizType: testBizType, Severity: entity.SeverityRequired, Params: map[string]any{"types": []any{"update", "delete"}}}

	cases := []struct {
		name   string
		policy *entity.TriggerPolicy
		raw    map[string]string
		want   entity.Severity
	}{
		{"未配置规则走兜底级别", &entity.TriggerPolicy{Version: 1, DefaultSeverity: entity.SeverityRequired}, map[string]string{"stmtType": "select"}, entity.SeverityRequired},
		{"配置了规则且未命中则放行", &entity.TriggerPolicy{Version: 1, DefaultSeverity: entity.SeverityRequired, Checks: []*entity.CheckConfig{writeCheck}}, map[string]string{"stmtType": "select"}, entity.SeverityDisabled},
		{"命中检查项", &entity.TriggerPolicy{Version: 1, DefaultSeverity: entity.SeverityDisabled, Checks: []*entity.CheckConfig{writeCheck}}, map[string]string{"stmtType": "update"}, entity.SeverityRequired},
		{"兜底为不放行且未命中", &entity.TriggerPolicy{Version: 1, DefaultSeverity: entity.SeverityDisabled, Checks: []*entity.CheckConfig{writeCheck}}, map[string]string{"stmtType": "select"}, entity.SeverityDisabled},
		{"检查项不匹配场景则不参与判定", &entity.TriggerPolicy{Version: 1, DefaultSeverity: entity.SeverityDisabled, Checks: []*entity.CheckConfig{{Key: writeCheck.Key, BizType: "other_flow", Severity: entity.SeverityRequired, Params: writeCheck.Params}}}, map[string]string{"stmtType": "update"}, entity.SeverityDisabled},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decision := Evaluate(context.Background(), tc.policy, newTestContext(tc.raw))
			if decision.Severity != tc.want {
				t.Fatalf("expected severity %v but got %v (findings=%d)", tc.want, decision.Severity, len(decision.Findings))
			}
		})
	}
}

// 兜底级别与「未命中」必须严格区分：前者是「绑定了流程但一条规则都没配」，
// 后者是「配了规则但这条操作不符合」。两者混用会让只拦写操作的策略把 SELECT 也拦下
func TestDefaultSeverityAppliesOnlyWithoutAnyRule(t *testing.T) {
	rule := &entity.CheckConfig{Key: "test.write-stmt", BizType: testBizType, Severity: entity.SeverityRequired, Params: map[string]any{"types": []any{"update"}}}

	notMatched := Evaluate(context.Background(), &entity.TriggerPolicy{
		Version: 1, DefaultSeverity: entity.SeverityRequired, Checks: []*entity.CheckConfig{rule},
	}, newTestContext(map[string]string{"stmtType": "select"}))
	if notMatched.Severity != entity.SeverityDisabled {
		t.Fatalf("a select must pass through a write-only policy, got %v", notMatched.Severity)
	}
	if notMatched.Matched {
		t.Fatalf("a pass-through decision must not be reported as matched")
	}

	noRule := Evaluate(context.Background(), &entity.TriggerPolicy{
		Version: 1, DefaultSeverity: entity.SeverityRequired,
	}, newTestContext(map[string]string{"stmtType": "select"}))
	if noRule.Severity != entity.SeverityRequired {
		t.Fatalf("a bound flow definition without any rule must fail closed, got %v", noRule.Severity)
	}
}

func TestEvaluateUnlessBeatsEveryTrigger(t *testing.T) {
	policy := &entity.TriggerPolicy{
		Version:         1,
		DefaultSeverity: entity.SeverityDisabled,
		Checks:          []*entity.CheckConfig{{Key: "test.write-stmt", BizType: testBizType, Severity: entity.SeverityForbidden, Params: map[string]any{"types": []any{"update"}}}},
		Customs: []*entity.CustomCondition{{
			BizType:  testBizType,
			Severity: entity.SeverityRequired,
			When:     condition("stmtType", OpEq, "update"),
			Unless:   condition("dangerous", OpEq, false),
		}},
	}

	decision := Evaluate(context.Background(), policy, newTestContext(map[string]string{"stmtType": "update", "dangerous": "false"}))
	if !decision.Exempted {
		t.Fatalf("unless must exempt the operation")
	}
	if decision.Severity != entity.SeverityDisabled {
		t.Fatalf("an exempted operation must not be enforced, got %v", decision.Severity)
	}

	// 豁免条件不成立时，触发结论照常生效
	notExempted := Evaluate(context.Background(), policy, newTestContext(map[string]string{"stmtType": "update", "dangerous": "true"}))
	if notExempted.Exempted || notExempted.Severity != entity.SeverityForbidden {
		t.Fatalf("the strictest severity must win when unless does not match, got %v", notExempted.Severity)
	}
}

func TestEvaluateTakesTheStrictestSeverity(t *testing.T) {
	policy := &entity.TriggerPolicy{
		Version:         1,
		DefaultSeverity: entity.SeverityDisabled,
		Checks: []*entity.CheckConfig{
			{Key: "test.write-stmt", BizType: testBizType, Severity: entity.SeverityWarning, Params: map[string]any{"types": []any{"update"}}},
			{Key: "test.write-stmt", BizType: testBizType, Severity: entity.SeverityForbidden, Params: map[string]any{"types": []any{"update"}}},
		},
	}
	decision := Evaluate(context.Background(), policy, newTestContext(map[string]string{"stmtType": "update"}))
	if decision.Severity != entity.SeverityForbidden {
		t.Fatalf("expected the strictest severity, got %v", decision.Severity)
	}
	if len(decision.Findings) != 2 {
		t.Fatalf("every matched rule must produce a finding, got %d", len(decision.Findings))
	}
}

// 策略不可评估时必须 fail-closed，且留下可定位的结论
func TestEvaluateFailsClosed(t *testing.T) {
	cases := []struct {
		name   string
		policy *entity.TriggerPolicy
		raw    map[string]string
	}{
		{"策略为空", nil, map[string]string{}},
		{"模型版本高于引擎", &entity.TriggerPolicy{Version: entity.TriggerPolicyVersion + 1, DefaultSeverity: entity.SeverityDisabled}, map[string]string{}},
		{"检查项未注册", &entity.TriggerPolicy{Version: 1, DefaultSeverity: entity.SeverityDisabled, Checks: []*entity.CheckConfig{{Key: "ghost.check", BizType: testBizType, Severity: entity.SeverityRequired}}}, map[string]string{}},
		{"检查项求值失败", &entity.TriggerPolicy{Version: 1, DefaultSeverity: entity.SeverityDisabled, Checks: []*entity.CheckConfig{{Key: "test.broken-check", BizType: testBizType, Severity: entity.SeverityDisabled}}}, map[string]string{}},
		{"条件树含未注册操作符", &entity.TriggerPolicy{Version: 1, DefaultSeverity: entity.SeverityDisabled, Customs: []*entity.CustomCondition{{BizType: testBizType, Severity: entity.SeverityRequired, When: condition("stmtType", "ghostOp", "update")}}}, map[string]string{"stmtType": "update"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decision := Evaluate(context.Background(), tc.policy, newTestContext(tc.raw))
			if decision.Severity != entity.SeverityRequired {
				t.Fatalf("expected fail-closed required severity, got %v", decision.Severity)
			}
			if len(decision.Findings) == 0 || decision.Findings[0].Source != SourceEngine {
				t.Fatalf("a fail-closed decision must carry an engine finding, got %+v", decision.Findings)
			}
		})
	}
}

// fail-closed 的落点要分场景：可审批场景兜到「需审批」（交人工，不硬拒），
// 没有审批回调的场景兜到「需审批」等于放行，必须退到最严的可用级别
func TestFailClosedSeverityFollowsScenarioCapability(t *testing.T) {
	if got := failClosedSeverity(testBizType); got != entity.SeverityRequired {
		t.Fatalf("an approvable scenario must fail closed to required, got %v", got)
	}

	const noChannel = "test_fail_closed_no_channel"
	RegisterBiz(BizMeta{BizType: noChannel, Fields: []TriggerField{{Key: "k", TitleKey: "f", Group: "g", Type: TypeString}}})

	if got := failClosedSeverity(noChannel); got != entity.SeverityForbidden {
		t.Fatalf("a scenario without an approval channel must fail closed to forbidden, got %v", got)
	}
}

// 字段结果在同一次求值内只计算一次，避免重复解析或重复查库
func TestFieldValueIsCachedPerEvaluation(t *testing.T) {
	calls := 0
	RegisterBiz(BizMeta{
		BizType: "test_counting_flow",
		Fields: []TriggerField{{
			Key: "counted", TitleKey: "flow.field.counted", Group: "risk", Type: TypeNumber,
			Resolve: func(*Context) (any, error) { calls++; return 5, nil },
		}},
	})

	ctx := NewContext(context.Background(), "test_counting_flow", 1, nil, nil)
	node := group(entity.LogicAll, condition("counted", OpGt, 1), condition("counted", OpLt, 10))
	if matched, err := eval(node, ctx); err != nil || !matched {
		t.Fatalf("expected the cached value to satisfy both conditions, got %v %v", matched, err)
	}
	if calls != 1 {
		t.Fatalf("the resolver must run once per evaluation, ran %d times", calls)
	}
}

func TestCountNodes(t *testing.T) {
	tree := group(entity.LogicAll, condition("stmtType", OpEq, "update"), group(entity.LogicAny, condition("sql", OpContains, "x"), condition("tableCount", OpGt, 1)))
	count, depth := countNodes(tree)
	if count != 5 {
		t.Fatalf("expected 5 nodes, got %d", count)
	}
	if depth != 3 {
		t.Fatalf("expected a depth of 3, got %d", depth)
	}
}

// 自定义条件命中也必须带上服务端原因：它没有内置检查项那样的固定规则名，
// 若不挂原因，拦截提示会渲染成空括号，被拦的人完全看不出是哪条规则起作用
func TestCustomWhenFindingCarriesReason(t *testing.T) {
	cases := []struct {
		name   string
		policy *entity.TriggerPolicy
		raw    map[string]string
		want   string
	}{
		{
			name: "只命中自定义条件",
			policy: &entity.TriggerPolicy{
				Version: entity.TriggerPolicyVersion, DefaultSeverity: entity.SeverityDisabled,
				Customs: []*entity.CustomCondition{{
					BizType: testBizType, Severity: entity.SeverityRequired,
					When: condition("sql", OpContains, "orders"),
				}},
			},
			raw:  map[string]string{"sql": "select * from orders", "stmtType": "select"},
			want: SourceCustomWhen,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			decision := Evaluate(context.Background(), tc.policy, newTestContext(tc.raw))
			if decision.Severity != entity.SeverityRequired {
				t.Fatalf("the custom condition must drive the decision, got %v", decision.Severity)
			}
			var matched *Finding
			for _, finding := range decision.Findings {
				if finding.Source == tc.want {
					matched = finding
				}
			}
			if matched == nil {
				t.Fatalf("expected a finding from %q, got %+v", tc.want, decision.Findings)
			}
			if matched.Summary == 0 {
				t.Fatalf("a matched custom condition must carry a server-side reason, got summary 0")
			}
		})
	}
}

// TestEvaluateToleratesNilRuleEntries 策略里混进 null 规则时求值不能 panic。
//
// 保存期 validate 会拒这种数据，但历史脏数据、手工改库、以后改结构时的半成品都存在；
// 求值发生在业务执行路径上，panic 会让这条 SQL / 这条命令以 500 失败，
// 而界面完全看不出是策略数据的问题
func TestEvaluateToleratesNilRuleEntries(t *testing.T) {
	writeStmt := &entity.CheckConfig{Key: "test.write-stmt", BizType: testBizType, Severity: entity.SeverityRequired, Params: map[string]any{"types": []any{"update"}}}
	ctx := newTestContext(map[string]string{"stmtType": "update"})

	mixed := &entity.TriggerPolicy{
		Version: entity.TriggerPolicyVersion, DefaultSeverity: entity.SeverityDisabled,
		Checks:  []*entity.CheckConfig{nil, writeStmt, nil},
		Customs: []*entity.CustomCondition{nil},
	}
	if decision := Evaluate(context.Background(), mixed, ctx); decision.Severity != entity.SeverityRequired {
		t.Errorf("null 元素旁边的有效规则仍要生效, got %v", decision.Severity)
	}

	// 只剩 null 时等价于「该场景没配规则」，按兜底处置：与 HasRuleFor 的判空口径必须一致，
	// 否则同一份数据在两个函数里得出相反结论
	onlyNil := &entity.TriggerPolicy{
		Version: entity.TriggerPolicyVersion, DefaultSeverity: entity.SeverityForbidden,
		Checks: []*entity.CheckConfig{nil}, Customs: []*entity.CustomCondition{nil},
	}
	if decision := Evaluate(context.Background(), onlyNil, newTestContext(map[string]string{"stmtType": "update"})); decision.Severity != entity.SeverityForbidden {
		t.Errorf("全是 null 时应走兜底级别, got %v", decision.Severity)
	}
}

// TestEvaluateDoesNotMutatePolicy 求值对入参只读。
//
// 版本归一曾经直接写回 policy.Version，一旦将来给流程定义加缓存（多个 goroutine 共享同一份
// 策略对象），这就是数据竞争；顺带也保证试算与真实执行看到的是同一份输入
func TestEvaluateDoesNotMutatePolicy(t *testing.T) {
	policy := &entity.TriggerPolicy{DefaultSeverity: entity.SeverityRequired, Checks: []*entity.CheckConfig{{Key: "test.write-stmt", BizType: testBizType, Severity: entity.SeverityRequired}}}
	Evaluate(context.Background(), policy, newTestContext(map[string]string{"stmtType": "update"}))

	if policy.Version != 0 {
		t.Errorf("求值不应改写入参的版本号, got %d", policy.Version)
	}
}

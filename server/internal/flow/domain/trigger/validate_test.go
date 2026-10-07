package trigger

import (
	"context"
	"strings"
	"testing"

	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/imsg"
)

func validPolicy() *entity.TriggerPolicy {
	return &entity.TriggerPolicy{
		Version:         entity.TriggerPolicyVersion,
		DefaultSeverity: entity.SeverityDisabled,
		Checks: []*entity.CheckConfig{{
			Key: "test.write-stmt", BizType: testBizType, Severity: entity.SeverityRequired,
			Params: map[string]any{"types": []any{"update", "delete"}},
		}},
		Customs: []*entity.CustomCondition{{
			BizType:  testBizType,
			Severity: entity.SeverityRequired,
			When:     group(entity.LogicAll, condition("stmtType", OpIn, []any{"update"}), condition("tableCount", OpGte, 2)),
			Unless:   condition("dangerous", OpEq, false),
		}},
	}
}

func TestValidatePolicyAcceptsWellFormedPolicy(t *testing.T) {
	if err := ValidatePolicy(validPolicy()); err != nil {
		t.Fatalf("a well formed policy must pass validation, got %v", err)
	}
	if err := ValidatePolicy(nil); err != nil {
		t.Fatalf("an absent policy is valid (the caller decides the fallback), got %v", err)
	}
}

func TestValidatePolicyRejects(t *testing.T) {
	cases := []struct {
		name       string
		policy     *entity.TriggerPolicy
		wantSource string
	}{
		{"模型版本高于引擎", withVersion(validPolicy(), entity.TriggerPolicyVersion+1), "version"},
		{"兜底级别非法", withDefaultSeverity(validPolicy(), entity.Severity(9)), "defaultSeverity"},
		{"检查项未注册", withChecks(validPolicy(), &entity.CheckConfig{Key: "ghost", BizType: testBizType, Severity: entity.SeverityRequired}), "checks[ghost@"},
		{"检查项未声明场景", withChecks(validPolicy(), &entity.CheckConfig{Key: "test.write-stmt", Severity: entity.SeverityRequired, Params: map[string]any{"types": []any{"update"}}}), "checks[test.write-stmt@]"},
		{"检查项场景未注册", withChecks(validPolicy(), &entity.CheckConfig{Key: "test.write-stmt", BizType: "ghost_flow", Severity: entity.SeverityRequired, Params: map[string]any{"types": []any{"update"}}}), "checks[test.write-stmt@ghost_flow]"},
		{"检查项不适用该场景", withChecks(validPolicy(), &entity.CheckConfig{Key: "test.write-stmt", BizType: "test_counting_flow", Severity: entity.SeverityRequired, Params: map[string]any{"types": []any{"update"}}}), "checks[test.write-stmt@test_counting_flow]"},
		{"检查项重复配置", withChecks(validPolicy(), &entity.CheckConfig{Key: "test.write-stmt", BizType: testBizType, Severity: entity.SeverityRequired, Params: map[string]any{"types": []any{"update"}}}), "checks[test.write-stmt@"},
		{"规则级别为不处置", withChecks(validPolicy(), &entity.CheckConfig{Key: "test.write-stmt", BizType: testBizType, Severity: entity.SeverityDisabled, Params: map[string]any{"types": []any{"update"}}}), ".severity"},
		{"必填参数缺失", onlyChecks(validPolicy(), &entity.CheckConfig{Key: "test.write-stmt", BizType: testBizType, Severity: entity.SeverityRequired}), "params.types"},
		{"未声明的参数", onlyChecks(validPolicy(), &entity.CheckConfig{Key: "test.write-stmt", BizType: testBizType, Severity: entity.SeverityRequired, Params: map[string]any{"types": []any{"update"}, "ghost": 1}}), "params.ghost"},
		{"枚举参数取值越界", onlyChecks(validPolicy(), &entity.CheckConfig{Key: "test.write-stmt", BizType: testBizType, Severity: entity.SeverityRequired, Params: map[string]any{"types": []any{"not_a_stmt"}}}), "params.types"},
		{"条件引用未注册字段", withWhen(validPolicy(), condition("ghost", OpEq, "x")), ".when"},
		{"条件操作符与字段类型不匹配", withWhen(validPolicy(), condition("stmtType", OpContains, "up")), ".when"},
		{"条件字段使用了不允许的操作符", withWhen(validPolicy(), condition("broken", OpBetween, []any{1, 2})), ".when"},
		{"枚举条件取值越界", withWhen(validPolicy(), condition("stmtType", OpIn, []any{"update", "ghost"})), ".when.value"},
		{"布尔条件取值非布尔", withWhen(validPolicy(), condition("dangerous", OpEq, 1)), ".when.value"},
		{"数值条件取值非数值", withWhen(validPolicy(), condition("tableCount", OpGt, "many")), ".when.value"},
		{"正则不可编译", withWhen(validPolicy(), condition("sql", OpRegex, "([unclosed")), ".when.value"},
		{"集合操作符缺少期望值", withWhen(validPolicy(), condition("stmtType", OpIn, []any{})), ".when.value"},
		{"区间操作符缺少端点", withWhen(validPolicy(), condition("tableCount", OpBetween, []any{1})), ".when.value"},
		{"分组为空", withWhen(validPolicy(), group(entity.LogicAll)), ".when"},
		{"分组逻辑非法", &entity.TriggerPolicy{Version: 1, DefaultSeverity: entity.SeverityDisabled, Customs: []*entity.CustomCondition{{BizType: testBizType, Severity: entity.SeverityRequired, When: &entity.RuleNode{Kind: entity.NodeKindGroup, Logic: "xor", Items: []*entity.RuleNode{condition("stmtType", OpEq, "update")}}}}}, ".when"},
		{"自定义条件场景未注册", withCustoms(validPolicy(), &entity.CustomCondition{BizType: "ghost_flow", Severity: entity.SeverityRequired, When: condition("stmtType", OpEq, "update")}), "customs[ghost_flow]"},
		{"自定义条件场景重复", withCustoms(validPolicy(), &entity.CustomCondition{BizType: testBizType, Severity: entity.SeverityRequired, When: condition("stmtType", OpEq, "update")}), "customs[" + testBizType + "]"},
		{"when 与 unless 均为空", withCustoms(validPolicy(), &entity.CustomCondition{BizType: "test_counting_flow", Severity: entity.SeverityRequired}), "customs[test_counting_flow]"},
		{"嵌套层数超限", withWhen(validPolicy(), deepGroup(MaxNodeDepth+2, condition("stmtType", OpEq, "update"))), "customs["},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePolicy(tc.policy)
			if err == nil {
				t.Fatalf("expected a validation error")
			}
			policyErr, ok := err.(*Error)
			if !ok {
				t.Fatalf("expected a *trigger.Error, got %T", err)
			}
			if !strings.Contains(policyErr.Source, tc.wantSource) {
				t.Fatalf("the error source %q should locate %q", policyErr.Source, tc.wantSource)
			}
			if policyErr.MsgId == 0 {
				t.Fatalf("the error must carry a localized reason id so the UI can explain which rule is wrong")
			}
		})
	}
}

// 没有审批回调的场景不能配「需审批」：工单批完没人把操作执行回去，等于配了一个无法落地的级别
func TestSeverityAvailabilityFollowsApprovalChannel(t *testing.T) {
	const silentBizType = "test_no_channel_flow"
	RegisterBiz(BizMeta{
		BizType: silentBizType,
		Fields:  []TriggerField{{Key: "cmd", TitleKey: "flow.field.cmd", Group: "risk", Type: TypeString}},
		Checks: []CheckDef{{
			Key: "silent.cmd", TitleKey: "flow.check.x", BizTypes: []string{silentBizType}, Default: entity.SeverityForbidden,
			Evaluate: func(ctx context.Context, tc *Context, params map[string]any) (bool, error) { return false, nil },
		}},
	})

	available := AvailableSeverities(silentBizType)
	if len(available) != 2 {
		t.Fatalf("a scenario without an approval handler must offer only warning and forbidden, got %v", available)
	}
	for _, severity := range available {
		if severity == entity.SeverityRequired {
			t.Fatalf("required must not be offered without an approval handler")
		}
	}

	policy := &entity.TriggerPolicy{
		Version: entity.TriggerPolicyVersion, DefaultSeverity: entity.SeverityDisabled,
		Checks: []*entity.CheckConfig{{Key: "silent.cmd", BizType: silentBizType, Severity: entity.SeverityRequired}},
	}
	err := ValidatePolicy(policy)
	if err == nil {
		t.Fatalf("configuring required on a scenario without an approval handler must be rejected")
	}
	if policyErr := AsError(err); policyErr == nil || policyErr.MsgId != imsg.ReasonSeverityNoChannel {
		t.Fatalf("the rejection must be classified as a missing approval channel, got %v", err)
	}

	// 同一个检查项配成「禁止执行」仍然合法：能力缺失只收窄级别选项，不禁止该场景参与策略
	policy.Checks[0].Severity = entity.SeverityForbidden
	if err := ValidatePolicy(policy); err != nil {
		t.Fatalf("forbidden must stay configurable, got %v", err)
	}
}

// 注册表写错必须在启动期就炸出来，而不是等到运行时静默 fail-closed
func TestRegistryContract(t *testing.T) {
	for _, meta := range BizMetas() {
		if meta.BizType == "" {
			t.Fatalf("a registered biz type must have a name")
		}
		seenFields := map[string]bool{}
		for _, field := range meta.Fields {
			if field.Key == "" || field.TitleKey == "" || field.Group == "" {
				t.Fatalf("field %+v of %s must declare key, title key and group", field, meta.BizType)
			}
			if seenFields[field.Key] {
				t.Fatalf("field %q of %s is registered twice", field.Key, meta.BizType)
			}
			seenFields[field.Key] = true

			if _, ok := TypeOf(field.Type); !ok {
				t.Fatalf("field %q of %s uses the unregistered type %q", field.Key, meta.BizType, field.Type)
			}
			if field.Type == TypeEnum && len(field.Options) == 0 {
				t.Fatalf("the enum field %q of %s must declare its options", field.Key, meta.BizType)
			}
			for _, op := range field.AvailableOps() {
				def, ok := OpOf(op)
				if !ok {
					t.Fatalf("field %q of %s allows the unregistered operator %q", field.Key, meta.BizType, op)
				}
				if !def.appliesTo(field.Type) {
					t.Fatalf("field %q of %s allows %q which does not apply to its type %q", field.Key, meta.BizType, op, field.Type)
				}
			}
		}

		for _, check := range ChecksOf(meta.BizType) {
			if check.TitleKey == "" {
				t.Fatalf("the check %q must declare a title key", check.Key)
			}
			if check.Evaluate == nil {
				t.Fatalf("the check %q must provide an evaluation function", check.Key)
			}
			if validErr := entity.SeverityEnum.Valid(check.Default); validErr != nil {
				t.Fatalf("the check %q declares an invalid default severity", check.Key)
			}
			seenParams := map[string]bool{}
			for _, param := range check.Params {
				if param.Key == "" || param.TitleKey == "" {
					t.Fatalf("the check %q has a parameter without a key and a title key", check.Key)
				}
				if seenParams[param.Key] {
					t.Fatalf("the check %q declares the parameter %q twice", check.Key, param.Key)
				}
				seenParams[param.Key] = true
				if _, ok := TypeOf(param.Type); !ok {
					t.Fatalf("the check %q parameter %q uses the unregistered type %q", check.Key, param.Key, param.Type)
				}
				if param.Type == TypeEnum && len(param.Options) == 0 {
					t.Fatalf("the check %q parameter %q must declare its options", check.Key, param.Key)
				}
				if param.Min != nil && param.Max != nil && *param.Min > *param.Max {
					t.Fatalf("the check %q parameter %q has an inverted range", check.Key, param.Key)
				}
			}
		}
	}
}

func withVersion(policy *entity.TriggerPolicy, version int) *entity.TriggerPolicy {
	clone := *policy
	clone.Version = version
	return &clone
}

func withDefaultSeverity(policy *entity.TriggerPolicy, severity entity.Severity) *entity.TriggerPolicy {
	clone := *policy
	clone.DefaultSeverity = severity
	return &clone
}

// onlyChecks 用给定配置整体替换检查项列表，用于单独验证某一条检查项的校验分支
func onlyChecks(policy *entity.TriggerPolicy, config *entity.CheckConfig) *entity.TriggerPolicy {
	clone := *policy
	clone.Checks = []*entity.CheckConfig{config}
	clone.Customs = nil
	return &clone
}

func withChecks(policy *entity.TriggerPolicy, extra *entity.CheckConfig) *entity.TriggerPolicy {
	clone := *policy
	clone.Checks = append(append([]*entity.CheckConfig{}, policy.Checks...), extra)
	return &clone
}

func withCustoms(policy *entity.TriggerPolicy, extra *entity.CustomCondition) *entity.TriggerPolicy {
	clone := *policy
	clone.Customs = append(append([]*entity.CustomCondition{}, policy.Customs...), extra)
	return &clone
}

func withWhen(policy *entity.TriggerPolicy, when *entity.RuleNode) *entity.TriggerPolicy {
	clone := *policy
	custom := *policy.Customs[0]
	custom.When = when
	clone.Customs = []*entity.CustomCondition{&custom}
	return &clone
}

// deepGroup 构造 n 层嵌套分组，用于触发深度上限
func deepGroup(depth int, leaf *entity.RuleNode) *entity.RuleNode {
	node := leaf
	for range depth {
		node = group(entity.LogicAll, node)
	}
	return node
}

// TestParamIssueSourceAndNumbers 校验定位串与数字写法必须是管理员能对照的形式。
//
// 定位串拼成 `...params.params.maxKb` 时，管理员按提示去找控件会找错一层；
// 上限渲染成 `1.048576e+06` 则没人能把它和输入框里的数字对上
func TestParamIssueSourceAndNumbers(t *testing.T) {
	const sizeBizType = "test_size_param_flow"
	min, max := 1.0, 1048576.0
	policy := &entity.TriggerPolicy{
		Version: entity.TriggerPolicyVersion, DefaultSeverity: entity.SeverityDisabled,
		Checks: []*entity.CheckConfig{{
			Key: "size.check", BizType: sizeBizType, Severity: entity.SeverityWarning, Params: map[string]any{"maxKb": 99999999.0},
		}},
	}
	RegisterBiz(BizMeta{
		BizType: sizeBizType,
		Fields:  []TriggerField{{Key: "sql", TitleKey: "flow.field.sql", Group: "risk", Type: TypeString}},
		Checks: []CheckDef{{
			Key: "size.check", TitleKey: "flow.check.x", BizTypes: []string{sizeBizType}, Default: entity.SeverityWarning,
			Params:   []CheckParam{{Key: "maxKb", TitleKey: "flow.checkParam.maxKb", Type: TypeNumber, Min: &min, Max: &max, Required: true}},
			Evaluate: func(ctx context.Context, tc *Context, params map[string]any) (bool, error) { return false, nil },
		}},
	})

	err := ValidatePolicy(policy)
	if err == nil {
		t.Fatalf("a parameter above the declared upper bound must be rejected")
	}
	policyErr := AsError(err)
	if policyErr == nil {
		t.Fatalf("expected a structured validation error, got %v", err)
	}
	if want := "checks[size.check@test_size_param_flow].params.maxKb"; policyErr.Source != want {
		t.Fatalf("the source must locate the parameter exactly once, got %q want %q", policyErr.Source, want)
	}
	reason := policyErr.Reason(context.Background())
	if !strings.Contains(reason, "1048576") {
		t.Fatalf("the upper bound must be written as a plain number, got %q", reason)
	}
	if strings.Contains(reason, "e+") {
		t.Fatalf("scientific notation is unreadable next to an input box, got %q", reason)
	}
}

// TestComparableTypesDeclareNumeric 可比较大小小的类型必须自己标成数值型。
//
// min/max 边界与期望值形态都按 Numeric 判定：新注册一个带 gt/between 的类型却忘了标，
// 校验会静默放行越界值、前端也不再拦，两侧同时失效且没有任何报错
func TestComparableTypesDeclareNumeric(t *testing.T) {
	// 不叫 comparable：那是 Go 的预声明约束名，遮蔽它会让后来人以为是别的东西
	orderOps := map[entity.OpName]bool{OpGt: true, OpGte: true, OpLt: true, OpLte: true, OpBetween: true}

	for name, def := range fieldTypeRegistry {
		for _, op := range def.DefaultOps {
			if !orderOps[op] {
				continue
			}
			if !def.Numeric {
				t.Errorf("类型 %s 支持比较操作符 %s，却没标记 Numeric", name, op)
			}
		}
	}
	if len(fieldTypeRegistry) == 0 {
		t.Fatal("类型注册表为空，本用例形同虚设")
	}
}

// TestNumericParamKindsAllGetBounds 三类数值型参数（含时间戳、时长）都要真的套上 min/max。
//
// 判定从「三行 case 类型名」改成读注册表之后，必须有行为用例兜着：谓词被收窄成只认
// number 时，时间戳/时长参数的越界值会静默放行，既不报错也没有任何提示
func TestNumericParamKindsAllGetBounds(t *testing.T) {
	low, high := 1.0, 100.0
	for _, name := range []FieldType{TypeNumber, TypeTime, TypeDuration} {
		param := &CheckParam{Key: "size", TitleKey: "flow.checkParam.maxKb", Type: name, Min: &low, Max: &high}

		err := validateParamValue("checks[db.sql-size-exceeds]", param, 9000)
		if policyErr := AsError(err); err == nil || policyErr == nil || policyErr.MsgId != imsg.ReasonParamTooLarge {
			t.Errorf("%s 类型参数的越界值必须报 paramTooLarge, got %v", name, err)
		}
		err = validateParamValue("checks[db.sql-size-exceeds]", param, "abc")
		if policyErr := AsError(err); err == nil || policyErr == nil || policyErr.MsgId != imsg.ReasonParamNotNumber {
			t.Errorf("%s 类型参数的非数值输入必须报 paramNotNumber, got %v", name, err)
		}
	}

	// 非数值型不参与这套判定，否则字符串参数会被误拒
	if err := validateParamValue("checks[x]", &CheckParam{Key: "note", TitleKey: "flow.checkParam.maxKb", Type: TypeString}, "hello"); err != nil {
		t.Errorf("字符串参数不该按数值校验: %v", err)
	}
}

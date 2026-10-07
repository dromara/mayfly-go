package init

import (
	"slices"
	"testing"

	// 触发各业务场景的注册表填充：注册发生在包级 init，因此只要这些包被链接进二进制就必须可见
	_ "mayfly-go/internal/db/application"
	_ "mayfly-go/internal/machine/application"
	_ "mayfly-go/internal/redis/application"

	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/trigger"
	"mayfly-go/internal/pkg/consts"
)

// 已接入触发策略的场景清单。新增资源类型（Mongo/ES/机器等）在此追加一行，
// 忘记注册就会红，避免「代码写了但没 init 注册」这类只在运行时才暴露的问题
var registeredBizTypes = []string{"db_sql_exec_flow", "machine_run_cmd_flow", "redis_run_cmd_flow"}

// 只服务于流程内部条件（连线跳转、节点完成）的字段字典：有字段、没有检查项与试算
var conditionOnlyBizTypes = []string{"flow_instance"}

func TestEveryScenarioIsRegistered(t *testing.T) {
	metas := trigger.BizMetas()
	bizTypes := make([]string, 0, len(metas))
	for _, meta := range metas {
		bizTypes = append(bizTypes, meta.BizType)
	}
	seen := make(map[string]bool, len(bizTypes))
	for _, bizType := range bizTypes {
		seen[bizType] = true
	}
	for _, want := range append(append([]string{}, registeredBizTypes...), conditionOnlyBizTypes...) {
		if !seen[want] {
			t.Fatalf("the biz type %q is not registered, got %v", want, bizTypes)
		}
	}
}

// 流程内部条件的字段字典不得出现在触发策略的场景清单里：
// 它没有检查项也不能试算，出现在页面上等于给管理员一张配不出规则的卡片
func TestConditionOnlyScenariosStayOutOfThePolicySchema(t *testing.T) {
	for _, bizType := range conditionOnlyBizTypes {
		for _, meta := range trigger.TriggerBizMetas() {
			if meta.BizType == bizType {
				t.Fatalf("the condition-only field dictionary %q leaked into the trigger policy scenarios", bizType)
			}
		}

		meta, ok := trigger.BizMetaOf(bizType)
		if !ok {
			t.Fatalf("the condition-only field dictionary %q is not registered", bizType)
		}
		if !meta.ConditionOnly {
			t.Fatalf("the scenario %q must declare itself condition-only", bizType)
		}
		if len(meta.Checks) > 0 || len(meta.Presets) > 0 || len(meta.SimulateFields) > 0 || meta.Simulate != nil {
			t.Fatalf("the condition-only scenario %q must not register checks, presets or a simulator", bizType)
		}
	}
}

// 每个注册场景都要能被 schema 完整描述：字段有可用操作符且操作符均已登记。
// 这里直接读注册表而不是走应用服务，避免为拿 schema 拉起整个 IOC 容器
func TestRegisteredScenariosExposeUsableSchema(t *testing.T) {
	metas := trigger.TriggerBizMetas()
	if len(metas) == 0 {
		t.Fatalf("no biz type registered, the policy builder would render nothing")
	}
	for _, meta := range metas {
		if len(meta.Fields) == 0 {
			t.Fatalf("the scenario %q registers no field, its custom conditions would be unusable", meta.BizType)
		}
		for _, field := range meta.Fields {
			ops := field.AvailableOps()
			if len(ops) == 0 {
				t.Fatalf("the field %q of %q exposes no operator", field.Key, meta.BizType)
			}
			for _, op := range ops {
				def, registered := trigger.OpOf(op)
				if !registered {
					t.Fatalf("the field %q of %q allows the unregistered operator %q", field.Key, meta.BizType, op)
				}
				if def.ValueKind == "" {
					t.Fatalf("the operator %q of field %q exposes no value kind", op, field.Key)
				}
			}
		}
		if len(trigger.ChecksOf(meta.BizType)) == 0 {
			t.Fatalf("the scenario %q registers no built-in check", meta.BizType)
		}
		if len(meta.SimulateFields) == 0 || meta.Simulate == nil {
			t.Fatalf("the scenario %q must declare its simulation inputs and the parser turning them into field values", meta.BizType)
		}
		// 资源名回填目标必须指向同场景声明的输入项：指到不存在的 key，
		// 前端选完资源后回填会静默丢失，运维又得手填一遍库名
		declared := make(map[string]bool, len(meta.SimulateFields))
		for _, input := range meta.SimulateFields {
			declared[input.Key] = true
		}
		for _, input := range meta.SimulateFields {
			if input.NameKey != "" && !declared[input.NameKey] {
				t.Fatalf("the simulate input %q of %q refills a resource name into the undeclared key %q", input.Key, meta.BizType, input.NameKey)
			}
		}
	}
}

// 流程内部条件用的字段字典同样要能被条件树引用：每个字段都得有可用操作符，
// 否则流程图里的条件编辑器会渲染出一个选了之后填不了的空行
func TestConditionOnlyFieldsAreComparable(t *testing.T) {
	for _, bizType := range conditionOnlyBizTypes {
		meta, ok := trigger.BizMetaOf(bizType)
		if !ok {
			t.Fatalf("the scenario %q is not registered", bizType)
		}
		for _, field := range meta.Fields {
			ops := field.AvailableOps()
			if len(ops) == 0 {
				t.Fatalf("the field %q of %q exposes no operator", field.Key, bizType)
			}
			for _, op := range ops {
				if _, registered := trigger.OpOf(op); !registered {
					t.Fatalf("the field %q of %q allows the unregistered operator %q", field.Key, bizType, op)
				}
			}
			if field.Type == trigger.TypeEnum && len(field.Options) == 0 {
				t.Fatalf("the enum field %q of %q declares no candidate values", field.Key, bizType)
			}
		}
	}
}

// 注册表与校验器必须共用同一份事实：检查项声明的默认级别要能通过规则级别校验
func TestRegisteredCheckDefaultsAreConfigurable(t *testing.T) {
	for _, meta := range trigger.BizMetas() {
		for _, check := range trigger.ChecksOf(meta.BizType) {
			usable := false
			for _, severity := range flowentity.EnforceSeverities {
				if check.Default == severity {
					usable = true
				}
			}
			if !usable {
				t.Fatalf("the check %q defaults to a severity that cannot be configured on a rule", check.Key)
			}
		}
	}
}

// 可配置级别必须由场景声明的审批能力决定：
// 没有审批通道的场景（如机器命令）若开放「需审批」，工单批完没人把操作执行回去
func TestAvailableSeveritiesFollowDeclaredCapability(t *testing.T) {
	approvable := 0
	for _, meta := range trigger.BizMetas() {
		severities := trigger.AvailableSeverities(meta.BizType)
		hasRequired := false
		for _, severity := range severities {
			if severity == flowentity.SeverityRequired {
				hasRequired = true
			}
			if severity == flowentity.SeverityDisabled {
				t.Fatalf("the scenario %q must not offer the no-action severity on a rule", meta.BizType)
			}
		}
		if meta.Approvable != hasRequired {
			t.Fatalf("the scenario %q declares approvable=%v but offers required=%v", meta.BizType, meta.Approvable, hasRequired)
		}
		if meta.Approvable {
			approvable++
		}
	}
	if approvable == 0 {
		t.Fatalf("at least one scenario must be approvable, otherwise the approval path is never reachable")
	}
}

// 每个可触发策略的场景都必须声明它的治理粒度（资源类型路径）。
//
// 「生效资源」的可勾选节点、以及保存期「配了某场景能不能命中」的判定都读这份声明：
// 漏声明或写错的场景即便注册了检查项，管理员也无法把资源纳入它的治理范围，
// 策略看着配好了却永远不会命中（机器命令场景曾就是这样）
//
// 段值合法性交给 consts.KnownResourceTypes 判定，与运行期自检同一份清单，两处不会各说各话
func TestEveryTriggerScenarioDeclaresGovernPaths(t *testing.T) {
	for _, bizType := range registeredBizTypes {
		meta, ok := trigger.BizMetaOf(bizType)
		if !ok {
			t.Fatalf("the biz type %q is not registered", bizType)
		}
		if len(meta.GovernPaths) == 0 {
			t.Fatalf("the scenario %q must declare the resource paths it governs, otherwise it can never be bound to a process definition", bizType)
		}
		for _, path := range meta.GovernPaths {
			if len(path) == 0 {
				t.Fatalf("the scenario %q declares an empty governance path", bizType)
			}
			for _, resourceType := range path {
				if !consts.IsResourceType(resourceType) {
					t.Fatalf("the scenario %q declares the unknown resource type %d in path %v", bizType, resourceType, path)
				}
			}
		}
	}
}

// 治理路径 → 场景 的反查必须能认回每一条声明：
// 它是资源树可勾选节点与保存期命中性校验的共同依据，认不出来就等于放行一份永不生效的配置
func TestGovernedScenariosCoverEveryDeclaredGovernPath(t *testing.T) {
	for _, bizType := range registeredBizTypes {
		meta, ok := trigger.BizMetaOf(bizType)
		if !ok {
			t.Fatalf("the biz type %q is not registered", bizType)
		}
		for _, path := range meta.GovernPaths {
			// 绑定在同一层级或更上层（前缀关系）都必须能反查回该场景
			for i := 1; i <= len(path); i++ {
				scenarios := trigger.GovernedBizTypes([][]int8{path[:i]})
				if !slices.Contains(scenarios, bizType) {
					t.Fatalf("binding the resource path %v must resolve scenario %q, got %v", path[:i], bizType, scenarios)
				}
			}
		}
	}
}

func TestDeclaredGovernPathsCoverEveryScenario(t *testing.T) {
	declared := trigger.DeclaredGovernPaths()
	for _, bizType := range registeredBizTypes {
		meta, ok := trigger.BizMetaOf(bizType)
		if !ok {
			t.Fatalf("the biz type %q is not registered", bizType)
		}
		for _, path := range meta.GovernPaths {
			found := false
			for _, got := range declared {
				if slices.Equal(got, path) {
					found = true
				}
			}
			if !found {
				t.Fatalf("the governance path %v declared by %q is missing from the schema, got %v", path, bizType, declared)
			}
		}
	}
}

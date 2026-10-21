package tools

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// Capability 工具能力等级：Agent Profile 的工具策略过滤与运行期护栏均以此为依据。
//
// 设计目标（开闭原则）：工具在注册处声明一次能力，Profile 的工具过滤与护栏中间件
// 基于能力做**通用**判定 —— 新增工具只需声明能力，过滤/护栏逻辑零改动。
type Capability int

const (
	// CapabilityRead 只读能力：无副作用的查询类工具（只读诊断档可放行）
	CapabilityRead Capability = iota
	// CapabilityWrite 变更能力：有副作用但影响可控的写操作
	CapabilityWrite
	// CapabilityDangerous 高危能力：不可逆或影响面大的操作（须强审批）
	CapabilityDangerous
)

// capabilityNames 能力名（诊断日志与护栏提示用）
var capabilityNames = map[Capability]string{
	CapabilityRead:      "read",
	CapabilityWrite:     "write",
	CapabilityDangerous: "dangerous",
}

func (c Capability) String() string {
	if name, ok := capabilityNames[c]; ok {
		return name
	}
	return "unknown"
}

var (
	capabilityMu sync.RWMutex
	// capabilities 工具名 → 能力等级：内置工具在此集中声明；插件/MCP 工具经
	// DeclareCapability 在注册处追加声明。
	capabilities = map[string]Capability{
		// 只读查询类
		"DbQueryData":     CapabilityRead,
		"DbQueryTables":   CapabilityRead,
		"DbQueryTableDDL": CapabilityRead,
		"ListResources":   CapabilityRead,
		"skill_read":      CapabilityRead,
		"memory_search":   CapabilityRead,
		// 可观测数据查询（阶段 4.2）：与告警引擎同源的只读取值/历史查询，无副作用
		"QueryMetrics":      CapabilityRead,
		"QueryAlertHistory": CapabilityRead,
		// 变更提案登记（只读档也可用）：本身不执行变更，仅登记提案交人工审批，故为只读能力
		"propose_remediation": CapabilityRead,
		// 变更/高危类
		ToolNameExecSql:            CapabilityDangerous,
		ToolNameMachineCommandExec: CapabilityDangerous,
	}
)

// DeclareCapability 声明工具能力（工具注册处调用；同名后声明覆盖先声明）。
//
// 用于插件/MCP 等运行期注册的工具补声明能力；空工具名忽略。
func DeclareCapability(toolName string, c Capability) {
	if toolName == "" {
		return
	}
	capabilityMu.Lock()
	defer capabilityMu.Unlock()
	capabilities[toolName] = c
}

// CapabilityOf 查询工具能力。
//
// 未声明的工具默认按 CapabilityWrite 处理（保守策略）：只读档不会放行未知工具，
// 避免 MCP 等外部工具绕过只读约束；如需放行须在注册处显式声明为 CapabilityRead。
func CapabilityOf(toolName string) Capability {
	capabilityMu.RLock()
	defer capabilityMu.RUnlock()
	if c, ok := capabilities[toolName]; ok {
		return c
	}
	return CapabilityWrite
}

// ==================== 内容级能力（按入参动态定级）====================

// ContentCapabilityJudge 按调用入参动态判定工具能力（argsJSON 是工具收到的原始参数 JSON）。
type ContentCapabilityJudge func(argsJSON string) Capability

var (
	contentCapabilityMu     sync.RWMutex
	contentCapabilityJudges = map[string]ContentCapabilityJudge{}
)

// DeclareContentCapability 声明「按入参动态定能力」的工具（工具注册处 init 调用）。
//
// 存在动机：MachineCommandExec 这类通道既能跑只读探查（top/ps/df）又能跑破坏命令，按工具名
// 一刀切成 Dangerous 会让只读诊断档连 top 都执行不了（只能转提案）。声明判定器后，只读档可下探
// 命令内容：命中只读白名单算 Read 放行，其余仍按 Dangerous 拦。判定器必须与工具体内的审批白名单
// 同源（同一 CmdAnalyzer），否则「护栏放行了、工具体又拦」或反之，两处口径分叉
func DeclareContentCapability(toolName string, judge ContentCapabilityJudge) {
	if toolName == "" || judge == nil {
		return
	}
	contentCapabilityMu.Lock()
	defer contentCapabilityMu.Unlock()
	contentCapabilityJudges[toolName] = judge
}

// CapabilityForArgs 返回工具在本次调用（给定 args）下的有效能力：注册了内容级判定器则用它，
// 否则回退静态 CapabilityOf。护栏据此判定，保证「内容级工具」与「静态工具」走同一条决策路径
func CapabilityForArgs(toolName, argsJSON string) Capability {
	contentCapabilityMu.RLock()
	judge := contentCapabilityJudges[toolName]
	contentCapabilityMu.RUnlock()
	if judge != nil {
		return judge(argsJSON)
	}
	return CapabilityOf(toolName)
}

// HasContentCapability 工具是否注册了内容级判定器（能力随入参变化，静态 CapabilityOf 不足以定级）。
func HasContentCapability(toolName string) bool {
	contentCapabilityMu.RLock()
	defer contentCapabilityMu.RUnlock()
	_, ok := contentCapabilityJudges[toolName]
	return ok
}

// IsDeclaredTool 工具名是否已在能力表登记（内置表 + 各注册处经 DeclareCapability 追加的外部工具）。
//
// 存在动机：本表是「哪些名字是真实可调用工具」的唯一事实来源，受控执行用它把提案的 action
// 解析成本次运行的工具许可：解析不出来就不是一个可调用的工具，应该拒绝启动执行（fail-closed），
// 而不是放开全量工具靠提示词约束。
func IsDeclaredTool(toolName string) bool {
	capabilityMu.RLock()
	defer capabilityMu.RUnlock()
	_, ok := capabilities[toolName]
	return ok
}

// ==================== 工具入参形状（提案参数校验的唯一真源）====================

// ParamSchema 工具入参形状：属性名与必填项。
type ParamSchema struct {
	// Props 全部入参属性名（即模型/提案必须用这些名字提交参数）
	Props []string
	// Required 必填属性名（取自 jsonschema:"required" 标记）
	Required []string
}

var (
	paramSchemaMu sync.RWMutex
	// paramSchemas 工具名 → 入参形状，由工具注册处反射自参数 struct 的 json tag。
	//
	// 为什么不让 agentops 侧手写一份参数名表：工具 struct 已经是参数名的真源，
	// 第2份表写歪一个键（实测过提案存的是 cmd 而工具收 command）只会表现为
	// 「校验永远不报错但批准后执行不了」，且编译与静态检查全绿
	paramSchemas = map[string]ParamSchema{}
)

// DeclareParamSchema 从工具入参 struct 反射出属性名与必填项（在工具注册处调用）。
//
// 与 DeclareCapability 同一模式：新增工具只需在注册处声明一行，校验逻辑零改动。
func DeclareParamSchema(toolName string, paramsSample any) {
	if toolName == "" || paramsSample == nil {
		return
	}
	t := reflect.TypeOf(paramsSample)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}
	schema := ParamSchema{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// json tag 可能带 omitempty 等修饰，取逗号前的属性名
		name := strings.TrimSpace(strings.Split(field.Tag.Get("json"), ",")[0])
		if name == "" || name == "-" {
			continue
		}
		schema.Props = append(schema.Props, name)
		if strings.Contains(field.Tag.Get("jsonschema"), "required") {
			schema.Required = append(schema.Required, name)
		}
	}
	paramSchemaMu.Lock()
	defer paramSchemaMu.Unlock()
	paramSchemas[toolName] = schema
}

// ParamSchemaOf 读取工具入参形状；未声明返回 false。
func ParamSchemaOf(toolName string) (ParamSchema, bool) {
	paramSchemaMu.RLock()
	defer paramSchemaMu.RUnlock()
	schema, ok := paramSchemas[toolName]
	return schema, ok
}

// ValidateToolParams 校验提案锁定的参数是该工具的合法入参。
//
// 不校验会怎样：提案能登记、能被批准，但受控执行那轮的参数绑定永远对不上（批准的是
// 一个工具收不到的键），于是本该能执行却挂在审批上——而且全程无报错。
// 未声明形状的工具（MCP/插件等运行期注册者）跳过：不猜它的参数，
// 「能不能调这个工具」已由能力表与许可清单兜住
func ValidateToolParams(toolName, paramsJSON string) error {
	schema, ok := ParamSchemaOf(toolName)
	if !ok {
		return nil
	}
	trimmed := strings.TrimSpace(paramsJSON)
	if trimmed == "" {
		trimmed = "{}"
	}
	var submitted map[string]any
	if err := json.Unmarshal([]byte(trimmed), &submitted); err != nil || submitted == nil {
		return fmt.Errorf("proposal params for tool %q must be a JSON object", toolName)
	}

	known := make(map[string]bool, len(schema.Props))
	for _, prop := range schema.Props {
		known[prop] = true
	}
	for key := range submitted {
		if !known[key] {
			return fmt.Errorf("proposal params for tool %q contain unknown parameter %q (accepted: %s)",
				toolName, key, strings.Join(schema.Props, ", "))
		}
	}
	// 要求给出该工具的**每一个**入参属性（可选参数显式写空串/0 即可）。
	//
	// 为什么不只校必填项：放行走的是「批准参数与执行实收参数逐项相等」，而执行实收参数由工具
	// 参数 struct marshal 得出、必然含全部属性（可选字段以零值出现）。登记侧一旦允许省略可选属性，
	// 两侧形状就永久不同构，出现「能登记、能被批准、却永远不放行」（真机核实的矛盾）。
	// 要求完整同时让审批人看到全部执行参数；必填缺失已涵盖在本检查内，不再单独判 Required
	var missing []string
	for _, prop := range schema.Props {
		if _, exists := submitted[prop]; !exists {
			missing = append(missing, prop)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("proposal params for tool %q miss parameter %q (every parameter of this tool must be provided, expected: %s)",
			toolName, strings.Join(missing, ", "), strings.Join(schema.Props, ", "))
	}
	return nil
}

// ToolPolicy 工具能力策略：声明某 Agent Profile 允许调用的工具能力集合。
//
// nil 或空 allowed 表示「不限制」（全能力放行）—— 用 nil *ToolPolicy 承载默认档，
// 避免为不限制场景构造对象。
type ToolPolicy struct {
	allowed map[Capability]bool
}

// NewToolPolicy 按允许的能力集构造策略（如只读档 NewToolPolicy(CapabilityRead)）
func NewToolPolicy(allowed ...Capability) *ToolPolicy {
	p := &ToolPolicy{allowed: make(map[Capability]bool, len(allowed))}
	for _, c := range allowed {
		p.allowed[c] = true
	}
	return p
}

// ReadOnlyPolicy 只读策略：仅放行只读工具（诊断档默认策略）
func ReadOnlyPolicy() *ToolPolicy { return NewToolPolicy(CapabilityRead) }

// IsRestricted 策略是否施加了限制（nil/空 = 能力维度不限制）
func (p *ToolPolicy) IsRestricted() bool { return p != nil && len(p.allowed) > 0 }

// Allows 判定工具是否被策略放行；nil/空策略恒放行
func (p *ToolPolicy) Allows(toolName string) bool {
	if !p.IsRestricted() {
		return true
	}
	// 内容级工具（如 MachineCommandExec：既有只读子集又有危险调用）装配期不能按静态 Dangerous 一刀切
	// 移除——装配期没有 args，若移除则只读档模型的工具清单里根本没有它、连只读命令都不会去试（实测：
	// 诊断结论只会「建议你手动跑 top」而非自己跑）。保留进清单，由运行期 capability_guard 按实际入参放行或拦截
	if HasContentCapability(toolName) {
		return true
	}
	return p.allowed[CapabilityOf(toolName)]
}

// AllowsCapability 判定「某个能力级别」是否被策略放行。
//
// 与 Allows(toolName) 的分工：Allows 面向**装配期**按工具名（内容级工具整体保留进清单，让模型看得到）；
// AllowsCapability 面向**运行期**按本次调用已算出的实际能力精确拦——内容级工具（如 MachineCommandExec
// 传 rm）下探出 Dangerous 时，只读档据此挡下，而不是因为「它是内容级工具」就整工具恒放行
func (p *ToolPolicy) AllowsCapability(c Capability) bool {
	if !p.IsRestricted() {
		return true
	}
	return p.allowed[c]
}

// DangerousToolNames 返回能力表里登记为高危（CapabilityDangerous）的工具名，只读取表。
//
// 存在动机：高危工具是「必须有入参形状声明」的那一批——ValidateToolParams 对未声明形状的工具
// 一律跳过（MCP/插件确实拿不到形状），这条宽松规则套在内置高危工具上就等于静默不设防：
// 新增一个高危工具忘了 DeclareParamSchema，坏参数一路登记入库，最后以「批准后执行不了」暴露。
// 只导出只读函数、不暴露表本身，避免消费方往表里塞东西
func DangerousToolNames() []string {
	capabilityMu.RLock()
	defer capabilityMu.RUnlock()
	names := make([]string, 0, 4)
	for name, capability := range capabilities {
		if capability == CapabilityDangerous {
			names = append(names, name)
		}
	}
	return names
}

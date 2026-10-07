package entity

import (
	"fmt"

	"mayfly-go/pkg/enumx"
	"mayfly-go/pkg/utils/collx"
)

// TriggerPolicyVersion 触发策略模型版本，模型结构演进时递增并在引擎侧登记升级函数
const TriggerPolicyVersion = 1

// Severity 触发策略命中后的处置级别，值越大约束越强
type Severity int8

const (
	// SeverityDisabled 不处置：检查项被关闭，或策略未命中且兜底为不处置
	SeverityDisabled Severity = 0
	// SeverityWarning 仅提醒：执行前告知风险，不阻断也不要求工单
	SeverityWarning Severity = 1
	// SeverityRequired 必须审批：需提交工单，审批通过后才允许执行
	SeverityRequired Severity = 2
	// SeverityForbidden 禁止执行：直接拒绝，不提供提单入口
	SeverityForbidden Severity = 3
)

// SeverityEnum 级别描述仅用于日志与错误上下文，界面文案由前端 i18n 承载
var SeverityEnum = enumx.NewEnum[Severity]("处置级别").
	Add(SeverityDisabled, "不处置").
	Add(SeverityWarning, "提醒").
	Add(SeverityRequired, "需审批").
	Add(SeverityForbidden, "禁止")

// EnforceSeverities 可配置到规则上的处置级别。
// 「不处置」由「规则不出现在配置里」表达，避免同一语义有两种写法
var EnforceSeverities = []Severity{SeverityWarning, SeverityRequired, SeverityForbidden}

// StrongerThan 表示 other 相对 self 是否需要更严格的处置
func (s Severity) StrongerThan(other Severity) bool {
	return s > other
}

// TriggerPolicy 流程定义的触发策略：决定一次资源操作是否需要走该审批流程。
//
// 处置语义：
//   - 配置了规则时只有命中才产生处置，未命中即放行
//   - 一条规则都没配置时按 DefaultSeverity 兜底，用于「绑定了流程但尚未配置规则」
//   - 同一场景多条规则命中时取最严格的处置级别
type TriggerPolicy struct {
	Version int `json:"version"`

	// DefaultSeverity 未配置任何规则时的兜底处置
	DefaultSeverity Severity `json:"defaultSeverity"`

	// Checks 内置检查项配置。未出现在此列表的检查项视为关闭
	Checks []*CheckConfig `json:"checks,omitempty"`

	// Customs 自定义条件，按业务场景各一条
	Customs []*CustomCondition `json:"customs,omitempty"`
}

// NewTriggerPolicy 创建带模型版本与兜底级别的触发策略
func NewTriggerPolicy(defaultSeverity Severity) *TriggerPolicy {
	return &TriggerPolicy{Version: TriggerPolicyVersion, DefaultSeverity: defaultSeverity}
}

// CheckConfigured 该场景的这条内置检查项是否被策略显式配置过（出现在 Checks 里即算配置，
// 无论级别是什么）。业务侧据此判断「管理员有没有明确处置过这件事」，
// 未被纳管的硬管控面（如机器命令过滤规则）需要由调用方继续兜住，不能被策略缺席地带走
func (p *TriggerPolicy) CheckConfigured(bizType, checkKey string) bool {
	if p == nil {
		return false
	}
	for _, config := range p.Checks {
		if config != nil && config.BizType == bizType && config.Key == checkKey {
			return true
		}
	}
	return false
}

// CustomOf 返回指定业务场景的自定义条件
func (p *TriggerPolicy) CustomOf(bizType string) *CustomCondition {
	if p == nil {
		return nil
	}
	for _, custom := range p.Customs {
		// 指针切片可能含 null 元素（历史脏数据、手工改库）：这里判空，否则 HasRuleFor 会带着它一起 panic
		if custom != nil && custom.BizType == bizType {
			return custom
		}
	}
	return nil
}

// HasRuleFor 该业务场景是否配置了至少一条规则（检查项或自定义条件）。
//
// 必须按场景判断而不是看整份策略：一份策略可以同时治理数据库与机器命令，
// 只给数据库配了规则时机器侧其实一条都没有。按整份策略判断会让机器的
// 「未配置规则时按兜底处置」永远轮不到，等于把机器绑进了流程却完全没有保护
func (p *TriggerPolicy) HasRuleFor(bizType string) bool {
	if p == nil {
		return false
	}
	for _, config := range p.Checks {
		if config != nil && config.BizType == bizType {
			return true
		}
	}
	return p.CustomOf(bizType) != nil
}

// CheckConfig 内置检查项的一次配置：勾选状态由「是否出现在策略中」表达，级别与参数在此覆盖默认
type CheckConfig struct {
	// Key 检查项标识，须存在于检查项注册表
	Key string `json:"key"`

	// BizType 生效的业务场景，必填。同一检查项需要在多个场景生效时按场景各配一条
	BizType string `json:"bizType"`

	// Severity 命中后的处置级别
	Severity Severity `json:"severity"`

	// Params 检查项参数，取值范围由注册表中的参数 schema 约束
	Params map[string]any `json:"params,omitempty"`
}

// CustomCondition 某业务场景下的自定义条件：when 命中则按 Severity 处置，unless 命中则豁免
type CustomCondition struct {
	// BizType 生效的业务场景，必填
	BizType string `json:"bizType"`

	// Severity when 命中后的处置级别
	Severity Severity `json:"severity"`

	// When 触发条件
	When *RuleNode `json:"when,omitempty"`

	// Unless 豁免条件，命中后本次操作直接放行，用于给特定人、特定时段或特定对象开口子
	Unless *RuleNode `json:"unless,omitempty"`
}

// NodeKind 条件节点形态
type NodeKind string

const (
	// NodeKindGroup 分组节点，按 Logic 组合 Items
	NodeKindGroup NodeKind = "group"
	// NodeKindCondition 条件叶子，对单个字段施加操作符与期望值比较
	NodeKindCondition NodeKind = "condition"
	// NodeKindSegment 引用可复用条件组（见 RuleSegment），只存引用不存内容：
	// 同一条「DBA 成员」「工作时段」之类的判断被多处规则使用时，改一处即全部生效
	NodeKindSegment NodeKind = "segment"
)

// Logic 分组节点的组合方式
type Logic string

const (
	// LogicAll 所有子节点成立
	LogicAll Logic = "all"
	// LogicAny 任一子节点成立
	LogicAny Logic = "any"
	// LogicNone 所有子节点均不成立
	LogicNone Logic = "none"
)

// OpName 操作符名称。具体操作符及其比较逻辑登记在引擎侧注册表，
// 新增操作符或新增业务场景的专属操作符都不需要修改模型
type OpName string

// RuleNode 条件树节点，由 Kind 判别使用哪些字段
type RuleNode struct {
	Kind NodeKind `json:"kind"`

	// Logic、Items 用于 group 节点
	Logic Logic       `json:"logic,omitempty"`
	Items []*RuleNode `json:"items,omitempty"`

	// Field、Op、Value 用于 condition 节点，Field 为字段字典中的 key
	Field string `json:"field,omitempty"`
	Op    OpName `json:"op,omitempty"`
	Value any    `json:"value,omitempty"`

	// Ref 用于 segment 节点，指向可复用条件组的标识
	Ref string `json:"ref,omitempty"`
}

// RuleNodeFromExtra 从节点/连线的 extra 中解析条件树。
//
// 返回 (nil, nil) 表示确实没有配置条件；extra 里有值但解析失败必须返回 error，
// 不能退化成「没有条件」：跳转条件与节点完成条件为空都意味着无条件流转，
// 把一份写坏的条件当成没写，等于让审批分支静默变成直通
func RuleNodeFromExtra(extra collx.M, key string) (*RuleNode, error) {
	if extra == nil || extra[key] == nil {
		return nil, nil
	}
	node := new(RuleNode)
	if err := extra.Unmarshal(key, node); err != nil {
		return nil, fmt.Errorf("the condition stored under %q is unreadable: %w", key, err)
	}
	return node, nil
}

package dto

import (
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/trigger"
)

type SaveProcdef struct {
	Procdef   *entity.Procdef
	MsgTmplId uint64 // 消息模板id
	CodePaths []string
}

type SaveFlowDef struct {
	Id      uint64
	FlowDef *entity.FlowDef // 消息模板id
}

// 启动流程实例请求入参
type StarProc struct {
	BizType string // 业务类型
	BizKey  string // 业务key，若已存在则为修改重新提交流程
	Remark  string // 备注
	BizForm string // 业务表单信息
}

// TriggerRequest 资源操作的前置触发策略校验入参
type TriggerRequest struct {
	// BizType 业务场景，须已在触发引擎注册
	BizType string
	// CodePaths 资源标签路径：既用于就近定位生效的流程定义，也透传给检查项按资源解析配置
	CodePaths []string
	// Raw 业务原始输入（一条 SQL、一条命令等），派生字段由场景注册的字段字典计算
	Raw map[string]string
	// Attributes 调用方已权威计算出的结构化事实（如执行器已得出的语句类型、解析出的目标表），
	// 避免在求值路径上重复解析
	Attributes map[string]any
}

// SimulateTrigger 策略试算入参：对随请求下发的策略草稿求值，不落库
type SimulateTrigger struct {
	BizType string
	Policy  *entity.TriggerPolicy
	// Raw 管理员粘贴的原始输入，经场景的 Simulate 解析为字段取值
	Raw map[string]string
}

// SimulateResult 策略试算结果
type SimulateResult struct {
	Decision *trigger.Decision `json:"decision"`
	// Fields 引擎实际解析到的字段取值，用于回答「为什么是这个结论」
	Fields map[string]any `json:"fields"`
}

// PolicySchema 触发策略 schema，前端策略构建器完全据此渲染，新增场景无需改前端代码
type PolicySchema struct {
	// Scenarios 可作为触发策略配置的业务场景及其字段字典、检查项与可用级别
	Scenarios []*PolicyScenario `json:"scenarios"`
	// ConditionScenarios 只服务于流程内部条件的字段字典（连线跳转、节点完成）。
	//
	// 与 Scenarios 分开列是因为它们配不出触发规则（没有检查项也不能试算），
	// 但流程设计器的条件编辑器需要同一份 schema 形态来选择字段与可复用条件组
	ConditionScenarios []*PolicyScenario `json:"conditionScenarios"`
	// GovernPaths 所有已注册场景的治理路径并集，供「流程定义 → 生效资源」的标签树筛选可勾选节点。
	//
	// 由注册表下发而不是前端写死：否则新场景（机器、mongo、es……）在后端注册完了，
	// 管理员却在资源树里根本选不到这类资源，策略配得再全也永远不会命中。
	// 下发的是整条路径而不只是末级类型：数据库按库治理，可勾选节点在「实例/凭证/库」第三层，
	// 前端若只拿到类型就得自己写「实例 → 换成库那条路径」的分支，新资源必须回来加分支
	GovernPaths [][]int8 `json:"governPaths"`
}

// PolicyScenario 单个业务场景可用的策略元素
type PolicyScenario struct {
	BizType string `json:"bizType"`
	// Severities 该场景可配置到规则上的处置级别，由是否存在审批业务回调推导
	Severities []entity.Severity `json:"severities"`
	// FailClosedSeverity 该场景的兜底落点：兜底级别在这里落不了地时，实际生效的就是这个级别
	// （编辑器据此提示，不让管理员在运行期才发现「配了需审批却变成整台机器禁止执行」）
	FailClosedSeverity entity.Severity `json:"failClosedSeverity"`
	Fields             []*PolicyField  `json:"fields"`
	Checks             []*PolicyCheck  `json:"checks"`
	// Presets 该场景的预置规则包
	Presets []trigger.PolicyPreset `json:"presets"`
	// SimulateFields 试算面板的输入项（含控件标识，前端不再按类型猜控件）
	SimulateFields []*PolicySimulateInput `json:"simulateFields"`
	// Segments 该场景可引用的可复用条件组，条件树里以 segment 节点引用它们的 Ref
	Segments []*PolicySegment `json:"segments"`
	// ConditionPresets 条件树预置写法（如或签 / 会签），流程设计器据此给出「一键套用」
	ConditionPresets []trigger.ConditionPreset `json:"conditionPresets"`
}

// PolicySimulateInput 试算面板的一个输入项
type PolicySimulateInput struct {
	Key      string            `json:"key"`
	TitleKey string            `json:"titleKey"`
	Type     trigger.FieldType `json:"type"`
	// EditorKey 值编辑器标识：条件字段从类型注册表推导，试算输入可由场景直接声明覆盖
	// ——资源引用类输入靠它换成资源树选择，而不是让管理员手填内部 id
	EditorKey string `json:"editorKey"`
	// NameKey 资源选择控件选中后回填资源显示名的输入项 key
	NameKey        string `json:"nameKey,omitempty"`
	PlaceholderKey string `json:"placeholderKey,omitempty"`
	Required       bool   `json:"required"`
	// Multiline 是否需要多行控件，由声明该输入的场景给出
	Multiline bool `json:"multiline"`
}

// PolicySegment 可复用的条件组（下发引用候选所需的最小信息）
type PolicySegment struct {
	Ref    string `json:"ref"`
	Name   string `json:"name"`
	Remark string `json:"remark,omitempty"`
}

// PolicyField 条件可引用的字段
type PolicyField struct {
	Key      string            `json:"key"`
	TitleKey string            `json:"titleKey"`
	Group    string            `json:"group"`
	Type     trigger.FieldType `json:"type"`
	// EditorKey 前端值编辑器标识，由后端按注册的字段类型下发（前端不再按类型名猜控件）
	EditorKey string `json:"editorKey"`
	// Numeric 期望值是否按数值比较，取自字段类型注册表：
	// 少了它前端只能用 type === 'number' 自判，而 time / duration 在后端同样按数值比较，
	// 注册一个新数值语义字段就会变成「界面全绿、保存被后端拒」
	Numeric bool              `json:"numeric"`
	Options []string          `json:"options,omitempty"`
	Ops     []*PolicyOperator `json:"ops"`
}

// PolicyOperator 字段可用的操作符及其期望值形态
type PolicyOperator struct {
	Name      entity.OpName     `json:"name"`
	LabelKey  string            `json:"labelKey"`
	ValueKind trigger.ValueKind `json:"valueKind"`
}

// PolicyCheck 内置检查项
type PolicyCheck struct {
	Key            string              `json:"key"`
	TitleKey       string              `json:"titleKey"`
	DescriptionKey string              `json:"descriptionKey"`
	Default        entity.Severity     `json:"default"`
	Params         []*PolicyCheckParam `json:"params,omitempty"`
}

// PolicyCheckParam 检查项参数定义
type PolicyCheckParam struct {
	Key      string            `json:"key"`
	TitleKey string            `json:"titleKey"`
	Type     trigger.FieldType `json:"type"`
	// EditorKey 值编辑器标识，与字段字典同源由后端下发：前端不再按 Type 自己猜控件
	EditorKey string `json:"editorKey"`
	// Numeric 该参数的期望值按数值比较（min/max 边界随之生效）。
	// 判定依据来自类型注册表，前端不再自己抄一份「哪些类型算数值」的清单
	Numeric  bool     `json:"numeric"`
	Default  any      `json:"default"`
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
	Options  []string `json:"options,omitempty"`
	Required bool     `json:"required"`
}

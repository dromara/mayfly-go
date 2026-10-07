// Package trigger 实现流程触发策略的注册表与求值引擎。
//
// 设计要点：
//   - 场景、字段、检查项、操作符、字段类型均为注册表驱动，新增业务场景（Mongo/ES/机器等）
//     只需在业务模块自身的包级 init 中注册，本包与前端通用组件无需改动
//   - 注册表只在「保存时校验」与「下发给前端的 schema」中使用，求值路径只做取值与比较，
//     避免每次求值都遍历注册表
//
// 与前端 models 的对应关系见 frontend/src/components/policy-builder/policyModel.ts。
package trigger

import (
	"context"
	"fmt"
	"maps"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/i18n"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// FieldType 字段值类型。取值为开放字符串，业务方可注册自定义类型
type FieldType string

const (
	TypeEnum       FieldType = "enum"
	TypeString     FieldType = "string"
	TypeNumber     FieldType = "number"
	TypeBool       FieldType = "bool"
	TypeStringList FieldType = "list<string>"
	TypeNumberList FieldType = "list<number>"
	TypeTime       FieldType = "time"
	TypeDuration   FieldType = "duration"
)

// ValueKind 操作符期望值的形态，决定前端渲染哪种值编辑器
type ValueKind string

const (
	ValueNone   ValueKind = "none"   // 无需期望值（如 empty）
	ValueSingle ValueKind = "single" // 单值
	ValueMulti  ValueKind = "multi"  // 多值（如 in）
	ValueRange  ValueKind = "range"  // 区间（如 between）
)

// FieldOption 枚举字段的候选取值
type FieldOption struct {
	Value string `json:"value"`
}

// FieldTypeDef 字段类型定义：默认操作符集合与前端编辑器标识
type FieldTypeDef struct {
	Name FieldType
	// DefaultOps 该类型字段的默认可用操作符
	DefaultOps []entity.OpName
	// EditorKey 前端值编辑器 key，未注册该类型时前端降级为文本输入
	EditorKey string
	// Numeric 该类型的期望值按数值比较：gt/lt/between 这类操作、以及参数的
	// min/max 边界都依赖它。时间戳与时长也归此类（存的是可排序的数值）
	//
	// 这个事实必须放在注册表里：散成「case TypeNumber, TypeTime, TypeDuration」后，
	// 校验、schema 下发、前端边界检查会各抄一份清单，新增类型时必然漏改其中一处
	Numeric bool
}

// TriggerField 触发策略可引用的一个字段（字段字典的一项）
type TriggerField struct {
	// Key 字段标识，条件叶子通过它取值
	Key string `json:"key"`
	// TitleKey 前端展示文案的 i18n key；后端只下发 key，文案归前端
	TitleKey string `json:"titleKey"`
	// Group 前端分组展示的分组 key（如 语句 / 目标对象 / 风险）
	Group string `json:"group"`
	// Type 字段值类型
	Type FieldType `json:"type"`
	// Options 枚举类型的候选取值
	Options []FieldOption `json:"options,omitempty"`
	// Ops 该字段允许的操作符，为空则取字段类型的默认集合
	Ops []entity.OpName `json:"ops,omitempty"`

	// Resolve 惰性派生字段值。为 nil 表示直接从原始输入取 Key 对应的值；
	// 非 nil 时只在条件真的引用到该字段才会被调用，避免为未使用的字段付出计算成本
	Resolve func(tc *Context) (any, error)
}

// AvailableOps 返回该字段实际可用的操作符
func (f TriggerField) AvailableOps() []entity.OpName {
	if len(f.Ops) > 0 {
		return f.Ops
	}
	if def, ok := fieldTypeRegistry[string(f.Type)]; ok {
		return def.DefaultOps
	}
	return nil
}

// CheckParam 检查项参数定义，驱动前端生成参数表单并约束取值
type CheckParam struct {
	Key      string        `json:"key"`
	TitleKey string        `json:"titleKey"`
	Type     FieldType     `json:"type"`
	Default  any           `json:"default"`
	Min      *float64      `json:"min,omitempty"`
	Max      *float64      `json:"max,omitempty"`
	Options  []FieldOption `json:"options,omitempty"`
	Required bool          `json:"required"`
}

// CheckDef 内置检查项：产品把常见运维意图实现为原生能力，用户只需勾选、调参、定级
type CheckDef struct {
	// Key 检查项全局唯一标识，建议 <场景>.<语义>，如 db.dml-without-where
	Key string
	// TitleKey 前端展示文案的 i18n key
	TitleKey string
	// DescriptionKey 前端参数说明文案的 i18n key
	DescriptionKey string
	// Summary 命中该检查项时用于服务端提示的消息 id（由各业务模块的 imsg 目录提供）。
	//
	// 与 TitleKey 分开是因为两者服务对象不同：TitleKey 是前端界面文案的 key，
	// Summary 要能被服务端渲染进拦截错误消息，让被拦的人知道「为什么被拦」而不只是一句「需提工单」
	Summary i18n.MsgId
	// BizTypes 适用的业务场景，为空表示适用于全部场景
	BizTypes []string
	// Params 参数 schema
	Params []CheckParam
	// Default 用户勾选后建议的默认处置级别
	Default entity.Severity
	// Evaluate 返回该检查项是否命中
	Evaluate func(ctx context.Context, tc *Context, params map[string]any) (bool, error)
}

// BizMeta 一个业务场景（bizType）向触发引擎注册的全部能力
type BizMeta struct {
	BizType string
	// Fields 该场景可被条件引用的字段字典
	Fields []TriggerField
	// Checks 该场景提供的内置检查项
	Checks []CheckDef
	// Presets 预置规则包：把该场景的常见意图固化为一组检查项选择，随 schema 下发。
	// 放在注册表而非前端，是为了让新增场景自带自己的模板，通用组件不必知道任何 check key
	Presets []PolicyPreset
	// SimulateFields 声明试算面板需要用户填写的输入项。前端据此渲染，不靠字段类型去猜
	// 哪些字段是「原始输入」、哪些是派生结果
	SimulateFields []SimulateInput
	// ConditionPresets 条件树的预置写法（如或签 / 会签），随 schema 下发给流程设计器直接套用。
	//
	// 与 Presets（检查项集合）分开是因为条件型场景没有检查项，但同样需要「把常见意图固化下来」；
	// 放在注册表而非前端，是为了让预置里引用的字段 key 与字段字典住在同一个文件里，
	// 改字段名时不会出现「后端改了、前端预设还写着旧 key」的分叉
	ConditionPresets []ConditionPreset
	// Approvable 声明该场景存在审批通道：流程通过后业务回调会把操作执行回去。
	//
	// 这里显式声明而不是去查回调注册表，是因为各模块的 RegisterBizHandler 由 starter
	// 以 goroutine 异步执行，用它做保存校验依据会出现「服务刚起来时拒绝合法配置」的竞态；
	// 声明与实现是否一致由 flow 应用层启动期自检负责报出来
	Approvable bool
	// Simulate 把管理员粘贴的原始输入（一条 SQL、一条命令）解析成求值所需的字段取值，用于策略试算。
	// 它必须与真实执行路径复用同一套事实计算函数，也要给出试算目标的资源路径：
	// 只补取值不补路径时，按资源定位的派生事实（如这台机器的命令黑名单）在试算里恒判不命中，
	// 管理员看到「放行」而线上真的被拦，试算就失去了它的唯一用途——预判真实结论
	Simulate func(ctx context.Context, raw map[string]string) (*SimulatedFacts, error)
	// GovernPaths 该场景的治理粒度：每条是从资源树根到被治理层的资源类型路径，
	// 段值取 consts.ResourceTypeXxx（如数据库为 实例/凭证/库，机器为 机器自身）。
	//
	// 用「路径」而不是「资源类型」，是因为治理粒度未必落在资源树的顶层节点：
	// 数据库按库治理，可勾选的节点在第三层。这份事实若不下发，前端就得自己写
	// 「实例类型 → 换成库那条路径」的分支，新场景（ES 索引、Mongo 集合）必须回来加分支
	//
	// 用数值而不是导入 tag 实体，是为了让策略域不依赖标签域；
	// 新增场景只需在自己的注册文件里声明治理路径，保存校验、资源树可勾选节点与求值都无需跟着改
	GovernPaths [][]int8
	// ConditionOnly 表示该注册项只提供字段字典，服务于流程内部的跳转条件、节点完成条件等布尔判定，
	// 由同一套求值引擎与校验器驱动。
	//
	// 它不是资源操作的触发场景：既没有内置检查项也无从试算（判定的输入是引擎自己算好的流程变量，
	// 不是管理员粘贴的一条 SQL），因此不出现在下发给触发策略编辑器的 schema 里，
	// 否则管理员会在策略页看到一个配不出任何触发规则的场景
	ConditionOnly bool
}

// SegmentDefinition 一个被引用的可复用条件组
type SegmentDefinition struct {
	// Ref 条件组标识，与条件节点的 Ref 对应
	Ref string
	// BizType 条件组所用的字段字典。条件只能引用本场景注册的字段，
	// 跨场景引用会让同名字段含义不同，因此保存时要拦住、求值时也要兜底
	BizType string
	// Condition 条件树本体
	Condition *entity.RuleNode
}

// SegmentResolver 按引用标识取条件组定义，取不到时返回 nil。
//
// 求值与保存校验都只通过它读外部数据，内核因此不依赖持久层，
// 也便于单测用内存 map 驱动「引用不存在 / 循环引用」这两类判定
type SegmentResolver func(ref string) (*SegmentDefinition, bool)

// ConditionPreset 一个条件树预置写法：短名 + 一句话说明 + 可直接落库的条件树
type ConditionPreset struct {
	Key      string `json:"key"`
	TitleKey string `json:"titleKey"`
	// DescriptionKey 该预置等价于什么规则，用于 tooltip，避免管理员只能靠名字猜
	DescriptionKey string           `json:"descriptionKey"`
	RuleNode       *entity.RuleNode `json:"ruleNode"`
}

// PolicyPreset 预置规则包：一组检查项的选择集合，套用后该场景只保留这些检查项
type PolicyPreset struct {
	Key       string   `json:"key"`
	TitleKey  string   `json:"titleKey"`
	CheckKeys []string `json:"checkKeys"`
}

// SimulateInput 试算面板的一个输入项
type SimulateInput struct {
	Key      string    `json:"key"`
	TitleKey string    `json:"titleKey"`
	Type     FieldType `json:"type"`
	// EditorKey 覆盖按类型推导的控件标识：资源引用（目标库、目标机器）这类输入必须用
	// 资源树选择——资源 id 是库里的主键，运维只知道资源叫什么名字，不可能背出 id
	EditorKey string `json:"editorKey,omitempty"`
	// NameKey 资源选择控件选中资源后，资源显示名回填到的输入项 key（如库名）。
	// 该输入项仍可手改；声明它省掉的是「选完资源再抄一遍名字」这一步
	NameKey string `json:"nameKey,omitempty"`
	// PlaceholderKey 输入示例文案的 i18n key，长文本输入（SQL/命令）用它提示格式
	PlaceholderKey string `json:"placeholderKey,omitempty"`
	Required       bool   `json:"required"`
	// Multiline 该输入是否需要多行控件，由声明该输入的场景给出：
	// 「字符串就用 textarea」是前端的猜法，新增一个单行字符串输入会被它带成三行高
	Multiline bool `json:"multiline"`
}

// SimulatedFacts 一次试算输入的解析结果
type SimulatedFacts struct {
	// Raw 试算面板填出来的原始字段值
	Raw map[string]string
	// Attributes 由 Raw 派生出的求值事实（语句类型、是否含 WHERE、命令名等）
	Attributes map[string]any
	// CodePaths 试算目标的资源标签路径，派生字段按它去查该资源的配置
	CodePaths []string
}

// AvailableSeverities 返回某场景可配置到规则上的处置级别。
//
// 「需审批」依赖该场景声明了审批通道（工单批完有人把操作执行回去）；
// 未声明时只提供「仅提醒」与「禁止执行」，避免管理员配出一个无法落地的级别
func AvailableSeverities(bizType string) []entity.Severity {
	severities := []entity.Severity{entity.SeverityWarning, entity.SeverityForbidden}
	if meta, ok := BizMetaOf(bizType); ok && meta.Approvable {
		severities = append([]entity.Severity{entity.SeverityRequired}, severities...)
	}
	return severities
}

var (
	mu                sync.RWMutex
	bizRegistry       = map[string]*BizMeta{}
	checkRegistry     = map[string]*CheckDef{}
	opRegistry        = map[entity.OpName]*OpDef{}
	fieldTypeRegistry = map[string]*FieldTypeDef{}
)

// RegisterType 注册字段类型
func RegisterType(def FieldTypeDef) {
	mu.Lock()
	defer mu.Unlock()
	if _, exist := fieldTypeRegistry[string(def.Name)]; exist {
		panic(fmt.Sprintf("trigger: field type %q registered twice", def.Name))
	}
	fieldTypeRegistry[string(def.Name)] = &def
}

// RegisterOp 注册操作符
func RegisterOp(def OpDef) {
	switch def.ValueKind {
	case ValueNone, ValueSingle, ValueMulti, ValueRange:
	default:
		panic(fmt.Sprintf("trigger: operator %q declares the value kind %q, which the value editor cannot render", def.Name, def.ValueKind))
	}

	mu.Lock()
	defer mu.Unlock()
	if _, exist := opRegistry[def.Name]; exist {
		panic(fmt.Sprintf("trigger: operator %q registered twice", def.Name))
	}
	opRegistry[def.Name] = &def
}

// RegisterBiz 注册业务场景（字段字典 + 检查项 + 试算解析）
func RegisterBiz(meta BizMeta) {
	mu.Lock()
	defer mu.Unlock()
	if _, exist := bizRegistry[meta.BizType]; exist {
		panic(fmt.Sprintf("trigger: biz type %q registered twice", meta.BizType))
	}
	registered := meta
	bizRegistry[meta.BizType] = &registered
	for _, check := range meta.Checks {
		if _, exist := checkRegistry[check.Key]; exist {
			panic(fmt.Sprintf("trigger: check %q registered twice", check.Key))
		}
		defined := check
		checkRegistry[check.Key] = &defined
	}
}

// BizMetas 返回全部已注册场景（含只提供条件字段字典的场景），按 bizType 稳定排序
func BizMetas() []BizMeta {
	mu.RLock()
	defer mu.RUnlock()
	metas := make([]BizMeta, 0, len(bizRegistry))
	for _, meta := range bizRegistry {
		metas = append(metas, *meta)
	}
	slices.SortFunc(metas, func(left, right BizMeta) int { return strings.Compare(left.BizType, right.BizType) })
	return metas
}

// BizMetaOf 返回指定场景的注册信息
func BizMetaOf(bizType string) (BizMeta, bool) {
	mu.RLock()
	defer mu.RUnlock()
	meta, ok := bizRegistry[bizType]
	if !ok {
		return BizMeta{}, false
	}
	return *meta, true
}

// FieldOf 返回场景字段字典中的某个字段
func FieldOf(bizType, fieldKey string) (TriggerField, bool) {
	mu.RLock()
	defer mu.RUnlock()
	meta, ok := bizRegistry[bizType]
	if !ok {
		return TriggerField{}, false
	}
	for _, field := range meta.Fields {
		if field.Key == fieldKey {
			return field, true
		}
	}
	return TriggerField{}, false
}

// CheckOf 返回指定 key 的检查项定义
func CheckOf(key string) (CheckDef, bool) {
	mu.RLock()
	defer mu.RUnlock()
	check, ok := checkRegistry[key]
	if !ok {
		return CheckDef{}, false
	}
	return *check, true
}

// ChecksOf 返回适用于指定场景的检查项（含与场景无关的通用检查项）。
// 按 key 稳定排序：注册表是 map，不排序会让前端检查项清单每次刷新都重排，用户无法按位置记忆规则
func ChecksOf(bizType string) []CheckDef {
	mu.RLock()
	defer mu.RUnlock()
	checks := make([]CheckDef, 0, len(checkRegistry))
	for _, key := range slices.Sorted(maps.Keys(checkRegistry)) {
		check := checkRegistry[key]
		if len(check.BizTypes) == 0 {
			checks = append(checks, *check)
			continue
		}
		for _, support := range check.BizTypes {
			if support == bizType {
				checks = append(checks, *check)
				break
			}
		}
	}
	return checks
}

// OpOf 返回操作符定义
func OpOf(name entity.OpName) (OpDef, bool) {
	mu.RLock()
	defer mu.RUnlock()
	def, ok := opRegistry[name]
	if !ok {
		return OpDef{}, false
	}
	return *def, true
}

// TypeOf 返回字段类型定义
func TypeOf(name FieldType) (FieldTypeDef, bool) {
	mu.RLock()
	defer mu.RUnlock()
	def, ok := fieldTypeRegistry[string(name)]
	if !ok {
		return FieldTypeDef{}, false
	}
	return *def, true
}

// TriggerBizMetas 返回可作为触发策略配置的场景。
//
// 流程内部条件用的字段字典只服务布尔判定，既没有检查项也没有试算输入：
// 混进策略 schema 会让编辑器渲染出一张永远配不出规则的卡片，
// 也不该出现在「可治理资源类型」的汇总里。这个筛选只在这里表达一次
func TriggerBizMetas() []BizMeta {
	metas := make([]BizMeta, 0, len(BizMetas()))
	for _, meta := range BizMetas() {
		if meta.ConditionOnly {
			continue
		}
		metas = append(metas, meta)
	}
	return metas
}

// GovernedBizTypes 返回能治理这些绑定资源路径的场景 bizType（去重，按注册顺序）。
//
// 匹配按路径而不是按单个类型：绑定在实例层的标签会连带治理其下的库（祖先展开），
// 因此绑定路径与治理路径互为前缀即算命中；只比末级类型会把「绑在实例上」误判成不治理
func GovernedBizTypes(boundPaths [][]int8) []string {
	bizTypes := make([]string, 0, len(boundPaths))
	seen := make(map[string]bool)
	for _, meta := range TriggerBizMetas() {
		if meta.ConditionOnly || seen[meta.BizType] {
			continue
		}
		for _, bound := range boundPaths {
			if governsPath(meta.GovernPaths, bound) {
				bizTypes = append(bizTypes, meta.BizType)
				seen[meta.BizType] = true
				break
			}
		}
	}
	return bizTypes
}

// ScenarioGoverned 该场景在这些绑定路径里是否真能被命中。
//
// 保存校验据此拦住「配了半天却永不命中」的策略：策略编辑器会列出所有已注册场景，
// 管理员给机器命令配了规则却只绑了数据库实例时，运行时永远不会走到那个场景，界面也不报错
func ScenarioGoverned(bizType string, boundPaths [][]int8) bool {
	meta, ok := BizMetaOf(bizType)
	if !ok {
		return false
	}
	for _, bound := range boundPaths {
		if governsPath(meta.GovernPaths, bound) {
			return true
		}
	}
	return false
}

// governsPath 任一治理路径与绑定路径互为前缀即命中（短的那条是绑定层级，长的是治理层级）
func governsPath(governPaths [][]int8, bound []int8) bool {
	if len(bound) == 0 {
		return false
	}
	for _, govern := range governPaths {
		if len(govern) == 0 {
			continue
		}
		short, long := govern, bound
		if len(short) > len(long) {
			short, long = long, govern
		}
		matched := true
		for i, resourceType := range short {
			if long[i] != resourceType {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

// DeclaredGovernPaths 返回所有场景声明的治理路径（去重，顺序稳定），供 schema 下发。
//
// 去重放在注册表而不是各调用点遍历，才能保证「新场景注册完就能被绑定」只依赖注册这一件事；
// 数据库这类多场景若共用同一条路径，也只会在资源树里出现一个可勾节点
func DeclaredGovernPaths() [][]int8 {
	paths := make([][]int8, 0)
	seen := make(map[string]bool)
	for _, meta := range TriggerBizMetas() {
		if meta.ConditionOnly {
			continue
		}
		for _, path := range meta.GovernPaths {
			key := strings.Join(pathSegments(path), "/")
			if len(path) == 0 || seen[key] {
				continue
			}
			seen[key] = true
			paths = append(paths, path)
		}
	}
	return paths
}

// pathSegments 把一条资源类型路径转成可比较的字符串段，仅用于注册表内部去重
func pathSegments(path []int8) []string {
	result := make([]string, 0, len(path))
	for _, resourceType := range path {
		result = append(result, strconv.Itoa(int(resourceType)))
	}
	return result
}

// GovernanceDeclarationProblems 报告该场景的治理路径声明本身是否成立。
//
// 判断放在域层、报日志放在应用层：写错的治理路径不会报错，只会表现为「这个场景在资源树里
// 永远选不到、配了也永远不命中」，界面上看不出任何异常，因此必须在能被看见的时机主动报出来。
// 判据用 consts.KnownResourceTypes，与契约测试同一份清单，避免两处各说各话
func GovernanceDeclarationProblems(meta BizMeta) []string {
	if meta.ConditionOnly {
		// 只提供字段字典的流程内条件场景不治理资源，没有治理路径可言
		return nil
	}
	problems := make([]string, 0, len(meta.GovernPaths)+1)
	if len(meta.GovernPaths) == 0 {
		problems = append(problems, "declares no governance path, so it can never be bound to a process definition")
	}
	for _, path := range meta.GovernPaths {
		if len(path) == 0 {
			problems = append(problems, "declares an empty governance path")
			continue
		}
		for _, resourceType := range path {
			if !consts.IsResourceType(resourceType) {
				problems = append(problems, fmt.Sprintf("governance path %v contains the unknown resource type %d", path, resourceType))
			}
		}
	}
	return problems
}

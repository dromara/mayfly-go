package service

import (
	"context"
	"fmt"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/pkg/i18n"
	"sort"
	"sync"
	"time"
)

// MetricDefinition 指标元信息，供保存校验与前端指标下拉共用同一数据源。
// 新增指标的展示文案必须在此登记，禁止在消费方（通知渲染/前端枚举）另行维护映射
type MetricDefinition struct {
	Key        string     `json:"key"`   // 指标名，如 cpu_rate
	Label      string     `json:"label"` // 展示名（前端 i18n key），如 alert.metricCpuRate
	LabelMsgId i18n.MsgId // 展示名（服务端 i18n 消息ID），通知渲染用；0 表示未配置，回退为原始 key
	Unit       string     `json:"unit"`      // 单位后缀，如 %
	IsPercent  bool       `json:"isPercent"` // 是否为百分比指标（前端阈值输入 0-100）
	HasRange   bool       `json:"hasRange"`  // 是否存在合法取值区间
	Min        float64    `json:"min"`       // 阈值允许的最小值
	Max        float64    `json:"max"`       // 阈值允许的最大值
}

// AlertEvaluator 告警评估器接口 —— 每种资源类型一个实现
//
// 新增一种资源类型评估器（如 Redis/DB）的全部步骤，全程无需改动引擎/校验/前端/通知链路：
//
//  1. 在 application/evaluator 包新增实现文件，实现本接口并保留
//     `var _ service.AlertEvaluator = (*XxxEvaluator)(nil)` 编译断言。
//     评估器为单例且被引擎并发调用（最多 10 规则并发），实现必须并发安全
//  2. 在该文件 init() 中调用 registerEvaluator(&XxxEvaluator{}) 完成注册
//  3. 在评估器内登记指标集（MetricDefinition 含 Label/LabelMsgId 展示文案、单位、
//     阈值区间约束），并在 imsg 两语言文件补充文案；指标提取函数使用具体数据
//     类型直接取值，禁止 interface{} + 反射
//  4. 为评估器接通运行时数据源（参照机器评估器：资源模块应用层接口 inject 注入 +
//     infra 缓存，评估循环读缓存而非直连资源实例）
//  5. 在 evaluator/contract_test.go 新增契约用例并保证全绿，按其 Evaluate 语义
//     checklist 补齐自身行为测试
//
// 以下能力全部自动跟随，无需任何改动：引擎评估循环（按 ResourceType 分发）、
// 规则保存校验（SupportsResourceType/SupportedMetric/阈值区间）、前端指标下拉
// （/alert-rules/metrics 同源下发）、通知与阈值文案 i18n（FindMetric + LabelMsgId）
type AlertEvaluator interface {
	// ResourceType 返回该评估器负责的资源类型（consts.ResourceType*）
	ResourceType() int8

	// Metrics 返回该资源类型支持的指标集合，用于规则保存校验与前端指标下拉
	Metrics() []MetricDefinition

	// Evaluate 评估指定资源是否满足告警条件
	Evaluate(ctx context.Context, param *EvalParam) (*EvalResult, error)

	// ResourceName 返回资源展示名，用于告警事件冗余存储与前端展示；
	// 无法解析时返回空字符串（不视为错误）
	ResourceName(resourceId uint64) string

	// ResourceExists 判断资源是否存在。规则保存时用于拦截指向已删除资源的配置，
	// 否则该规则永远不会命中任何资源，用户无感知
	ResourceExists(resourceId uint64) bool

	// ResourceIdsByCodes 通过资源 code 列表查询资源 ID 列表，用于标签范围展开
	ResourceIdsByCodes(codes []string) []uint64
}

// EvalParam 评估参数
type EvalParam struct {
	Rule       *entity.AlertRule
	ResourceId uint64
}

// EvalResult 评估结果
//
// 不变式：Items 与规则 Condition.Items 一一对应（下标对齐），指标无法取值时不跳过而是
// 记录 Err，否则 AND/OR 语义与 Duration 计算都会因下标错位而失真。
type EvalResult struct {
	Triggered bool            // 是否触发告警
	Evaluated bool            // 本次判定是否可靠；false 时调用方必须跳过状态迁移（不可触发也不可恢复）
	Items     []ConditionEval // 各条件项的评估结果（与 Condition.Items 下标对齐）
}

// ConditionEval 单个条件项的评估结果
type ConditionEval struct {
	Metric       string  // 指标名
	CurrentValue float64 // 当前值
	Threshold    float64 // 阈值
	Compare      string  // 比较方式
	Satisfied    bool    // 是否满足
	Err          string  // 非空表示该条件本次无法评估（指标缺失/取值失败/比较方式非法）
}

// ---- 评估器注册表 ----

var (
	evaluators = make(map[int8]AlertEvaluator)
	evalMu     sync.RWMutex
)

// RegisterEvaluator 注册评估器（在 init.Init() 阶段调用，此时 IOC 已完成依赖注入）
func RegisterEvaluator(evaluator AlertEvaluator) {
	evalMu.Lock()
	defer evalMu.Unlock()
	evaluators[evaluator.ResourceType()] = evaluator
}

// GetEvaluator 根据资源类型获取对应的评估器
func GetEvaluator(resourceType int8) (AlertEvaluator, bool) {
	evalMu.RLock()
	defer evalMu.RUnlock()
	e, ok := evaluators[resourceType]
	return e, ok
}

// SupportsResourceType 是否存在负责该资源类型的评估器。
// 无评估器的资源类型规则不会被任何评估周期处理，因此保存时必须拒绝。
func SupportsResourceType(resourceType int8) bool {
	_, ok := GetEvaluator(resourceType)
	return ok
}

// SupportedResourceTypes 返回所有已注册评估器的资源类型（升序），供前端资源类型下拉使用
func SupportedResourceTypes() []int8 {
	evalMu.RLock()
	defer evalMu.RUnlock()
	types := make([]int8, 0, len(evaluators))
	for rt := range evaluators {
		types = append(types, rt)
	}
	sort.Slice(types, func(i, j int) bool { return types[i] < types[j] })
	return types
}

// MetricsOf 获取资源类型支持的指标集合；资源类型无评估器时返回 nil
func MetricsOf(resourceType int8) []MetricDefinition {
	e, ok := GetEvaluator(resourceType)
	if !ok {
		return nil
	}
	return e.Metrics()
}

// SupportedMetric 指标是否属于该资源类型
func SupportedMetric(resourceType int8, metric string) bool {
	_, ok := FindMetric(resourceType, metric)
	return ok
}

// FindMetric 查找指标元信息，用于保存校验阈值区间与前端下拉渲染
func FindMetric(resourceType int8, metric string) (MetricDefinition, bool) {
	for _, m := range MetricsOf(resourceType) {
		if m.Key == metric {
			return m, true
		}
	}
	return MetricDefinition{}, false
}

// ValidateMetrics 校验指标元信息的完整性与一致性，单一真源供两处消费：
// 评估器注册时的启动自检（fail-fast 拒绝带缺陷元信息上线）与评估器契约测试（测试期拦截）。
// 这些元信息同时是保存校验、前端下拉、通知渲染的正确性前提
func ValidateMetrics(defs []MetricDefinition) error {
	if len(defs) == 0 {
		return fmt.Errorf("metrics is empty")
	}
	seen := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		if def.Key == "" {
			return fmt.Errorf("metric key is empty, label=%s", def.Label)
		}
		if _, ok := seen[def.Key]; ok {
			return fmt.Errorf("duplicate metric key: %s", def.Key)
		}
		seen[def.Key] = struct{}{}

		if def.Label == "" {
			return fmt.Errorf("metric %s: label(frontend i18n key) is empty", def.Key)
		}
		// LabelMsgId 允许为 0（服务端展示回退原始 key），但配置了就必须已登记文案，
		// 否则通知渲染出的指标名为空串
		if def.LabelMsgId != 0 && i18n.T(def.LabelMsgId) == "" {
			return fmt.Errorf("metric %s: label msgId %d has no registered text", def.Key, def.LabelMsgId)
		}
		if def.HasRange && def.Min > def.Max {
			return fmt.Errorf("metric %s: invalid threshold range [%v, %v]", def.Key, def.Min, def.Max)
		}
		if def.IsPercent && (!def.HasRange || def.Min < 0 || def.Max > 100) {
			return fmt.Errorf("metric %s: percent metric must have range within [0, 100]", def.Key)
		}
	}
	return nil
}

// ResourceExists 资源是否存在；资源类型无评估器时返回 false
func ResourceExists(resourceType int8, resourceId uint64) bool {
	e, ok := GetEvaluator(resourceType)
	if !ok {
		return false
	}
	return e.ResourceExists(resourceId)
}

// ResourceName 获取资源展示名；资源类型无评估器或解析失败时返回空字符串
func ResourceName(resourceType int8, resourceId uint64) string {
	e, ok := GetEvaluator(resourceType)
	if !ok {
		return ""
	}
	return e.ResourceName(resourceId)
}

// ---- 越界时间追踪器接口（由 infra/cache 实现） ----

// BreachTracker 首次越界时间追踪器，用于防抖判断
type BreachTracker interface {
	// GetFirstBreach 获取首次越界时间，零值表示无记录
	GetFirstBreach(ruleId, resourceId uint64) time.Time
	// SetFirstBreach 记录首次越界时间
	SetFirstBreach(ruleId, resourceId uint64, t time.Time)
	// DelFirstBreach 删除越界记录（恢复时调用）
	DelFirstBreach(ruleId, resourceId uint64)
}

// ---- 连续计数器接口（由 infra/cache 实现） ----

// ConsecutiveCounter 连续计数器，用于 TriggerCount / RecoveryCount 防抖
type ConsecutiveCounter interface {
	// Get 获取计数器值
	Get(key string) int64
	// Incr 自增并返回新值
	Incr(key string) int64
	// Reset 重置计数器
	Reset(key string)
}

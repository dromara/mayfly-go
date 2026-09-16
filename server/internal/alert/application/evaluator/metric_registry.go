package evaluator

import (
	"fmt"
	"mayfly-go/internal/alert/domain/service"
	"sort"
)

// MetricExtractor 指标提取器：将指标元信息与取值逻辑封装为自包含单元。
//
// 设计动机：传统 switch-case 提取指标违反开闭原则——每新增一个指标都要修改
// extractMetric 的分支。注册表模式下，新增指标只需在 init 中注册一条记录，
// 无需触碰任何现有代码。
//
// 业界参考：Prometheus exporter 的 Collector 模式、Datadog Agent 的 Check 模式
// 均采用"指标自描述 + 注册"的方式实现可扩展的指标采集。
type MetricExtractor struct {
	// Definition 指标元信息，供前端下拉渲染与保存校验共用
	Definition service.MetricDefinition

	// Extract 从资源原始状态数据中提取该指标的数值。
	// stats 为评估器获取的资源状态对象（类型由评估器决定），
	// statsErr 为获取状态时的错误（非 nil 表示状态不可得）。
	// 返回 (value, nil) 表示成功，(0, error) 表示取值失败。
	//
	// 特殊场景：某些指标（如"在线状态"）在 statsErr 非 nil 时仍有语义
	// （如视为离线），此时应返回 (0, nil) 而非 error。
	Extract func(stats interface{}, statsErr error) (float64, error)
}

// metricRegistry 指标注册表：key=指标名，value=提取器。
// 每个评估器维护自己的注册表，避免不同资源类型的指标命名冲突。
type metricRegistry map[string]*MetricExtractor

// register 注册一个指标提取器。重复注册同名指标时会覆盖（init 阶段调用，不会出现竞态）。
func (r metricRegistry) register(ext *MetricExtractor) {
	r[ext.Definition.Key] = ext
}

// definitions 返回注册表中所有指标的元信息列表，顺序稳定（按 key 排序）。
func (r metricRegistry) definitions() []service.MetricDefinition {
	defs := make([]service.MetricDefinition, 0, len(r))
	for _, ext := range r {
		defs = append(defs, ext.Definition)
	}
	// 按 key 排序保证前端下拉顺序稳定
	sortMetricDefinitions(defs)
	return defs
}

// extract 根据指标名查找并执行提取函数。
// 指标不存在时返回 error，由调用方记录日志并保持 ConditionEval 下标对齐。
func (r metricRegistry) extract(stats interface{}, statsErr error, metric string) (float64, error) {
	ext, ok := r[metric]
	if !ok {
		return 0, fmt.Errorf("unknown metric: %s", metric)
	}
	return ext.Extract(stats, statsErr)
}

// contains 判断注册表中是否存在指定指标
func (r metricRegistry) contains(metric string) bool {
	_, ok := r[metric]
	return ok
}

// sortMetricDefinitions 按指标 key 升序排序，保证前端下拉顺序稳定
func sortMetricDefinitions(defs []service.MetricDefinition) {
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Key < defs[j].Key
	})
}

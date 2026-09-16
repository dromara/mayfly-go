package evaluator

import (
	"context"
	"fmt"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/service"
	machineapp "mayfly-go/internal/machine/application"
	pkgconsts "mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"reflect"
)

// 机器指标名
const (
	MetricCpuRate   = "cpu_rate"
	MetricMemRate   = "mem_rate"
	MetricDiskUsage = "disk_usage"
	MetricStatus    = "status"
)

// MachineEvaluator 机器指标评估器
type MachineEvaluator struct {
	machineApp machineapp.Machine `inject:"T"`
	// metrics 指标注册表：key=指标名，value=提取器（定义 + 取值逻辑自包含）
	metrics metricRegistry
}

var _ service.AlertEvaluator = (*MachineEvaluator)(nil)

func init() {
	registerEvaluatorFactory[MachineEvaluator]()
}

// initMetrics 初始化机器指标注册表。
// 新增指标只需在此追加一行注册，无需修改 extractMetric 等现有逻辑。
func (e *MachineEvaluator) initMetrics() {
	e.metrics = make(metricRegistry)
	e.metrics.register(&MetricExtractor{
		Definition: service.MetricDefinition{Key: MetricCpuRate, Label: "alert.metricCpuRate", Unit: "%", IsPercent: true, HasRange: true, Min: 0, Max: 100},
		Extract:    extractCpuRate,
	})
	e.metrics.register(&MetricExtractor{
		Definition: service.MetricDefinition{Key: MetricMemRate, Label: "alert.metricMemRate", Unit: "%", IsPercent: true, HasRange: true, Min: 0, Max: 100},
		Extract:    extractMemRate,
	})
	e.metrics.register(&MetricExtractor{
		Definition: service.MetricDefinition{Key: MetricDiskUsage, Label: "alert.metricDiskUsage", Unit: "%", IsPercent: true, HasRange: true, Min: 0, Max: 100},
		Extract:    extractDiskUsage,
	})
	// status 指标特殊：不依赖 stats 对象，以"能否取到新鲜状态"判定在线/离线
	e.metrics.register(&MetricExtractor{
		Definition: service.MetricDefinition{Key: MetricStatus, Label: "alert.metricStatus", Unit: "", HasRange: true, Min: 0, Max: 1},
		Extract:    extractStatus,
	})
}

// ensureMetrics 懒初始化指标注册表（IoC 创建实例后首次调用时触发）
func (e *MachineEvaluator) ensureMetrics() {
	if e.metrics == nil {
		e.initMetrics()
	}
}

func (e *MachineEvaluator) ResourceType() int8 {
	return pkgconsts.ResourceTypeMachine
}

func (e *MachineEvaluator) Metrics() []service.MetricDefinition {
	e.ensureMetrics()
	return e.metrics.definitions()
}

// ResourceExists 机器是否存在（含已逻辑删除判定交由 IOC 层过滤）
func (e *MachineEvaluator) ResourceExists(resourceId uint64) bool {
	machine, err := e.machineApp.GetById(resourceId)
	return err == nil && machine != nil
}

// ResourceName 机器展示名，取不到时返回空串（不影响告警主流程）
func (e *MachineEvaluator) ResourceName(resourceId uint64) string {
	machine, err := e.machineApp.GetById(resourceId)
	if err != nil || machine == nil {
		logx.Warnf("[alert] get machine[%d] name error: %v", resourceId, err)
		return ""
	}
	return machine.Name
}

// ResourceIdsByCodes 通过机器 code 列表查询机器 ID 列表
func (e *MachineEvaluator) ResourceIdsByCodes(codes []string) []uint64 {
	if len(codes) == 0 {
		return nil
	}
	machines, err := e.machineApp.ListByCond(model.NewCond().In("code", codes), "id")
	if err != nil {
		logx.Errorf("[alert] list machines by codes error: %s", err.Error())
		return nil
	}
	ids := make([]uint64, 0, len(machines))
	for _, m := range machines {
		ids = append(ids, m.Id)
	}
	return ids
}

// Evaluate 评估机器告警条件。
//
// 结果严格保持 Items 与 rule.Condition.Items 下标对齐：指标无法取值时不跳过，而是记录 Err。
// 若本次有条件无法取值导致判定结论不可靠，则 Evaluated=false，调用方必须跳过状态迁移，
// 避免把"取不到数据"误判为"已恢复"。
func (e *MachineEvaluator) Evaluate(ctx context.Context, param *service.EvalParam) (*service.EvalResult, error) {
	e.ensureMetrics() // 确保指标注册表已初始化

	condition := param.Rule.Condition
	if condition == nil || len(condition.Items) == 0 {
		return &service.EvalResult{Evaluated: true}, nil
	}

	stats, statsErr := e.machineApp.GetMachineStats(param.ResourceId)
	// 机器状态取不到时：
	// - 若规则使用了 status 指标，继续评估（extractStatus 会处理 statsErr 返回离线值）
	// - 若规则未使用 status 指标，本次评估不可靠，返回 Evaluated=false 让引擎跳过状态迁移
	if statsErr != nil && !e.usesStatusMetric(condition) {
		logx.Warnf("[alert] machine[%d] stats unavailable: %s, evaluation unreliable", param.ResourceId, statsErr.Error())
		return &service.EvalResult{Evaluated: false}, nil
	}
	logx.Debugf("[alert] machine[%d] stats retrieved successfully", param.ResourceId)

	result := &service.EvalResult{Evaluated: true}
	for _, item := range condition.Items {
		eval := service.ConditionEval{
			Metric:    item.Metric,
			Threshold: item.Value,
			Compare:   item.Compare,
		}

		currentValue, err := e.extractMetric(stats, statsErr, item.Metric)
		if err != nil {
			// 下标保持对齐：记录失败原因而非跳过该条件
			eval.Err = err.Error()
			logx.Warnf("[alert] machine[%d] metric[%s] eval error: %s", param.ResourceId, item.Metric, err.Error())
		} else if !entity.IsValidCompare(item.Compare) {
			eval.Err = fmt.Sprintf("invalid compare operator: %s", item.Compare)
			logx.Warnf("[alert] machine[%d] metric[%s] invalid compare operator: %s", param.ResourceId, item.Metric, item.Compare)
		} else {
			eval.CurrentValue = currentValue
			eval.Satisfied = compareValue(currentValue, item.Compare, item.Value)
		}
		result.Items = append(result.Items, eval)
	}

	result.Triggered = e.isTriggered(condition.Operator, result.Items)
	// 判定可靠性：已触发即为可靠；未触发但存在无法取值的条件项时结论不可靠，
	// 必须禁止调用方据此把告警置为"已恢复"
	result.Evaluated = result.Triggered || !hasUnevaluableItem(result.Items)
	return result, nil
}

// hasUnevaluableItem 是否存在无法取值的条件项
func hasUnevaluableItem(items []service.ConditionEval) bool {
	for _, item := range items {
		if item.Err != "" {
			return true
		}
	}
	return false
}

// isTriggered 按组合方式判定是否触发：
//   - and：所有条件都满足；任一条件不可判定则整体不触发（由 Evaluated 标记为不可靠）
//   - or ：任一条件满足即触发；无任一满足且存在不可判定条件时结论不可靠
func (e *MachineEvaluator) isTriggered(operator string, items []service.ConditionEval) bool {
	if operator == "or" {
		for _, item := range items {
			if item.Satisfied {
				return true
			}
		}
		return false
	}
	// 默认 and
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		if !item.Satisfied {
			return false
		}
	}
	return true
}

// usesStatusMetric 条件中是否使用了在线状态指标
func (e *MachineEvaluator) usesStatusMetric(condition *entity.AlertCondition) bool {
	if condition == nil {
		return false
	}
	for _, item := range condition.Items {
		if item.Metric == MetricStatus {
			return true
		}
	}
	return false
}

// extractMetric 从已获取的机器状态中提取指定指标的值（单次 GetMachineStats，避免 N+1）。
//
// 完全委托指标注册表，无特殊分支：每个指标的 Extract 函数自行处理 statsErr。
// 新增指标只需在 initMetrics 中注册，此处逻辑完全不变。
func (e *MachineEvaluator) extractMetric(stats interface{}, statsErr error, metric string) (float64, error) {
	return e.metrics.extract(stats, statsErr, metric)
}

// extractStatus 在线状态指标：取不到 stats 时视为离线(0)，取到时视为在线(1)。
func extractStatus(_ interface{}, statsErr error) (float64, error) {
	if statsErr != nil {
		return 0, nil // 离线
	}
	return 1, nil // 在线
}

func extractCpuRate(stats interface{}, statsErr error) (float64, error) {
	if statsErr != nil {
		return 0, fmt.Errorf("machine stats unavailable: %w", statsErr)
	}
	v := reflect.ValueOf(stats)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	cpu := v.FieldByName("CPU")
	if !cpu.IsValid() {
		return 0, fmt.Errorf("CPU field not found in stats")
	}
	idle := cpu.FieldByName("Idle")
	if !idle.IsValid() {
		return 0, fmt.Errorf("CPU.Idle field not found")
	}
	return 100 - idle.Float(), nil
}

func extractMemRate(stats interface{}, statsErr error) (float64, error) {
	if statsErr != nil {
		return 0, fmt.Errorf("machine stats unavailable: %w", statsErr)
	}
	v := reflect.ValueOf(stats)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	mem := v.FieldByName("MemInfo")
	if !mem.IsValid() {
		return 0, fmt.Errorf("MemInfo field not found in stats")
	}
	total := mem.FieldByName("Total").Uint()
	available := mem.FieldByName("Available").Uint()
	// 总内存为 0 说明数据不可信，返回错误而非伪造成 0%，避免误判为"已恢复"
	if total == 0 {
		return 0, fmt.Errorf("memory total is 0, stats data unreliable")
	}
	return float64(total-available) / float64(total) * 100, nil
}

func extractDiskUsage(stats interface{}, statsErr error) (float64, error) {
	if statsErr != nil {
		return 0, fmt.Errorf("machine stats unavailable: %w", statsErr)
	}
	v := reflect.ValueOf(stats)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	fsInfos := v.FieldByName("FSInfos")
	if !fsInfos.IsValid() || fsInfos.Len() == 0 {
		return 0, fmt.Errorf("no filesystem info in stats")
	}

	// 优先根分区；不存在时取所有挂载点中最高使用率（告警场景关注最满的盘）
	var maxUsage float64
	var maxFound bool
	for i := 0; i < fsInfos.Len(); i++ {
		fs := fsInfos.Index(i)
		used := fs.FieldByName("Used").Uint()
		free := fs.FieldByName("Free").Uint()
		total := used + free
		if total == 0 {
			continue
		}
		usage := float64(used) / float64(total) * 100
		if fs.FieldByName("MountPoint").String() == "/" {
			return usage, nil
		}
		if !maxFound || usage > maxUsage {
			maxUsage, maxFound = usage, true
		}
	}
	if !maxFound {
		return 0, fmt.Errorf("no usable filesystem partition found")
	}
	return maxUsage, nil
}

// compareValue 比较当前值与阈值；未知比较方式返回 false（保存侧已校验，此处为兜底）
func compareValue(current float64, compare string, threshold float64) bool {
	switch compare {
	case "gt":
		return current > threshold
	case "gte":
		return current >= threshold
	case "lt":
		return current < threshold
	case "lte":
		return current <= threshold
	case "eq":
		return current == threshold
	case "neq":
		return current != threshold
	default:
		return false
	}
}

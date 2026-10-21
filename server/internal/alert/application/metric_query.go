package application

import (
	"context"
	"fmt"

	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/service"
	"mayfly-go/pkg/errorx"
)

// MetricQuery 可观测指标查询端口（阶段 4.2 收口）：供 ai/tools/monitor 等只读消费方
// 按资源即时取指标当前值，与告警引擎同源同口径。
//
// 存在动机（依赖倒置 + 契约收口）：指标取值复用 AlertEvaluator.Evaluate 通道——
// 以「阈值 0 恒不触发的合成规则」借评估器的取值逻辑拿当前值。该做法依赖两个评估器
// 实现约定（只用 Rule.Condition + ResourceId 取值、Items 与 Condition.Items 严格下标对齐），
// 若由消费方（ai/tools）自行拼合成规则，评估器未来一旦依赖 Rule 其他字段（如按规则的
// 计数器/Duration）就会静默失真。故把合规则与下标对齐逻辑收进本模块唯一实现内，
// 评估器契约演进时只需改这里，消费方零感知。
type MetricQuery interface {
	// CurrentValues 查询指定资源的当前指标值。
	//   - metrics 为空：不取值，仅返回 Available（该资源类型全部可用指标名，供发现/探索）；
	//   - 含未知指标名：不报错，逐个在 Values 中标记 Err 并附 Available（Agent 友好容错，
	//     模型可能凭常识猜指标名，硬失败会中断整轮诊断）；
	//   - 混合请求：仅对已知指标发起评估，未知项保留 Err 标记。
	CurrentValues(ctx context.Context, resourceType int8, resourceId uint64, metrics []string) (*MetricQueryResult, error)
}

// MetricQueryResult 指标查询结果
type MetricQueryResult struct {
	// Values 每个请求指标一项（含未知指标的错误标记），与请求顺序一致
	Values []MetricValue `json:"values"`
	// Available 该资源类型全部可用指标名（metrics 为空或含未知项时填充，供自纠重试）
	Available []string `json:"available,omitempty"`
	// ResourceErr 资源级错误（资源不存在 / resourceId 缺失），非空时 Values 无效。
	// 不作为 error 返回：让 Agent 收到可读错误标记后自纠（如先 ListResources），
	// 而非中断整轮运行
	ResourceErr string `json:"resourceErr,omitempty"`
}

// MetricValue 单个指标当前值
type MetricValue struct {
	Metric string  `json:"metric"`
	Value  float64 `json:"value"`
	Unit   string  `json:"unit,omitempty"`
	// Err 非空表示该项取值失败（未知指标 / 评估器取值失败），此时 Value 无意义
	Err string `json:"err,omitempty"`
}

// GetMetricQuery 指标查询端口（无状态实现，无需 IoC 生命周期）
func GetMetricQuery() MetricQuery { return metricQueryImpl{} }

// metricKeys 提取指标定义的 key 列表
func metricKeys(defs []service.MetricDefinition) []string {
	keys := make([]string, 0, len(defs))
	for _, def := range defs {
		keys = append(keys, def.Key)
	}
	return keys
}

type metricQueryImpl struct{}

var _ MetricQuery = metricQueryImpl{}

func (metricQueryImpl) CurrentValues(ctx context.Context, resourceType int8, resourceId uint64, metrics []string) (*MetricQueryResult, error) {
	evaluator, ok := service.GetEvaluator(resourceType)
	if !ok {
		return nil, errorx.NewBizf("no metric evaluator registered for resourceType %d", resourceType)
	}
	res := &MetricQueryResult{Available: make([]string, 0, len(evaluator.Metrics()))}

	// 资源级问题（id 缺失 / 资源不存在）不硬失败：标记 ResourceErr + 附可用指标列表，
	// 模型凭上下文猜 resourceId 是常态，报错会中断整轮诊断
	if resourceId == 0 {
		res.ResourceErr = "resourceId is required, query resources first (e.g. ListResources)"
		res.Available = metricKeys(evaluator.Metrics())
		return res, nil
	}
	if !evaluator.ResourceExists(resourceId) {
		res.ResourceErr = fmt.Sprintf("resource[%d] of type %d not found, query resources first (e.g. ListResources) then retry with a valid resourceId", resourceId, resourceType)
		res.Available = metricKeys(evaluator.Metrics())
		return res, nil
	}

	defs := evaluator.Metrics()
	defsByKey := make(map[string]service.MetricDefinition, len(defs))
	available := make([]string, 0, len(defs))
	for _, def := range defs {
		defsByKey[def.Key] = def
		available = append(available, def.Key)
	}

	res.Available = available

	// metrics 为空：发现模式，仅列可用指标
	if len(metrics) == 0 {
		return res, nil
	}

	// 逐项判定已知/未知：未知项标记 Err；已知项进入合成规则待评估。
	// idxMap 记录「合成规则下标 → Values 下标」映射：Values 按请求顺序含未知项，
	// 而 Evaluate 返回的 Items 仅与合成规则对齐，直接按下标回填会错位污染相邻指标。
	items := make([]entity.ConditionItem, 0, len(metrics))
	idxMap := make([]int, 0, len(metrics))
	for _, m := range metrics {
		if def, found := defsByKey[m]; found {
			idxMap = append(idxMap, len(res.Values))
			res.Values = append(res.Values, MetricValue{Metric: m, Unit: def.Unit})
			items = append(items, entity.ConditionItem{Metric: m, Compare: "gte", Value: 0})
		} else {
			res.Values = append(res.Values, MetricValue{Metric: m, Err: "unknown metric for this resourceType"})
		}
	}
	if len(items) == 0 {
		return res, nil
	}

	// 合成评估规则：阈值 0 恒不触发，借 Evaluate 取值通道拿当前值（与告警引擎同源）
	rule := &entity.AlertRule{
		ResourceType: resourceType,
		Condition:    &entity.AlertCondition{Operator: "and", Items: items},
	}
	evalRes, err := evaluator.Evaluate(ctx, &service.EvalParam{Rule: rule, ResourceId: resourceId})
	if err != nil {
		return nil, err
	}
	written := make(map[int]bool, len(idxMap))
	writtenAny := evalRes.Evaluated
	missingReason := "evaluator returned no value for this metric"
	for i, item := range evalRes.Items {
		if i >= len(idxMap) {
			break // 防御：评估器违约返回超长 Items 时不得越界
		}
		vi := idxMap[i]
		res.Values[vi].Value = item.CurrentValue
		if item.Err != "" {
			res.Values[vi].Err = item.Err
		}
		written[vi] = true
	}
	// 缺口必须显式标错：评估器少返回条目（违约截断、内部吞错、Evaluated=false）时，
	// 未回填的项会停在「Value 0 + 空 Err + 带单位」——这个形状与「真的测到 0%」完全一致，
	// 既看不出丢了值，又会被模型当成真实读数下结论（错值以事实形态进入诊断链路）
	if !writtenAny {
		missingReason = "evaluator did not evaluate this resource"
	}
	for _, vi := range idxMap {
		if written[vi] || res.Values[vi].Err != "" {
			continue
		}
		res.Values[vi].Err = missingReason
		res.Values[vi].Value = 0
	}
	return res, nil
}

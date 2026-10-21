// Package monitor 可观测数据工具（阶段 4.2：指标查询 + 告警历史）。
//
// 以只读工具形式把指标/告警历史暴露给 Agent，补齐「可观测数据工具化」差距。
// 全部为只读查询，capability=read（声明集中于 ai/tools/capability.go），只读诊断档直接可用。
// 与 dbtool 同范式：本包 import 业务应用层（alert），由 ai/init 宿主级安装器装配；
// 指标取值的评估器契约细节收口在 alert/application.MetricQuery 窄端口内，本包仅做 DTO 转换。
package monitor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"mayfly-go/internal/ai/imsg"
	"mayfly-go/internal/ai/tools"
	alertapp "mayfly-go/internal/alert/application"
	alertentity "mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/errorx"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

const (
	// defaultAlertHistoryLimit 告警历史默认返回条数
	defaultAlertHistoryLimit = 20
	// maxAlertHistoryLimit 告警历史单次返回上限（防一次工具调用撑爆上下文）
	maxAlertHistoryLimit = 50
	// alertTimeLayout 事件时间输出格式（与其余工具口径一致）
	alertTimeLayout = "2006-01-02 15:04:05"
)

// resourceTypeCodesDesc 由 consts 真源生成「资源类型编码表」拼进工具说明。
//
// 此前类型清单手写在两个 struct tag 里且已与实际映射漂移（说明只列 6 类、映射覆盖 8 类），
// 模型按说明传未列出的类型只能靠试错发现；收口后新增资源类型只需改 consts 一处
func resourceTypeCodesDesc() string {
	parts := make([]string, 0, len(consts.ResourceTypeWithKey()))
	for _, rt := range consts.ResourceTypeWithKey() {
		key, _ := consts.ResourceTypeKey(rt)
		parts = append(parts, fmt.Sprintf("%d=%s", rt, key))
	}
	return strings.Join(parts, " ")
}

// Tools 创建全部可观测数据工具（聚合创建错误，由装配方统一处理）
func Tools() ([]tool.BaseTool, error) {
	metrics, err := GetMetricsQuery()
	if err != nil {
		return nil, fmt.Errorf("agent tool - get QueryMetrics failed: %w", err)
	}
	history, err := GetAlertHistory()
	if err != nil {
		return nil, fmt.Errorf("agent tool - get QueryAlertHistory failed: %w", err)
	}
	return []tool.BaseTool{metrics, history}, nil
}

// ---- 指标查询工具 ----

// MetricsQueryParam 指标查询入参
type MetricsQueryParam struct {
	ResourceType int8     `json:"resourceType" jsonschema_description:"资源类型编码，取值见工具说明的类型表。优先从会话上下文/前序工具输出继承，禁止凭空猜测。"`
	ResourceId   uint64   `json:"resourceId" jsonschema_description:"资源ID。取值逻辑：1. 用户本次明确指定；2. 从 ListResources/告警事件等前序输出继承；3. 若均无从 ListResources 工具查询，禁止凭空猜测。"`
	Metrics      []string `json:"metrics" jsonschema_description:"要查询的指标名数组，如 [\"cpu_rate\",\"mem_rate\",\"disk_usage\"]。传空数组时返回该资源类型的全部可用指标名。未知指标名不会报错，结果中会标记错误并附带可用指标列表。"`
}

// MetricValue 单个指标当前值
type MetricValue struct {
	Metric string  `json:"metric" jsonschema_description:"指标名"`
	Value  float64 `json:"value" jsonschema_description:"当前值"`
	Unit   string  `json:"unit,omitempty" jsonschema_description:"单位（如 %）"`
	Err    string  `json:"err,omitempty" jsonschema_description:"非空表示该指标本次取值失败及原因（如未知指标），此时 value 无意义"`
}

// MetricsQueryOutput 指标查询结果
type MetricsQueryOutput struct {
	ResourceType int8          `json:"resourceType" jsonschema_description:"资源类型"`
	ResourceId   uint64        `json:"resourceId" jsonschema_description:"资源ID"`
	ResourceErr  string        `json:"resourceErr,omitempty" jsonschema_description:"非空表示资源级错误（资源不存在/resourceId 缺失），此时 values 无效；应先经 ListResources 查询真实资源后再用有效 resourceId 重试"`
	Values       []MetricValue `json:"values" jsonschema_description:"各指标当前值（与请求顺序一致；未知指标仅含错误标记）"`
	Available    []string      `json:"available,omitempty" jsonschema_description:"该资源类型的全部可用指标名（metrics 为空或含未知指标时返回，供自纠重试）"`
}

// GetMetricsQuery 创建指标查询工具：与告警引擎同源的只读即时取值，无副作用。
// 取值/容错/下标对齐逻辑在 alert/application.MetricQuery 窄端口内，本工具仅转译入参出参。
func GetMetricsQuery() (tool.InvokableTool, error) {
	return utils.InferTool("QueryMetrics",
		"查询指定资源的当前监控指标值（CPU/内存/磁盘等，与告警引擎同源）。metrics 传空数组时返回该资源类型的全部可用指标名。未知指标名不会报错，结果中会标记错误并附带可用指标列表。资源类型编码："+resourceTypeCodesDesc()+"。仅已接入指标评估的类型可取值，其余类型会在 resourceErr 里明确报错。",
		func(ctx context.Context, param *MetricsQueryParam) (*MetricsQueryOutput, error) {
			// 授权先于取值：指标是账号级资源的运行时数据，不得跨 owner 读取任意资源
			if err := requireResourceAccess(ctx, param.ResourceType, param.ResourceId); err != nil {
				return nil, err
			}
			res, err := alertapp.GetMetricQuery().CurrentValues(ctx, param.ResourceType, param.ResourceId, param.Metrics)
			if err != nil {
				return nil, err
			}
			out := &MetricsQueryOutput{
				ResourceType: param.ResourceType,
				ResourceId:   param.ResourceId,
				ResourceErr:  res.ResourceErr,
				Available:    res.Available,
			}
			for _, v := range res.Values {
				out.Values = append(out.Values, MetricValue{Metric: v.Metric, Value: v.Value, Unit: v.Unit, Err: v.Err})
			}
			return out, nil
		})
}

// ---- 告警历史查询工具 ----

// AlertHistoryParam 告警历史查询入参
type AlertHistoryParam struct {
	ResourceType int8   `json:"resourceType" jsonschema_description:"资源类型（必填，用于界定可见面与避免跨类型 id 歧义），取值见工具说明的类型表"`
	ResourceId   uint64 `json:"resourceId,omitempty" jsonschema_description:"资源ID过滤（可选）；不传则返回该类型下当前运行身份可见的全部告警"`
	Status       int8   `json:"status,omitempty" jsonschema_description:"状态过滤（可选）：1=告警中 2=已确认 3=已恢复 4=已关闭；0=不过滤"`
	Limit        int    `json:"limit,omitempty" jsonschema_description:"返回条数上限（可选，默认 20，最大 50），按最近触发时间倒序"`
}

// AlertHistoryItem 单条告警事件摘要
type AlertHistoryItem struct {
	RuleName       string  `json:"ruleName" jsonschema_description:"告警规则名"`
	Priority       int8    `json:"priority" jsonschema_description:"优先级：0=P0紧急 1=P1重要 2=P2一般 3=P3提示"`
	Status         int8    `json:"status" jsonschema_description:"状态：1=告警中 2=已确认 3=已恢复 4=已关闭"`
	ResourceType   int8    `json:"resourceType" jsonschema_description:"资源类型"`
	ResourceId     uint64  `json:"resourceId" jsonschema_description:"资源ID"`
	ResourceName   string  `json:"resourceName" jsonschema_description:"资源名"`
	Metric         string  `json:"metric" jsonschema_description:"触发指标"`
	CurrentValue   float64 `json:"currentValue" jsonschema_description:"最近取值"`
	Threshold      string  `json:"threshold,omitempty" jsonschema_description:"阈值描述"`
	TriggerCount   int     `json:"triggerCount" jsonschema_description:"累计触发次数"`
	FirstTriggerAt string  `json:"firstTriggerAt" jsonschema_description:"首次触发时间"`
	LastTriggerAt  string  `json:"lastTriggerAt" jsonschema_description:"最近触发时间"`
}

// AlertHistoryOutput 告警历史查询结果
type AlertHistoryOutput struct {
	Total  int64              `json:"total" jsonschema_description:"符合条件的总数"`
	Events []AlertHistoryItem `json:"events" jsonschema_description:"告警事件列表（按最近触发倒序）"`
}

// GetAlertHistory 创建告警历史查询工具：查询告警事件（诊断「为什么触发告警」「历史是否复发」
// 的证据源）。只读，无副作用。
func GetAlertHistory() (tool.InvokableTool, error) {
	return utils.InferTool("QueryAlertHistory",
		"查询告警事件历史（按资源/状态过滤，最近触发倒序），用于根因诊断与复发分析。必须指定资源类型，结果仅含当前运行身份可见的资源。资源类型编码："+resourceTypeCodesDesc()+"。",
		func(ctx context.Context, param *AlertHistoryParam) (*AlertHistoryOutput, error) {
			limit := param.Limit
			if limit <= 0 {
				limit = defaultAlertHistoryLimit
			}
			if limit > maxAlertHistoryLimit {
				limit = maxAlertHistoryLimit
			}
			// 授权面：必须显式声明资源类型才能安全收窄——告警事件的 resource_id 在不同类型下会重复，
			// 只按 id 集合过滤会跨类型误放行，所以不提供「全类型一次查完」
			if param.ResourceType == 0 {
				return nil, tools.NewToolError(errors.New("resourceType is required to scope alert history"), tools.RecoverRetry)
			}
			typeKey, visibleIds, err := resolveVisibleType(ctx, param.ResourceType)
			if err != nil {
				return nil, err
			}
			cond := &alertentity.AlertEventQuery{ResourceType: param.ResourceType}
			if param.ResourceId > 0 {
				if !visibleIds[param.ResourceId] {
					return nil, tools.NewToolError(
						fmt.Errorf("resource %s:%d is not accessible by the run owner", typeKey, param.ResourceId), tools.RecoverNone)
				}
				cond.ResourceId = param.ResourceId
			} else {
				if len(visibleIds) == 0 {
					// 该类型下无可见资源：返回空结果。不能退化成「不加过滤」，那等于放开全库告警
					return &AlertHistoryOutput{}, nil
				}
				cond.ResourceIds = sortedIds(visibleIds)
			}
			if param.Status > 0 {
				cond.Status = alertentity.AlertEventStatus(param.Status)
			}
			cond.PageNum, cond.PageSize = 1, limit

			eventApp := alertapp.GetAlertEventApp()
			if eventApp == nil {
				return nil, errorx.NewBizI(ctx, imsg.ErrAlertEventAppNotReady)
			}
			page, err := eventApp.GetAlertEventList(cond, "last_trigger_time desc")
			if err != nil {
				return nil, err
			}
			out := &AlertHistoryOutput{Total: page.Total}
			for _, ev := range page.List {
				out.Events = append(out.Events, AlertHistoryItem{
					RuleName:       ev.RuleName,
					Priority:       int8(ev.Priority),
					Status:         int8(ev.Status),
					ResourceType:   ev.ResourceType,
					ResourceId:     ev.ResourceId,
					ResourceName:   ev.ResourceName,
					Metric:         ev.Metric,
					CurrentValue:   ev.CurrentValue,
					Threshold:      ev.Threshold,
					TriggerCount:   ev.TriggerCount,
					FirstTriggerAt: ev.FirstTriggerTime.Format(alertTimeLayout),
					LastTriggerAt:  ev.LastTriggerTime.Format(alertTimeLayout),
				})
			}
			return out, nil
		})
}

// sortedIds 把可见资源集合转为升序切片（固定顺序，使同一身份的多次调用输出可比较）
func sortedIds(ids map[uint64]bool) []uint64 {
	out := make([]uint64, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

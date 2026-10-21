package trigger

import (
	"context"
	"encoding/json"
	"strconv"

	"mayfly-go/internal/agentops/imsg"
	"mayfly-go/internal/pkg/event"
	"mayfly-go/pkg/errorx"
)

// WebhookSource 外部监控 webhook 触发源（Prometheus/Zabbix/自定义）。
//
// 把原始 JSON body（map[string]any）按宽松约定归一化为中立事件：识别字段取用、
// 未识别字段整体留在 Payload 作为诊断上下文；数值统一按 JSON 解码后的形态（float64 /
// json.Number / 数字字符串）容错解析，缺字段留零值，不因外部系统书写差异而拒绝。
type WebhookSource struct{}

var _ TriggerSource = (*WebhookSource)(nil)

// NewWebhookSource 创建 webhook 触发源
func NewWebhookSource() *WebhookSource { return &WebhookSource{} }

func (s *WebhookSource) Type() string { return SourceTypeWebhook }

// Normalize 原始 JSON 对象 → 中立事件
func (s *WebhookSource) Normalize(ctx context.Context, raw any) (*event.OpsTriggerEvent, error) {
	m, ok := raw.(map[string]any)
	if !ok || m == nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrWebhookPayloadNotObject)
	}
	return &event.OpsTriggerEvent{
		Source:       SourceTypeWebhook,
		SourceRef:    jsonStr(m, "sourceRef"),
		ResourceType: jsonStr(m, "resourceType"),
		ResourceId:   jsonUint64(m, "resourceId"),
		ResourceName: jsonStr(m, "resourceName"),
		Severity:     jsonStr(m, "severity"),
		Metric:       jsonStr(m, "metric"),
		Value:        jsonFloat64(m, "value"),
		OccurredAt:   jsonInt64(m, "occurredAt"),
		Payload:      m,
	}, nil
}

func (s *WebhookSource) Match(evt *event.OpsTriggerEvent, cond *MatchCondition) bool {
	return MatchByFields(evt, cond)
}

// InstallWebhookSource 向注册表注册 webhook 触发源
func InstallWebhookSource(r *Registry) { r.Register(NewWebhookSource()) }

// ── JSON 松散取值助手：兼容 float64 / json.Number / 字符串三种解码形态 ──

func jsonStr(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func jsonFloat64(m map[string]any, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case json.Number:
		f, _ := v.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0
	}
}

func jsonInt64(m map[string]any, key string) int64 {
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	default:
		return 0
	}
}

func jsonUint64(m map[string]any, key string) uint64 {
	n := jsonInt64(m, key)
	if n < 0 {
		return 0
	}
	return uint64(n)
}

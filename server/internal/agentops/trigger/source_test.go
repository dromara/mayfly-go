package trigger

import (
	"encoding/json"
	"testing"

	"mayfly-go/internal/pkg/event"

	"context"
	"github.com/stretchr/testify/require"
)

// --- MatchByFields：通用字段白名单匹配 ---

func TestMatchByFields(t *testing.T) {
	evt := &event.OpsTriggerEvent{
		Source:       SourceTypeAlert,
		ResourceType: "machine",
		ResourceId:   7,
		Severity:     "P0",
		Metric:       "cpu_usage",
	}

	// nil 事件或 nil 条件 → 全命中（触发器可不设条件）
	require.True(t, MatchByFields(nil, nil))
	require.True(t, MatchByFields(evt, nil), "nil 条件视为无条件全命中")
	require.True(t, MatchByFields(evt, &MatchCondition{}), "零值条件视为无条件全命中")

	// 资源类型：大小写不敏感命中 / 不命中
	require.True(t, MatchByFields(evt, &MatchCondition{ResourceType: "MACHINE"}))
	require.False(t, MatchByFields(evt, &MatchCondition{ResourceType: "db"}))

	// 资源 id 白名单
	require.True(t, MatchByFields(evt, &MatchCondition{ResourceIds: []uint64{1, 7, 9}}))
	require.False(t, MatchByFields(evt, &MatchCondition{ResourceIds: []uint64{1, 9}}))

	// 严重级白名单：大小写不敏感
	require.True(t, MatchByFields(evt, &MatchCondition{Severities: []string{"p0", "P1"}}))
	require.False(t, MatchByFields(evt, &MatchCondition{Severities: []string{"P2", "P3"}}))

	// 指标白名单
	require.True(t, MatchByFields(evt, &MatchCondition{Metrics: []string{"cpu_usage"}}))
	require.False(t, MatchByFields(evt, &MatchCondition{Metrics: []string{"mem_usage"}}))

	// 多维度 AND：全部命中才 true
	require.True(t, MatchByFields(evt, &MatchCondition{ResourceType: "machine", Severities: []string{"P0"}, Metrics: []string{"cpu_usage"}}))
	require.False(t, MatchByFields(evt, &MatchCondition{ResourceType: "machine", Severities: []string{"P3"}}), "任一维度不命中即 false")
}

// --- AlertSource：透传校验 + Source 补全 ---

func TestAlertSourceNormalize(t *testing.T) {
	s := NewAlertSource()
	require.Equal(t, SourceTypeAlert, s.Type())

	// 已是中立事件 → 透传，Source 为空时补全
	in := &event.OpsTriggerEvent{SourceRef: "42", Severity: "P0"}
	out, err := s.Normalize(context.Background(), in)
	require.NoError(t, err)
	require.Same(t, in, out, "alert 发布侧已归一化，此处透传同一实例")
	require.Equal(t, SourceTypeAlert, out.Source, "Source 为空须补全为 alert")

	// 非中立事件载荷 → 拒绝
	_, err = s.Normalize(context.Background(), map[string]any{"x": 1})
	require.Error(t, err, "alert 源只接受 *OpsTriggerEvent")
	_, err = s.Normalize(context.Background(), nil)
	require.Error(t, err)
}

// --- WebhookSource：原始 JSON 归一化（容错 float64 / json.Number / 字符串） ---

func TestWebhookSourceNormalize(t *testing.T) {
	s := NewWebhookSource()
	require.Equal(t, SourceTypeWebhook, s.Type())

	raw := map[string]any{
		"sourceRef":    "ext-1",
		"resourceType": "db",
		"resourceId":   float64(12), // encoding/json 默认把数字解成 float64
		"resourceName": "tokhub",
		"severity":     "P1",
		"metric":       "slow_query",
		"value":        json.Number("3.5"), // 兼容 json.Number
		"occurredAt":   "1700000000",       // 兼容数字字符串
		"extra":        "kept-in-payload",
	}
	evt, err := s.Normalize(context.Background(), raw)
	require.NoError(t, err)
	require.Equal(t, SourceTypeWebhook, evt.Source)
	require.Equal(t, "ext-1", evt.SourceRef)
	require.Equal(t, "db", evt.ResourceType)
	require.Equal(t, uint64(12), evt.ResourceId)
	require.Equal(t, "tokhub", evt.ResourceName)
	require.Equal(t, "P1", evt.Severity)
	require.Equal(t, "slow_query", evt.Metric)
	require.InDelta(t, 3.5, evt.Value, 1e-9)
	require.Equal(t, int64(1700000000), evt.OccurredAt)
	require.Equal(t, "kept-in-payload", evt.Payload["extra"], "未识别字段整体留在 Payload")

	// 非 JSON 对象 → 拒绝
	_, err = s.Normalize(context.Background(), "not-an-object")
	require.Error(t, err)
	_, err = s.Normalize(context.Background(), nil)
	require.Error(t, err)
}

// --- Registry：注册 / 取用 / 覆盖 / 脏注册防护 ---

func TestRegistry(t *testing.T) {
	r := NewRegistry()

	// nil 与空 Type 的脏注册被忽略
	r.Register(nil)
	r.Register(&fakeSource{typ: ""})
	require.Empty(t, r.Types())

	r.Register(NewAlertSource())
	r.Register(NewWebhookSource())
	require.ElementsMatch(t, []string{SourceTypeAlert, SourceTypeWebhook}, r.Types())

	got, ok := r.Get(SourceTypeAlert)
	require.True(t, ok)
	require.Equal(t, SourceTypeAlert, got.Type())

	_, ok = r.Get("unknown")
	require.False(t, ok)

	// 同类型后注册覆盖先注册
	r.Register(&fakeSource{typ: SourceTypeAlert})
	got, _ = r.Get(SourceTypeAlert)
	_, isFake := got.(*fakeSource)
	require.True(t, isFake, "同类型后注册应覆盖先注册")
}

// --- 内置来源安装到默认注册表 ---

func TestInstallBuiltins(t *testing.T) {
	r := NewRegistry()
	InstallAlertSource(r)
	InstallWebhookSource(r)
	_, ok := r.Get(SourceTypeAlert)
	require.True(t, ok)
	_, ok = r.Get(SourceTypeWebhook)
	require.True(t, ok)
}

// --- OpsTriggerEvent.DedupKey ---

func TestDedupKey(t *testing.T) {
	evt := &event.OpsTriggerEvent{Source: "alert", SourceRef: "42", OccurredAt: 1700000000}
	require.Equal(t, "alert:42:1700000000", evt.DedupKey())

	// OccurredAt 前移（再次越限）→ 不同去重键（视为新事件）
	evt2 := &event.OpsTriggerEvent{Source: "alert", SourceRef: "42", OccurredAt: 1700000999}
	require.NotEqual(t, evt.DedupKey(), evt2.DedupKey())
}

// fakeSource 可编程触发源，用于注册表覆盖/脏注册测试
type fakeSource struct {
	typ       string
	normalize func(any) (*event.OpsTriggerEvent, error)
}

func (f *fakeSource) Type() string { return f.typ }
func (f *fakeSource) Normalize(_ context.Context, raw any) (*event.OpsTriggerEvent, error) {
	if f.normalize != nil {
		return f.normalize(raw)
	}
	return &event.OpsTriggerEvent{Source: f.typ}, nil
}
func (f *fakeSource) Match(evt *event.OpsTriggerEvent, cond *MatchCondition) bool {
	return MatchByFields(evt, cond)
}

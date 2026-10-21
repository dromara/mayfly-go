package event

import "strconv"

// 触发源类型标识（中立契约的一部分：发布方与订阅方共用同一字面量，避免魔法字符串两侧漂移）。
// agentops 的 trigger.SourceType* 即别名到这些常量，alert 发布侧直接引用，二者天然对齐。
const (
	OpsTriggerSourceAlert   = "alert"   // 告警联动（alert 引擎通知成功后发布）
	OpsTriggerSourceWebhook = "webhook" // 外部监控 webhook 推送
)

// OpsTriggerEvent 中立运维触发事件（事件驱动 Agent 运行的统一入站契约）。
//
// 由事件源（告警引擎、webhook 等）在各自边界内归一化后，经 global.EventBus 以
// EventTopicOpsTrigger 发布；agentops 订阅后匹配触发器 → 派生只读诊断 Run。
//
// 刻意做成「中立」结构以维持依赖不变式：发布方（alert）不依赖 ai/agentops，
// 订阅方（agentops）不依赖 alert，二者仅经此契约 + EventBus 通信（无环）。
// 字段取「运维触发」的最大公约数，来源特有信息一律进 Payload，避免契约随来源膨胀。
type OpsTriggerEvent struct {
	// Source 触发源类型（alert/webhook/...；与 agentops TriggerSource.Type 对齐）
	Source string `json:"source"`
	// SourceRef 来源唯一引用（alert=事件id；用于幂等去重与结论回写定位）
	SourceRef string `json:"sourceRef"`
	// ResourceType 资源类型（machine/db/...；空=不限定资源类型）
	ResourceType string `json:"resourceType"`
	// ResourceId 资源 id（0=未指定）
	ResourceId uint64 `json:"resourceId"`
	// ResourceName 资源名（展示/诊断上下文）
	ResourceName string `json:"resourceName"`
	// Severity 归一化严重级（如 P0..P3；空=未知）
	Severity string `json:"severity"`
	// Metric 触发指标名（空=非指标类触发）
	Metric string `json:"metric"`
	// Value 指标当前值
	Value float64 `json:"value"`
	// OccurredAt 事件发生时间（unix 秒；用于幂等键细分，使同一来源的不同次触发各自成 Run）
	OccurredAt int64 `json:"occurredAt"`
	// Payload 来源原始/快照载荷（webhook 自由字段 / alert 详情快照），不参与核心匹配
	Payload map[string]any `json:"payload"`
}

// DedupKey 事件去重键：同源 + 同引用 + 同发生时刻视为同一事件。
//
// agentops 以 "event:" + DedupKey() 作为 Run 幂等键，阻断同一事件的重复投递
// （EventBus 重投、多分组重复通知等）派生出重复诊断；OccurredAt 前移（如告警再次越限）
// 则视为新事件、可再次派生（仍受触发器冷却约束）。
func (e *OpsTriggerEvent) DedupKey() string {
	return e.Source + ":" + e.SourceRef + ":" + strconv.FormatInt(e.OccurredAt, 10)
}

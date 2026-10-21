package application

import (
	"context"
	"sync"
	"testing"
	"time"

	"mayfly-go/internal/alert/domain/entity"
	pkgconsts "mayfly-go/internal/pkg/consts"
	opsevent "mayfly-go/internal/pkg/event"
	"mayfly-go/pkg/eventbus"
	"mayfly-go/pkg/global"

	"github.com/stretchr/testify/require"
)

// TestOpsTriggerResourceType 校验告警资源类型枚举 → 中立事件资源类型字符串的映射。
// 覆盖到的类型映射为稳定语义名；未覆盖的类型回退为其数值字符串（信息不丢失，供触发器匹配/诊断上下文）。
func TestOpsTriggerResourceType(t *testing.T) {
	require.Equal(t, "machine", opsTriggerResourceType(pkgconsts.ResourceTypeMachine))
	require.Equal(t, "db", opsTriggerResourceType(pkgconsts.ResourceTypeDbInstance))
	require.Equal(t, "redis", opsTriggerResourceType(pkgconsts.ResourceTypeRedis))
	require.Equal(t, "mongo", opsTriggerResourceType(pkgconsts.ResourceTypeMongo))
	require.Equal(t, "es", opsTriggerResourceType(pkgconsts.ResourceTypeEsInstance))
	require.Equal(t, "container", opsTriggerResourceType(pkgconsts.ResourceTypeContainer))
	require.Equal(t, "kafka", opsTriggerResourceType(pkgconsts.ResourceTypeMqKafka))
	require.Equal(t, "milvus", opsTriggerResourceType(pkgconsts.ResourceTypeMilvus))
	require.Equal(t, "99", opsTriggerResourceType(99), "未覆盖类型须回退为数值字符串")
}

// TestPublishOpsTriggerFailOpen 验证告警事件 → 中立 OpsTriggerEvent 的归一化与发布：
//  1. fail-open：无订阅者时发布不得 panic，也不得回流影响告警通知主链路；
//  2. 归一化正确：订阅后经 Publish + WaitAsync 收到的事件字段须与告警事件一一对应。
//
// publishOpsTrigger 仅依赖入参事件 + global.EventBus + logx，不触碰 alertEngineAppImpl 的其他字段，
// 故可在零值实例上直接调用。
func TestPublishOpsTriggerFailOpen(t *testing.T) {
	a := &alertEngineAppImpl{}

	ev := &entity.AlertEvent{
		Id:              42,
		RuleId:          5,
		RuleName:        "机器CPU告警",
		ResourceType:    pkgconsts.ResourceTypeMachine,
		ResourceId:      3,
		ResourceName:    "web-1",
		Priority:        entity.AlertPriorityCritical,
		Metric:          "cpu_usage",
		CurrentValue:    99.5,
		Threshold:       ">90",
		Labels:          `{"app":"web"}`,
		Status:          entity.AlertEventStatusFiring,
		TriggerCount:    2,
		LastTriggerTime: time.Unix(1700000000, 0),
	}

	// 无订阅者：发布不得 panic（fail-open，Publish 本身无返回值）
	require.NotPanics(t, func() { a.publishOpsTrigger(ev) }, "无订阅者时发布必须静默成功，不得影响告警主链路")

	// 订阅后：验证归一化字段与 OccurredAt 取自 LastTriggerTime
	var mu sync.Mutex
	var got *opsevent.OpsTriggerEvent
	const subId = "test-ops-trigger-publish"
	require.NoError(t, global.EventBus.SubscribeAsync(opsevent.EventTopicOpsTrigger, subId,
		func(ctx context.Context, e *eventbus.Event[any]) error {
			evt, ok := e.Val.(*opsevent.OpsTriggerEvent)
			if !ok {
				return nil
			}
			mu.Lock()
			got = evt
			mu.Unlock()
			return nil
		}, false))
	t.Cleanup(func() { _ = global.EventBus.Unsubscribe(opsevent.EventTopicOpsTrigger, subId) })

	a.publishOpsTrigger(ev)
	global.EventBus.WaitAsync()

	mu.Lock()
	defer mu.Unlock()
	require.NotNil(t, got, "订阅者应收到发布的运维触发事件")
	require.Equal(t, opsevent.OpsTriggerSourceAlert, got.Source)
	require.Equal(t, "42", got.SourceRef, "SourceRef 取告警事件 id（用于幂等/结论回写定位）")
	require.Equal(t, "machine", got.ResourceType)
	require.Equal(t, uint64(3), got.ResourceId)
	require.Equal(t, "web-1", got.ResourceName)
	require.Equal(t, "P0", got.Severity, "Severity 取优先级 Code()")
	require.Equal(t, "cpu_usage", got.Metric)
	require.InDelta(t, 99.5, got.Value, 1e-9)
	require.Equal(t, int64(1700000000), got.OccurredAt, "OccurredAt 取 LastTriggerTime 的 unix 秒")
	// Payload 保留告警详情快照
	require.Equal(t, uint64(5), got.Payload["ruleId"])
	require.Equal(t, "机器CPU告警", got.Payload["ruleName"])
	require.Equal(t, ">90", got.Payload["threshold"])
	require.Equal(t, `{"app":"web"}`, got.Payload["labels"])
	require.Equal(t, int8(entity.AlertEventStatusFiring), got.Payload["status"])
	require.Equal(t, 2, got.Payload["triggerCount"])
}

// TestPublishOpsTriggerZeroLastTriggerTime 验证 LastTriggerTime 为零值时 OccurredAt 回退为当前时间（非 0），
// 保证幂等去重键的时间维度始终有效。
func TestPublishOpsTriggerZeroLastTriggerTime(t *testing.T) {
	a := &alertEngineAppImpl{}

	ev := &entity.AlertEvent{
		Id:           7,
		ResourceType: pkgconsts.ResourceTypeDbInstance,
		ResourceId:   1,
		Priority:     entity.AlertPriorityHigh,
		// LastTriggerTime 留零值
	}

	var mu sync.Mutex
	var got *opsevent.OpsTriggerEvent
	const subId = "test-ops-trigger-zero-time"
	require.NoError(t, global.EventBus.SubscribeAsync(opsevent.EventTopicOpsTrigger, subId,
		func(ctx context.Context, e *eventbus.Event[any]) error {
			evt, ok := e.Val.(*opsevent.OpsTriggerEvent)
			if !ok {
				return nil
			}
			mu.Lock()
			got = evt
			mu.Unlock()
			return nil
		}, false))
	t.Cleanup(func() { _ = global.EventBus.Unsubscribe(opsevent.EventTopicOpsTrigger, subId) })

	before := time.Now().Unix()
	a.publishOpsTrigger(ev)
	global.EventBus.WaitAsync()
	after := time.Now().Unix()

	mu.Lock()
	defer mu.Unlock()
	require.NotNil(t, got)
	require.Equal(t, "db", got.ResourceType)
	require.Equal(t, "P1", got.Severity)
	require.GreaterOrEqual(t, got.OccurredAt, before, "零值 LastTriggerTime 须回退为当前时间")
	require.LessOrEqual(t, got.OccurredAt, after)
}

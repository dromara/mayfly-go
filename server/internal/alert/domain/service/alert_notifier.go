package service

import (
	"context"
	"mayfly-go/internal/alert/domain/entity"
)

// NotifyTarget 通知目标：渠道ID + 接收人ID
type NotifyTarget struct {
	ChannelIds  []uint64
	ReceiverIds []int64
}

// AlertNotifier 通知编排接口 —— 隔离通知实现细节。
// 通知路由由独立的通知策略（AlertNotifyPolicy）负责，调用方通过 MatchPolicies 获取匹配策略后
// 合并渠道/接收人构造 NotifyTarget，再调用 NotifyWithTarget / NotifyRecoverWithTarget 发送。
type AlertNotifier interface {
	// NotifyWithTarget 使用指定通知目标发送告警。
	// 返回已投递的渠道数量：为 0 表示无可用渠道、实际未发出任何通知，
	// 调用方必须据此避免虚增事件的通知次数。
	NotifyWithTarget(ctx context.Context, event *entity.AlertEvent, rule *entity.AlertRule, target NotifyTarget) (int, error)

	// NotifyRecoverWithTarget 使用指定通知目标发送恢复通知。
	// 确保恢复通知与触发通知走相同目标，避免"告警发给 A 团队、恢复发给 B 团队"。
	NotifyRecoverWithTarget(ctx context.Context, event *entity.AlertEvent, rule *entity.AlertRule, target NotifyTarget) (int, error)
}

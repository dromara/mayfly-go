package service

import (
	"context"
	"mayfly-go/internal/alert/domain/entity"
)

// NotifyTarget 通知目标：渠道ID + 接收人ID
type NotifyTarget struct {
	ChannelIds  []uint64
	ReceiverIds []int64

	// PolicyByChannel 渠道ID → 来源策略ID（通知策略或升级策略）。
	// 通知日志按渠道归因到具体策略；多策略合并命中同一渠道时，
	// 归因到首个声明该渠道的策略。为 nil 时日志记录策略ID 0
	PolicyByChannel map[uint64]uint64
}

// PolicyOfChannel 渠道归因：未登记的渠道返回 0（无策略来源）
func (t NotifyTarget) PolicyOfChannel(channelId uint64) uint64 {
	if t.PolicyByChannel == nil {
		return 0
	}
	return t.PolicyByChannel[channelId]
}

// WithPolicyBinding 为渠道登记来源策略（懒初始化 map，链式调用）
func (t NotifyTarget) WithPolicyBinding(policyId uint64, channelIds []uint64) NotifyTarget {
	if t.PolicyByChannel == nil {
		t.PolicyByChannel = make(map[uint64]uint64, len(channelIds))
	}
	for _, ch := range channelIds {
		// 多策略命中同一渠道时保留首个声明者，与合并去重的保留顺序一致
		if _, ok := t.PolicyByChannel[ch]; !ok {
			t.PolicyByChannel[ch] = policyId
		}
	}
	return t
}

// AlertNotifier 通知编排接口 —— 隔离通知实现细节。
// 通知路由由独立的通知策略（AlertNotifyPolicy）负责，调用方匹配策略后
// 合并渠道/接收人构造 NotifyTarget（含渠道→策略归因），再调用 NotifyWithTarget / NotifyRecoverWithTarget 发送。
type AlertNotifier interface {
	// NotifyWithTarget 使用指定通知目标发送告警。
	// 返回已投递的渠道数量：为 0 表示无可用渠道、实际未发出任何通知，
	// 调用方必须据此避免虚增事件的通知次数。
	NotifyWithTarget(ctx context.Context, event *entity.AlertEvent, rule *entity.AlertRule, target NotifyTarget) (int, error)

	// NotifyRecoverWithTarget 使用指定通知目标发送恢复通知。
	// 确保恢复通知与触发通知走相同目标，避免"告警发给 A 团队、恢复发给 B 团队"。
	NotifyRecoverWithTarget(ctx context.Context, event *entity.AlertEvent, rule *entity.AlertRule, target NotifyTarget) (int, error)
}

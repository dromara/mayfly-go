package service

import (
	"context"
	"mayfly-go/internal/alert/domain/entity"
)

// PolicyRef 通知渠道的来源策略：来源类型 + 策略 ID 必须成对出现。
//
// 只有 ID 不够（真机实测）：通知策略与升级策略各有独立的主键空间且已实际重叠，
// 日志里的 policy_id=11 曾在通知策略表查无此行——它是升级策略 11。缺了类型，
// 「这条通知为什么发给了这个渠道」既无从溯源，还会把排查引到错误的路由分支上
type PolicyRef struct {
	Type entity.NotifyPolicyType
	Id   uint64
}

// NotifyTarget 通知目标：渠道ID + 接收人ID
type NotifyTarget struct {
	ChannelIds  []uint64
	ReceiverIds []int64

	// PolicyByChannel 渠道ID → 来源策略（含类型）。
	// 通知日志按渠道归因到具体策略；多策略合并命中同一渠道时，
	// 归因到首个声明该渠道的策略。为 nil 时日志记为「来源未知」
	PolicyByChannel map[uint64]PolicyRef
}

// PolicyOfChannel 渠道归因：未登记的渠道返回零值（policy_type=0 来源未知），
// 而不是猜一个类型——归因宁缺勿假
func (t NotifyTarget) PolicyOfChannel(channelId uint64) PolicyRef {
	if t.PolicyByChannel == nil {
		return PolicyRef{}
	}
	return t.PolicyByChannel[channelId]
}

// WithNotifyPolicyBinding 登记「这些渠道由通知策略 policyId 路由而来」
func (t NotifyTarget) WithNotifyPolicyBinding(policyId uint64, channelIds []uint64) NotifyTarget {
	return t.withPolicyBinding(entity.NotifyPolicyTypeNotify, policyId, channelIds)
}

// WithEscalationBinding 登记「这些渠道由升级策略 policyId 推送而来」
func (t NotifyTarget) WithEscalationBinding(policyId uint64, channelIds []uint64) NotifyTarget {
	return t.withPolicyBinding(entity.NotifyPolicyTypeEscalation, policyId, channelIds)
}

// withPolicyBinding 为渠道登记来源策略（懒初始化 map，链式调用）。
//
// 不开放为导出方法、而是按来源类型拆成两个具名入口：类型与 ID 必须匹配才有溯源价值，
// 把类型做成参数就只能靠调用方自觉（写错一个常量就重现本缺陷）；拆成两个方法后，
// 选哪个方法就等于声明了走的是哪条路由分支，错配无法通过编译
func (t NotifyTarget) withPolicyBinding(policyType entity.NotifyPolicyType, policyId uint64, channelIds []uint64) NotifyTarget {
	// policyId 为 0 时不登记：0 不可能是任何一张策略表的有效主键（单测里现造的策略常未赋 Id），
	// 记上去只会写出一条「来源=升级策略、id=0」的假日志，比留空更难排查
	if policyId == 0 {
		return t
	}
	if t.PolicyByChannel == nil {
		t.PolicyByChannel = make(map[uint64]PolicyRef, len(channelIds))
	}
	for _, ch := range channelIds {
		// 多策略命中同一渠道时保留首个声明者，与合并去重的保留顺序一致
		if _, ok := t.PolicyByChannel[ch]; !ok {
			t.PolicyByChannel[ch] = PolicyRef{Type: policyType, Id: policyId}
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

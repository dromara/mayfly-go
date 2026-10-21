package service

import (
	"testing"

	"mayfly-go/internal/alert/domain/entity"

	"github.com/stretchr/testify/assert"
)

// 本文件钉住通知日志的「渠道 → 来源策略」归因契约。
//
// 归因是纯取证用途（无 API、无界面引用），写错不会有任何报错，只会在某天排查
// 「这条通知为什么发给了这个渠道」时把人引到错误的策略表上——真机实测已经发生过一次：
// 日志里的 policy_id=11 在 t_alert_notify_policy 查无此行，它是 t_alert_escalation.id=11，
// 排查方向因此从「升级策略无标签绑定 → 匹配全部同类规则」被带偏到「通知策略标签匹配」。
// 所以这里必须逐条钉住「哪个分支写哪个类型」，只断言 id 相等是拦不住本缺陷的

// TestPolicyAttributionCarriesSourceType 两张策略表主键空间独立且会重叠：同一个 id 11
// 经不同路由分支登记后，类型必须能区分开。这是本契约里唯一不可替代的断言
func TestPolicyAttributionCarriesSourceType(t *testing.T) {
	target := NotifyTarget{ChannelIds: []uint64{4, 1}}.
		WithNotifyPolicyBinding(11, []uint64{4}).
		WithEscalationBinding(11, []uint64{1})

	assert.Equal(t, PolicyRef{Type: entity.NotifyPolicyTypeNotify, Id: 11}, target.PolicyOfChannel(4),
		"通知策略路由的渠道必须归因到 t_alert_notify_policy")
	assert.Equal(t, PolicyRef{Type: entity.NotifyPolicyTypeEscalation, Id: 11}, target.PolicyOfChannel(1),
		"升级策略推送的渠道必须归因到 t_alert_escalation；只写 id 时这两条日志完全无法区分")
}

// TestPolicyAttributionFirstDeclarerWins 多策略命中同一渠道时归因到首个声明者，
// 与渠道合并去重的保留顺序一致——否则日志里记的策略并不是真正把该渠道带进来的那一个。
// 跨类型也适用：后到的升级策略不得覆盖已有的通知策略归因
func TestPolicyAttributionFirstDeclarerWins(t *testing.T) {
	target := NotifyTarget{}.
		WithNotifyPolicyBinding(2, []uint64{4}).
		WithNotifyPolicyBinding(7, []uint64{4}).
		WithEscalationBinding(9, []uint64{4})

	assert.Equal(t, PolicyRef{Type: entity.NotifyPolicyTypeNotify, Id: 2}, target.PolicyOfChannel(4))
}

// TestPolicyAttributionUnknownWhenUnregistered 未登记归因的渠道必须落「来源未知」（零值），
// 而不是猜一个类型：猜错的归因比没有归因更坏，它会把排查引到一张根本不相关的表上
func TestPolicyAttributionUnknownWhenUnregistered(t *testing.T) {
	assert.Equal(t, PolicyRef{}, NotifyTarget{}.PolicyOfChannel(4), "PolicyByChannel 为 nil 时")
	assert.Equal(t, PolicyRef{}, NotifyTarget{PolicyByChannel: map[uint64]PolicyRef{}}.PolicyOfChannel(4),
		"map 已初始化但该渠道未登记时")
	assert.Equal(t, entity.NotifyPolicyTypeUnknown, NotifyTarget{}.PolicyOfChannel(4).Type)
}

// TestPolicyAttributionSkipsZeroPolicyId 策略 id 为 0 时不登记。
//
// 0 不可能是任何一张策略表的有效主键（单测里现造的策略常未赋 Id），记上去会写出一条
// 「来源=升级策略、id=0」的假日志——它声称了一个不存在的策略行，比留空更难排查
func TestPolicyAttributionSkipsZeroPolicyId(t *testing.T) {
	target := NotifyTarget{ChannelIds: []uint64{4}}.WithEscalationBinding(0, []uint64{4})

	assert.Nil(t, target.PolicyByChannel, "id 为 0 时连 map 都不该建")
	assert.Equal(t, PolicyRef{}, target.PolicyOfChannel(4))
}

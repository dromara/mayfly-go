package application

import (
	"context"
	"testing"

	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 本文件覆盖「通知策略路由」这条主路径的渠道归因。
//
// 既有的 mockNotifyPolicyApp.ListEnabledWithLabels 恒返回空列表，于是 resolveNotifyTarget
// 的策略遍历分支在单测里从未被执行过——归因写错来源类型也不会有任何用例变红。
// 而这一列在真机上已经误导过一次排查（升级策略 id 被当成通知策略 id 去查，查无此行）

// policyReturningApp 可配置策略列表的通知策略服务替身
type policyReturningApp struct {
	mockNotifyPolicyApp

	policies []*entity.AlertNotifyPolicy
}

func (m *policyReturningApp) ListEnabledWithLabels(context.Context) ([]*entity.AlertNotifyPolicy, error) {
	return m.policies, nil
}

// notifyPolicy 造一条启用中的通知策略；matchLabels 为 JSON 串（与 t_label_binding 回填后的形态一致）
func notifyPolicy(id uint64, matchLabels string, channelIds ...uint64) *entity.AlertNotifyPolicy {
	p := &entity.AlertNotifyPolicy{
		Name:        "策略",
		Status:      entity.AlertNotifyPolicyStatusEnable,
		MatchLabels: matchLabels,
		ChannelIds:  channelIds,
	}
	p.Id = id
	return p
}

// engineWithPolicies 只装配 resolveNotifyTarget 真正触达的依赖（策略服务 + 其缓存锁的零值）
func engineWithPolicies(policies ...*entity.AlertNotifyPolicy) *alertEngineAppImpl {
	return &alertEngineAppImpl{notifyPolicyApp: &policyReturningApp{policies: policies}}
}

// devEvent 带两个标签的事件：既能被 notify=dev 命中，也能被 env=test 命中
func devEvent() *entity.AlertEvent {
	return &entity.AlertEvent{Labels: `{"notify":"dev","env":"test"}`}
}

// TestResolveNotifyTargetAttributesToNotifyPolicy 通知策略路由出的渠道，归因类型必须是
// 「通知策略」。升级链路写的是同一列 policy_id，类型是唯一的判别依据
func TestResolveNotifyTargetAttributesToNotifyPolicy(t *testing.T) {
	engine := engineWithPolicies(notifyPolicy(2, `{"notify":"dev"}`, 4))

	target := engine.resolveNotifyTarget(&entity.AlertRule{}, devEvent())

	require.Equal(t, []uint64{4}, target.ChannelIds)
	assert.Equal(t, service.PolicyRef{Type: entity.NotifyPolicyTypeNotify, Id: 2}, target.PolicyOfChannel(4),
		"归因类型写错会让日志里的 policy_id 指向另一张表")
}

// TestResolveNotifyTargetMergesAttributionAcrossPolicies 多策略命中时渠道去重合并，
// 归因各自独立，同一渠道保留首个声明者。
//
// listPoliciesWithLabels 已按 Id 升序固定遍历顺序，所以「首个声明者」是 Id 更小的那个——
// 不依赖 SQL 未定义的返回行序，否则日志归因会在两次查询之间跳变
func TestResolveNotifyTargetMergesAttributionAcrossPolicies(t *testing.T) {
	// 刻意逆序传入，验证排序由 listPoliciesWithLabels 负责而不是靠调用方喂对顺序
	engine := engineWithPolicies(
		notifyPolicy(7, `{"env":"test"}`, 4, 5),
		notifyPolicy(2, `{"notify":"dev"}`, 4, 6),
	)

	target := engine.resolveNotifyTarget(&entity.AlertRule{}, devEvent())

	assert.Equal(t, []uint64{4, 6, 5}, target.ChannelIds, "按策略 Id 升序遍历、渠道去重后追加")
	assert.Equal(t, service.PolicyRef{Type: entity.NotifyPolicyTypeNotify, Id: 2}, target.PolicyOfChannel(4),
		"渠道 4 被两条策略声明，归因到 Id 更小的首个声明者")
	assert.Equal(t, service.PolicyRef{Type: entity.NotifyPolicyTypeNotify, Id: 7}, target.PolicyOfChannel(5))
	assert.Equal(t, service.PolicyRef{Type: entity.NotifyPolicyTypeNotify, Id: 2}, target.PolicyOfChannel(6))
}

// TestResolveNotifyTargetNoMatchIsUnattributed 无策略命中时返回空目标（不发通知），
// 且不留任何归因——空目标上查渠道必须落「来源未知」，不能凭空给出一个策略
func TestResolveNotifyTargetNoMatchIsUnattributed(t *testing.T) {
	engine := engineWithPolicies(notifyPolicy(2, `{"notify":"ops"}`, 4))

	target := engine.resolveNotifyTarget(&entity.AlertRule{}, devEvent())

	assert.Empty(t, target.ChannelIds, "标签不匹配时不该路由到任何渠道")
	assert.Empty(t, target.ReceiverIds)
	assert.Equal(t, service.PolicyRef{}, target.PolicyOfChannel(4))
}

// TestResolveNotifyTargetEmptyMatchLabelsHitsEverything 通知策略未绑定标签时命中所有事件。
//
// 与升级策略同源的既定语义（matchRuleLabelsMap：空 matchLabels = 匹配所有），
// 钉成用例以便将来任何收紧都是显式决策
func TestResolveNotifyTargetEmptyMatchLabelsHitsEverything(t *testing.T) {
	engine := engineWithPolicies(notifyPolicy(2, "", 4))

	target := engine.resolveNotifyTarget(&entity.AlertRule{}, devEvent())

	require.Equal(t, []uint64{4}, target.ChannelIds)
	assert.Equal(t, service.PolicyRef{Type: entity.NotifyPolicyTypeNotify, Id: 2}, target.PolicyOfChannel(4))
}

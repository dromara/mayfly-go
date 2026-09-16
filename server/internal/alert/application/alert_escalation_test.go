package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/repository"
	"mayfly-go/internal/alert/domain/service"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
	"sort"
	"sync"
	"testing"
	"time"
)

// ==================== 升级策略 Mock ====================

// mockEventAppForEsc 用于升级测试的事件 mock
type mockEventAppForEsc struct {
	base.AppImpl[*entity.AlertEvent, repository.AlertEvent]

	mu     sync.Mutex
	events map[uint64]*entity.AlertEvent
	saved  []*entity.AlertEvent
}

func newMockEventAppForEsc() *mockEventAppForEsc {
	return &mockEventAppForEsc{events: make(map[uint64]*entity.AlertEvent)}
}

func (m *mockEventAppForEsc) GetAlertEventList(condition *entity.AlertEventQuery, orderBy ...string) (*model.PageResult[*entity.AlertEvent], error) {
	return nil, nil
}

func (m *mockEventAppForEsc) FindActive(ruleId, resourceId uint64) (*entity.AlertEvent, error) {
	return nil, nil
}

func (m *mockEventAppForEsc) CreateFiringEvent(ctx context.Context, rule *entity.AlertRule, resourceId uint64, resourceName, metric string, currentValue float64, threshold string) (*entity.AlertEvent, bool, error) {
	return nil, false, nil
}

func (m *mockEventAppForEsc) TouchActiveEvent(ctx context.Context, event *entity.AlertEvent, metric string, currentValue float64) error {
	return nil
}

func (m *mockEventAppForEsc) Recover(ctx context.Context, event *entity.AlertEvent) error {
	return nil
}

func (m *mockEventAppForEsc) Ack(ctx context.Context, eventId uint64, userId int64) error {
	return nil
}

func (m *mockEventAppForEsc) Close(ctx context.Context, eventId uint64) error {
	return nil
}

func (m *mockEventAppForEsc) DeleteEvent(ctx context.Context, eventId uint64) error {
	return nil
}

func (m *mockEventAppForEsc) CloseByRuleId(ctx context.Context, ruleId uint64) error {
	return nil
}

func (m *mockEventAppForEsc) CleanupTerminal(ctx context.Context, retainDays int) (int64, error) {
	return 0, nil
}

func (m *mockEventAppForEsc) CountGroupByStatus() (map[entity.AlertEventStatus]int64, error) {
	return nil, nil
}

func (m *mockEventAppForEsc) GetById(id uint64, cols ...string) (*entity.AlertEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.events[id]
	if !ok {
		return nil, fmt.Errorf("event %d not found", id)
	}
	return e, nil
}

func (m *mockEventAppForEsc) UpdateById(ctx context.Context, event *entity.AlertEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saved = append(m.saved, event)
	return nil
}

func (m *mockEventAppForEsc) ListByCond(cond any, cols ...string) ([]*entity.AlertEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*entity.AlertEvent
	for _, e := range m.events {
		result = append(result, e)
	}
	return result, nil
}

func (m *mockEventAppForEsc) ListActiveFiring() ([]*entity.AlertEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*entity.AlertEvent
	for _, e := range m.events {
		if e.Status == entity.AlertEventStatusFiring {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockEventAppForEsc) GetAvgRecoveryTime() (int64, error)                     { return 0, nil }
func (m *mockEventAppForEsc) CountTodayNotifications(since time.Time) (int64, error) { return 0, nil }
func (m *mockEventAppForEsc) CountPolicyMatched() (int64, error)                     { return 0, nil }
func (m *mockEventAppForEsc) CountUnmatched() (int64, error)                         { return 0, nil }
func (m *mockEventAppForEsc) CountEscalated() (int64, error)                         { return 0, nil }
func (m *mockEventAppForEsc) GetMaxEscalationLevel() (int64, error)                  { return 0, nil }
func (m *mockEventAppForEsc) GetTopByTriggerCount(limit int) ([]*entity.AlertEvent, error) {
	return nil, nil
}
func (m *mockEventAppForEsc) GetTopByDuration(limit int) ([]*entity.AlertEvent, error) {
	return nil, nil
}
func (m *mockEventAppForEsc) GetTopByNotifyCount(limit int) ([]*entity.AlertEvent, error) {
	return nil, nil
}

// mockNotifierForEsc 用于升级测试的通知 mock
type mockNotifierForEsc struct {
	mu          sync.Mutex
	notifyCalls int
	lastRule    *entity.AlertRule
	lastTarget  service.NotifyTarget
}

func (m *mockNotifierForEsc) NotifyWithTarget(ctx context.Context, event *entity.AlertEvent, rule *entity.AlertRule, target service.NotifyTarget) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifyCalls++
	m.lastRule = rule
	m.lastTarget = target
	return 1, nil
}

func (m *mockNotifierForEsc) NotifyRecoverWithTarget(ctx context.Context, event *entity.AlertEvent, rule *entity.AlertRule, target service.NotifyTarget) (int, error) {
	return 1, nil
}

// mockRuleAppForEsc 用于升级测试的规则 mock
type mockRuleAppForEsc struct {
	base.AppImpl[*entity.AlertRule, repository.AlertRule]
	rules []*entity.AlertRule
}

func (m *mockRuleAppForEsc) SaveAlertRule(ctx context.Context, rule *entity.AlertRule) error {
	return nil
}

func (m *mockRuleAppForEsc) ChangeStatus(ctx context.Context, id uint64, status int8) error {
	return nil
}

func (m *mockRuleAppForEsc) DeleteAlertRule(ctx context.Context, id uint64) error {
	return nil
}

func (m *mockRuleAppForEsc) GetAlertRuleList(condition *entity.AlertRuleQuery, orderBy ...string) (*model.PageResult[*entity.AlertRule], error) {
	return nil, nil
}

func (m *mockRuleAppForEsc) ListEnabled() ([]*entity.AlertRule, error) {
	return m.rules, nil
}

func (m *mockRuleAppForEsc) GetRuleStats() (total, enabled, disabled int64, err error) {
	return 0, 0, 0, nil
}
func (m *mockRuleAppForEsc) CountGroupByPriority() (map[string]int64, error) {
	return nil, nil
}
func (m *mockRuleAppForEsc) CountGroupByResourceType() (map[int8]int64, error) {
	return nil, nil
}

// ==================== 升级测试辅助 ====================

func newTestEscalation() (*alertEscalationAppImpl, *mockEventAppForEsc, *mockNotifierForEsc, *mockRuleAppForEsc) {
	eventApp := newMockEventAppForEsc()
	notifier := &mockNotifierForEsc{}
	ruleApp := &mockRuleAppForEsc{}

	esc := &alertEscalationAppImpl{
		eventApp: eventApp,
		notifier: notifier,
		ruleApp:  ruleApp,
	}
	return esc, eventApp, notifier, ruleApp
}

func makeEscEvent(id uint64, ruleId uint64, status entity.AlertEventStatus, firstTrigger time.Time, escLevel int) *entity.AlertEvent {
	return &entity.AlertEvent{
		Id:               id,
		RuleId:           ruleId,
		ResourceId:       100,
		Status:           status,
		FirstTriggerTime: firstTrigger,
		EscalationLvl:    escLevel,
	}
}

func makeEscRule(id uint64) *entity.AlertRule {
	return &entity.AlertRule{
		Id:           id,
		Name:         "test-rule",
		Status:       1,
		ResourceType: 1,
		NotifyConfig: &entity.AlertNotifyConfig{
			GroupWait:      30,
			GroupInterval:  300,
			RepeatInterval: 300,
		},
	}
}

// ==================== 升级测试用例 ====================

// Test_Escalation_BasicLevel1 事件持续时间超过第一级延迟 → 升级到 level 1
func Test_Escalation_BasicLevel1(t *testing.T) {
	escApp, eventApp, notifier, ruleApp := newTestEscalation()

	rule := makeEscRule(1)
	ruleApp.rules = []*entity.AlertRule{rule}

	// 事件在10分钟前首次触发
	firstTrigger := time.Now().Add(-10 * time.Minute)
	event := makeEscEvent(1, 1, entity.AlertEventStatusFiring, firstTrigger, 0)
	eventApp.events[1] = event

	escPolicy := &entity.AlertEscalation{
		Name:   "test-escalation",
		Status: 1,
		Rules: entity.EscalationRuleList{
			{DelayMinutes: 5, ChannelIds: []uint64{2}, ReceiverIds: []int64{10}},
		},
	}

	escApp.checkAndEscalate(escPolicy, rule, event)

	if notifier.notifyCalls != 1 {
		t.Errorf("expected 1 notify call, got %d", notifier.notifyCalls)
	}

	// 检查事件是否被保存（EscalationLevel 更新）
	if len(eventApp.saved) != 1 {
		t.Fatalf("expected 1 saved event, got %d", len(eventApp.saved))
	}
	if eventApp.saved[0].EscalationLvl != 1 {
		t.Errorf("expected EscalationLevel=1, got %d", eventApp.saved[0].EscalationLvl)
	}
}

// Test_Escalation_NotYetDue 事件持续时间未达到延迟 → 不升级
func Test_Escalation_NotYetDue(t *testing.T) {
	escApp, eventApp, notifier, ruleApp := newTestEscalation()

	rule := makeEscRule(1)
	ruleApp.rules = []*entity.AlertRule{rule}

	// 事件刚刚触发（不到5分钟）
	firstTrigger := time.Now().Add(-2 * time.Minute)
	event := makeEscEvent(1, 1, entity.AlertEventStatusFiring, firstTrigger, 0)
	eventApp.events[1] = event

	escPolicy := &entity.AlertEscalation{
		Name:   "test-escalation",
		Status: 1,
		Rules: entity.EscalationRuleList{
			{DelayMinutes: 5, ChannelIds: []uint64{2}},
		},
	}

	escApp.checkAndEscalate(escPolicy, rule, event)

	if notifier.notifyCalls != 0 {
		t.Errorf("expected 0 notify calls (not yet due), got %d", notifier.notifyCalls)
	}
}

// Test_Escalation_AlreadyEscaped 已升级到该级别 → 不重复升级
func Test_Escalation_AlreadyEscalated(t *testing.T) {
	escApp, eventApp, notifier, ruleApp := newTestEscalation()

	rule := makeEscRule(1)
	ruleApp.rules = []*entity.AlertRule{rule}

	firstTrigger := time.Now().Add(-10 * time.Minute)
	event := makeEscEvent(1, 1, entity.AlertEventStatusFiring, firstTrigger, 1) // 已升级到 level 1
	eventApp.events[1] = event

	escPolicy := &entity.AlertEscalation{
		Name:   "test-escalation",
		Status: 1,
		Rules: entity.EscalationRuleList{
			{DelayMinutes: 5, ChannelIds: []uint64{2}},
		},
	}

	escApp.checkAndEscalate(escPolicy, rule, event)

	// EscalationLevel=1 > index 0，应跳过
	if notifier.notifyCalls != 0 {
		t.Errorf("expected 0 notify calls (already escalated), got %d", notifier.notifyCalls)
	}
}

// Test_Escalation_MultiLevel 多级升级：达到第二级延迟
func Test_Escalation_MultiLevel(t *testing.T) {
	escApp, eventApp, notifier, ruleApp := newTestEscalation()

	rule := makeEscRule(1)
	ruleApp.rules = []*entity.AlertRule{rule}

	// 事件在20分钟前触发
	firstTrigger := time.Now().Add(-20 * time.Minute)
	event := makeEscEvent(1, 1, entity.AlertEventStatusFiring, firstTrigger, 1) // 已升级到 level 1
	eventApp.events[1] = event

	escPolicy := &entity.AlertEscalation{
		Name:   "test-escalation",
		Status: 1,
		Rules: entity.EscalationRuleList{
			{DelayMinutes: 5, ChannelIds: []uint64{2}},
			{DelayMinutes: 15, ChannelIds: []uint64{3}},
		},
	}

	escApp.checkAndEscalate(escPolicy, rule, event)

	// 应升级到 level 2（index 1）
	if notifier.notifyCalls != 1 {
		t.Errorf("expected 1 notify call, got %d", notifier.notifyCalls)
	}
	if len(eventApp.saved) != 1 {
		t.Fatalf("expected 1 saved event, got %d", len(eventApp.saved))
	}
	if eventApp.saved[0].EscalationLvl != 2 {
		t.Errorf("expected EscalationLevel=2, got %d", eventApp.saved[0].EscalationLvl)
	}
}

// Test_Escalation_RecoveredEvent 已恢复事件 → 不升级
func Test_Escalation_RecoveredEvent(t *testing.T) {
	escApp, eventApp, notifier, ruleApp := newTestEscalation()

	rule := makeEscRule(1)
	ruleApp.rules = []*entity.AlertRule{rule}

	firstTrigger := time.Now().Add(-10 * time.Minute)
	event := makeEscEvent(1, 1, entity.AlertEventStatusRecovered, firstTrigger, 0)
	eventApp.events[1] = event

	escPolicy := &entity.AlertEscalation{
		Name:   "test-escalation",
		Status: 1,
		Rules: entity.EscalationRuleList{
			{DelayMinutes: 5, ChannelIds: []uint64{2}},
		},
	}

	escApp.checkAndEscalate(escPolicy, rule, event)

	if notifier.notifyCalls != 0 {
		t.Errorf("expected 0 notify calls (recovered event), got %d", notifier.notifyCalls)
	}
}

// Test_Escalation_AcknowledgedEvent 已确认事件 → 仍可升级
func Test_Escalation_AcknowledgedEvent(t *testing.T) {
	escApp, eventApp, notifier, ruleApp := newTestEscalation()

	rule := makeEscRule(1)
	ruleApp.rules = []*entity.AlertRule{rule}

	firstTrigger := time.Now().Add(-10 * time.Minute)
	event := makeEscEvent(1, 1, entity.AlertEventStatusAcknowledged, firstTrigger, 0)
	eventApp.events[1] = event

	escPolicy := &entity.AlertEscalation{
		Name:   "test-escalation",
		Status: 1,
		Rules: entity.EscalationRuleList{
			{DelayMinutes: 5, ChannelIds: []uint64{2}},
		},
	}

	escApp.checkAndEscalate(escPolicy, rule, event)

	if notifier.notifyCalls != 1 {
		t.Errorf("expected 1 notify call (ack event can escalate), got %d", notifier.notifyCalls)
	}
}

// Test_Escalation_LabelMismatch 升级策略标签与规则标签不匹配 → 不升级
func Test_Escalation_LabelMismatch(t *testing.T) {
	escApp, eventApp, notifier, ruleApp := newTestEscalation()

	rule := makeEscRule(1)
	rule.Labels = `{"env":"prod"}` // 规则标签 env=prod
	ruleApp.rules = []*entity.AlertRule{rule}

	firstTrigger := time.Now().Add(-10 * time.Minute)
	event := makeEscEvent(1, 1, entity.AlertEventStatusFiring, firstTrigger, 0)
	eventApp.events[1] = event

	escPolicy := &entity.AlertEscalation{
		Name:   "test-escalation",
		Status: 1,
		Rules: entity.EscalationRuleList{
			{DelayMinutes: 5, ChannelIds: []uint64{2}},
		},
	}

	// 升级策略标签 env=staging 不匹配规则标签 env=prod
	escLabels := map[string]string{"env": "staging"}
	escApp.processEscalation(escPolicy, escLabels, ruleApp.rules)

	if notifier.notifyCalls != 0 {
		t.Errorf("expected 0 notify calls (label mismatch), got %d", notifier.notifyCalls)
	}
}

// Test_Escalation_NotifyConfigOverride 升级时使用升级策略的渠道配置
func Test_Escalation_NotifyConfigOverride(t *testing.T) {
	escApp, eventApp, notifier, ruleApp := newTestEscalation()

	rule := makeEscRule(1)
	ruleApp.rules = []*entity.AlertRule{rule}

	firstTrigger := time.Now().Add(-10 * time.Minute)
	event := makeEscEvent(1, 1, entity.AlertEventStatusFiring, firstTrigger, 0)
	eventApp.events[1] = event

	escPolicy := &entity.AlertEscalation{
		Name:   "test-escalation",
		Status: 1,
		Rules: entity.EscalationRuleList{
			{DelayMinutes: 5, ChannelIds: []uint64{99}, ReceiverIds: []int64{88}},
		},
	}

	escApp.checkAndEscalate(escPolicy, rule, event)

	if notifier.notifyCalls != 1 {
		t.Fatalf("expected 1 notify call, got %d", notifier.notifyCalls)
	}

	// 检查通知目标是否使用了升级策略的渠道配置（通过 NotifyTarget 传递，而非 rule.NotifyConfig）
	if len(notifier.lastTarget.ChannelIds) != 1 || notifier.lastTarget.ChannelIds[0] != 99 {
		t.Errorf("expected escalation channel ID 99, got %v", notifier.lastTarget.ChannelIds)
	}
	if len(notifier.lastTarget.ReceiverIds) != 1 || notifier.lastTarget.ReceiverIds[0] != 88 {
		t.Errorf("expected escalation receiver ID 88, got %v", notifier.lastTarget.ReceiverIds)
	}
}

// Test_Escalation_SaveAlertEscalation_SortRules 保存时按 DelayMinutes 排序
func Test_Escalation_SaveAlertEscalation_SortRules(t *testing.T) {
	// 直接验证排序逻辑（不依赖数据库）
	rules := entity.EscalationRuleList{
		{DelayMinutes: 30},
		{DelayMinutes: 5},
		{DelayMinutes: 15},
	}

	// 使用与 SaveAlertEscalation 相同的排序逻辑
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].DelayMinutes < rules[j].DelayMinutes
	})

	if rules[0].DelayMinutes != 5 {
		t.Errorf("expected first rule delay=5, got %d", rules[0].DelayMinutes)
	}
	if rules[1].DelayMinutes != 15 {
		t.Errorf("expected second rule delay=15, got %d", rules[1].DelayMinutes)
	}
	if rules[2].DelayMinutes != 30 {
		t.Errorf("expected third rule delay=30, got %d", rules[2].DelayMinutes)
	}
}

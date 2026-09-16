package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/repository"
	"mayfly-go/internal/alert/domain/service"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
	"sync"
	"testing"
	"time"
)

// ==================== Mock 实现 ====================

// --- mockBreachTracker ---
type mockBreachTracker struct {
	mu      sync.Mutex
	records map[string]time.Time
}

func newMockBreachTracker() *mockBreachTracker {
	return &mockBreachTracker{records: make(map[string]time.Time)}
}

func (m *mockBreachTracker) GetFirstBreach(ruleId, resourceId uint64) time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := "breach:" + breachCounterKey(ruleId, resourceId)
	t, ok := m.records[key]
	if !ok {
		return time.Time{}
	}
	return t
}

func (m *mockBreachTracker) SetFirstBreach(ruleId, resourceId uint64, t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := "breach:" + breachCounterKey(ruleId, resourceId)
	m.records[key] = t
}

func (m *mockBreachTracker) DelFirstBreach(ruleId, resourceId uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := "breach:" + breachCounterKey(ruleId, resourceId)
	delete(m.records, key)
}

// --- mockConsecutiveCounter ---
type mockConsecutiveCounter struct {
	mu       sync.Mutex
	counters map[string]int64
}

func newMockCounter() *mockConsecutiveCounter {
	return &mockConsecutiveCounter{counters: make(map[string]int64)}
}

func (m *mockConsecutiveCounter) Get(key string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counters[key]
}

func (m *mockConsecutiveCounter) Incr(key string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[key]++
	return m.counters[key]
}

func (m *mockConsecutiveCounter) Reset(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[key] = 0
}

// --- mockEventApp ---
type mockEventApp struct {
	base.AppImpl[*entity.AlertEvent, repository.AlertEvent]

	mu            sync.Mutex
	activeEvents  map[string]*entity.AlertEvent
	createdEvents []*entity.AlertEvent
	touchedEvents []*entity.AlertEvent
	recoveredEvts []*entity.AlertEvent
	createErr     error
}

func newMockEventApp() *mockEventApp {
	return &mockEventApp{activeEvents: make(map[string]*entity.AlertEvent)}
}

func eventKey(ruleId, resourceId uint64) string {
	return fmt.Sprintf("%d:%d", ruleId, resourceId)
}

func (m *mockEventApp) GetAlertEventList(condition *entity.AlertEventQuery, orderBy ...string) (*model.PageResult[*entity.AlertEvent], error) {
	return nil, nil
}

func (m *mockEventApp) FindActive(ruleId, resourceId uint64) (*entity.AlertEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := eventKey(ruleId, resourceId)
	e, ok := m.activeEvents[key]
	if !ok {
		return nil, nil
	}
	return e, nil
}

func (m *mockEventApp) CreateFiringEvent(ctx context.Context, rule *entity.AlertRule, resourceId uint64, resourceName, metric string, currentValue float64, threshold string) (*entity.AlertEvent, bool, error) {
	if m.createErr != nil {
		return nil, false, m.createErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := eventKey(rule.Id, resourceId)
	if e, ok := m.activeEvents[key]; ok {
		return e, true, nil
	}
	e := &entity.AlertEvent{
		Id:               uint64(len(m.createdEvents) + 1),
		RuleId:           rule.Id,
		RuleName:         rule.Name,
		Priority:         rule.Priority,
		ResourceType:     rule.ResourceType,
		ResourceId:       resourceId,
		ResourceName:     resourceName,
		Status:           entity.AlertEventStatusFiring,
		Metric:           metric,
		CurrentValue:     currentValue,
		Threshold:        threshold,
		FirstTriggerTime: time.Now(),
		LastTriggerTime:  time.Now(),
		TriggerCount:     1,
	}
	m.createdEvents = append(m.createdEvents, e)
	m.activeEvents[key] = e
	return e, false, nil
}

func (m *mockEventApp) TouchActiveEvent(ctx context.Context, event *entity.AlertEvent, metric string, currentValue float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	event.LastTriggerTime = time.Now()
	event.TriggerCount++
	event.Metric = metric
	event.CurrentValue = currentValue
	m.touchedEvents = append(m.touchedEvents, event)
	return nil
}

func (m *mockEventApp) Recover(ctx context.Context, event *entity.AlertEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	event.Status = entity.AlertEventStatusRecovered
	event.RecoverTime = &now
	m.recoveredEvts = append(m.recoveredEvts, event)
	key := eventKey(event.RuleId, event.ResourceId)
	delete(m.activeEvents, key)
	return nil
}

func (m *mockEventApp) Ack(ctx context.Context, eventId uint64, userId int64) error {
	return nil
}

func (m *mockEventApp) Close(ctx context.Context, eventId uint64) error {
	return nil
}

func (m *mockEventApp) DeleteEvent(ctx context.Context, eventId uint64) error {
	return nil
}

func (m *mockEventApp) CloseByRuleId(ctx context.Context, ruleId uint64) error {
	return nil
}

func (m *mockEventApp) CleanupTerminal(ctx context.Context, retainDays int) (int64, error) {
	return 0, nil
}

func (m *mockEventApp) CountGroupByStatus() (map[entity.AlertEventStatus]int64, error) {
	return nil, nil
}

func (m *mockEventApp) GetAvgRecoveryTime() (int64, error)                           { return 0, nil }
func (m *mockEventApp) CountTodayNotifications(since time.Time) (int64, error)       { return 0, nil }
func (m *mockEventApp) CountPolicyMatched() (int64, error)                           { return 0, nil }
func (m *mockEventApp) CountUnmatched() (int64, error)                               { return 0, nil }
func (m *mockEventApp) CountEscalated() (int64, error)                               { return 0, nil }
func (m *mockEventApp) GetMaxEscalationLevel() (int64, error)                        { return 0, nil }
func (m *mockEventApp) GetTopByTriggerCount(limit int) ([]*entity.AlertEvent, error) { return nil, nil }
func (m *mockEventApp) GetTopByDuration(limit int) ([]*entity.AlertEvent, error)     { return nil, nil }
func (m *mockEventApp) GetTopByNotifyCount(limit int) ([]*entity.AlertEvent, error)  { return nil, nil }

func (m *mockEventApp) ListActiveFiring() ([]*entity.AlertEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	events := make([]*entity.AlertEvent, 0, len(m.activeEvents))
	for _, e := range m.activeEvents {
		if e.Status == entity.AlertEventStatusFiring {
			events = append(events, e)
		}
	}
	return events, nil
}

func (m *mockEventApp) UpdateById(ctx context.Context, event *entity.AlertEvent) error {
	return nil
}

// --- mockNotifier ---
type mockNotifier struct {
	mu           sync.Mutex
	notifyCalls  int
	recoverCalls int
	// sentChannels 模拟"可用渠道数"，为 0 时表示实际未发出任何通知
	sentChannels int
	notifyErr    error
}

func (m *mockNotifier) NotifyWithTarget(ctx context.Context, event *entity.AlertEvent, rule *entity.AlertRule, target service.NotifyTarget) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifyCalls++
	if m.notifyErr != nil {
		return 0, m.notifyErr
	}
	return m.sentChannels, nil
}

func (m *mockNotifier) NotifyRecoverWithTarget(ctx context.Context, event *entity.AlertEvent, rule *entity.AlertRule, target service.NotifyTarget) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recoverCalls++
	return m.sentChannels, nil
}

// --- mockEvaluator ---

// mockEvaluator 满足 service.AlertEvaluator 契约的最小实现。
// handleEvalResult 依赖它解析资源展示名，因此测试统一使用固定资源名
type mockEvaluator struct{}

func (m *mockEvaluator) ResourceType() int8 { return 1 }

func (m *mockEvaluator) Metrics() []service.MetricDefinition { return nil }

func (m *mockEvaluator) Evaluate(ctx context.Context, param *service.EvalParam) (*service.EvalResult, error) {
	return nil, nil
}

func (m *mockEvaluator) ResourceName(resourceId uint64) string { return "test-machine" }

func (m *mockEvaluator) ResourceExists(resourceId uint64) bool { return resourceId == 1 }

func (m *mockEvaluator) ResourceIdsByCodes(codes []string) []uint64 {
	// 测试用：返回 code 对应的 ID（简单映射）
	ids := make([]uint64, 0, len(codes))
	for _, code := range codes {
		if code == "test-code" {
			ids = append(ids, 1)
		}
	}
	return ids
}

var testEvaluator service.AlertEvaluator = &mockEvaluator{}

// handleResult 以测试用评估器驱动状态机，避免每个用例重复传参
func handleResult(engine *alertEngineAppImpl, rule *entity.AlertRule, resourceId uint64, result *service.EvalResult) {
	engine.handleEvalResult(context.Background(), rule, resourceId, testEvaluator, result)
}

// ==================== 辅助函数 ====================

// mockNotifyPolicyApp 通知策略 mock：MatchPolicies 始终返回空列表（无匹配策略）
type mockNotifyPolicyApp struct {
	base.AppImpl[*entity.AlertNotifyPolicy, repository.AlertNotifyPolicy]
}

func (m *mockNotifyPolicyApp) SaveAlertNotifyPolicy(ctx context.Context, policy *entity.AlertNotifyPolicy) error {
	return nil
}
func (m *mockNotifyPolicyApp) GetAlertNotifyPolicyList(condition *entity.AlertNotifyPolicyQuery, orderBy ...string) (*model.PageResult[*entity.AlertNotifyPolicy], error) {
	return nil, nil
}
func (m *mockNotifyPolicyApp) DeleteNotifyPolicy(ctx context.Context, id uint64) error { return nil }
func (m *mockNotifyPolicyApp) ChangeStatus(ctx context.Context, id uint64, status int8) error {
	return nil
}
func (m *mockNotifyPolicyApp) MatchPolicies(ctx context.Context, eventLabels map[string]string) ([]*entity.AlertNotifyPolicy, error) {
	return nil, nil
}
func (m *mockNotifyPolicyApp) GetChannelDistribution() (map[string]int64, error) {
	return nil, nil
}

func newTestEngine() (*alertEngineAppImpl, *mockBreachTracker, *mockConsecutiveCounter, *mockEventApp, *mockNotifier) {
	tracker := newMockBreachTracker()
	counter := newMockCounter()
	eventApp := newMockEventApp()
	notifier := &mockNotifier{sentChannels: 1}

	engine := &alertEngineAppImpl{
		breachTracker:   tracker,
		counter:         counter,
		eventApp:        eventApp,
		notifyPolicyApp: &mockNotifyPolicyApp{},
		notifier:        notifier,
	}
	engine.pendingGroups = make(map[groupKey]*pendingGroup)
	engine.lastEvalTime = make(map[uint64]time.Time)
	engine.evalSem = make(chan struct{}, 10)

	return engine, tracker, counter, eventApp, notifier
}

func makeRule(id uint64, triggerCount, recoveryCount int, duration int) *entity.AlertRule {
	return &entity.AlertRule{
		Id:            id,
		Name:          "test-rule",
		Status:        1,
		Priority:      entity.AlertPriorityHigh,
		ResourceType:  1,
		ScopeType:     entity.AlertScopeResource,
		ScopeValue:    "1",
		TriggerCount:  triggerCount,
		RecoveryCount: recoveryCount,
		Condition: &entity.AlertCondition{
			Operator: "and",
			Items: []entity.ConditionItem{
				{Metric: "cpu_rate", Compare: "gt", Value: 80, Duration: duration},
			},
		},
		NotifyConfig: &entity.AlertNotifyConfig{
			GroupWait:      30,
			GroupInterval:  300,
			RepeatInterval: 300,
		},
	}
}

func breachedResult() *service.EvalResult {
	return &service.EvalResult{
		Triggered: true,
		Items: []service.ConditionEval{
			{Metric: "cpu_rate", CurrentValue: 95.0, Threshold: 80, Compare: "gt", Satisfied: true},
		},
	}
}

func normalResult() *service.EvalResult {
	return &service.EvalResult{
		Triggered: false,
		Items: []service.ConditionEval{
			{Metric: "cpu_rate", CurrentValue: 50.0, Threshold: 80, Compare: "gt", Satisfied: false},
		},
	}
}

// ==================== 测试用例 ====================

func Test1_BreachCreatesEvent(t *testing.T) {
	engine, _, _, eventApp, _ := newTestEngine()
	rule := makeRule(1, 1, 1, 0)

	handleResult(engine, rule, 100, breachedResult())

	if len(eventApp.createdEvents) != 1 {
		t.Fatalf("expected 1 created event, got %d", len(eventApp.createdEvents))
	}
	e := eventApp.createdEvents[0]
	if e.Status != entity.AlertEventStatusFiring {
		t.Errorf("expected Firing status, got %d", e.Status)
	}
	if e.Metric != "cpu_rate" {
		t.Errorf("expected metric cpu_rate, got %s", e.Metric)
	}
	if e.CurrentValue != 95.0 {
		t.Errorf("expected currentValue 95.0, got %v", e.CurrentValue)
	}
}

func Test2_TriggerCountDebounce(t *testing.T) {
	engine, _, _, eventApp, _ := newTestEngine()
	rule := makeRule(1, 3, 1, 0)

	handleResult(engine, rule, 100, breachedResult())
	if len(eventApp.createdEvents) != 0 {
		t.Fatalf("1st breach: expected 0 events, got %d", len(eventApp.createdEvents))
	}

	handleResult(engine, rule, 100, breachedResult())
	if len(eventApp.createdEvents) != 0 {
		t.Fatalf("2nd breach: expected 0 events, got %d", len(eventApp.createdEvents))
	}

	handleResult(engine, rule, 100, breachedResult())
	if len(eventApp.createdEvents) != 1 {
		t.Fatalf("3rd breach: expected 1 event, got %d", len(eventApp.createdEvents))
	}
}

func Test3_DurationDebounce(t *testing.T) {
	engine, tracker, _, eventApp, _ := newTestEngine()
	rule := makeRule(1, 1, 1, 300)

	tracker.SetFirstBreach(1, 100, time.Now())
	handleResult(engine, rule, 100, breachedResult())
	if len(eventApp.createdEvents) != 0 {
		t.Fatalf("duration not met: expected 0 events, got %d", len(eventApp.createdEvents))
	}

	tracker.SetFirstBreach(1, 100, time.Now().Add(-301*time.Second))
	handleResult(engine, rule, 100, breachedResult())
	if len(eventApp.createdEvents) != 1 {
		t.Fatalf("duration met: expected 1 event, got %d", len(eventApp.createdEvents))
	}
}

func Test4_ActiveEventTouch(t *testing.T) {
	engine, _, _, eventApp, _ := newTestEngine()
	rule := makeRule(1, 1, 1, 0)

	handleResult(engine, rule, 100, breachedResult())
	if len(eventApp.createdEvents) != 1 {
		t.Fatalf("expected 1 created event, got %d", len(eventApp.createdEvents))
	}

	handleResult(engine, rule, 100, breachedResult())
	if len(eventApp.createdEvents) != 1 {
		t.Fatalf("expected still 1 created event, got %d", len(eventApp.createdEvents))
	}
	if len(eventApp.touchedEvents) != 1 {
		t.Fatalf("expected 1 touched event, got %d", len(eventApp.touchedEvents))
	}
	if eventApp.touchedEvents[0].TriggerCount != 2 {
		t.Errorf("expected TriggerCount=2, got %d", eventApp.touchedEvents[0].TriggerCount)
	}
}

func Test5_RecoveryCountDebounce(t *testing.T) {
	engine, _, _, eventApp, _ := newTestEngine()
	rule := makeRule(1, 1, 3, 0)

	handleResult(engine, rule, 100, breachedResult())
	if len(eventApp.createdEvents) != 1 {
		t.Fatalf("expected 1 created event, got %d", len(eventApp.createdEvents))
	}

	handleResult(engine, rule, 100, normalResult())
	if len(eventApp.recoveredEvts) != 0 {
		t.Fatalf("1st recover: expected 0 recovered, got %d", len(eventApp.recoveredEvts))
	}

	handleResult(engine, rule, 100, normalResult())
	if len(eventApp.recoveredEvts) != 0 {
		t.Fatalf("2nd recover: expected 0 recovered, got %d", len(eventApp.recoveredEvts))
	}

	handleResult(engine, rule, 100, normalResult())
	if len(eventApp.recoveredEvts) != 1 {
		t.Fatalf("3rd recover: expected 1 recovered, got %d", len(eventApp.recoveredEvts))
	}
	if eventApp.recoveredEvts[0].Status != entity.AlertEventStatusRecovered {
		t.Errorf("expected Recovered status, got %d", eventApp.recoveredEvts[0].Status)
	}
}

func Test6_RecoveryNotify(t *testing.T) {
	engine, _, _, _, notifier := newTestEngine()
	rule := makeRule(1, 1, 1, 0)

	handleResult(engine, rule, 100, breachedResult())
	handleResult(engine, rule, 100, normalResult())

	if notifier.recoverCalls != 1 {
		t.Errorf("expected 1 recover notify, got %d", notifier.recoverCalls)
	}
}

func Test7_BreachResetsRecoverCounter(t *testing.T) {
	engine, _, counter, eventApp, _ := newTestEngine()
	rule := makeRule(1, 1, 3, 0)

	handleResult(engine, rule, 100, breachedResult())
	handleResult(engine, rule, 100, normalResult())
	handleResult(engine, rule, 100, normalResult())

	handleResult(engine, rule, 100, breachedResult())

	recoverKey := recoverCounterKey(1, 100)
	if counter.Get(recoverKey) != 0 {
		t.Errorf("expected recover counter reset to 0, got %d", counter.Get(recoverKey))
	}

	handleResult(engine, rule, 100, normalResult())
	if len(eventApp.recoveredEvts) != 0 {
		t.Fatalf("expected 0 recovered after reset, got %d", len(eventApp.recoveredEvts))
	}
}

func Test8_NotBreachedNoActiveEvent(t *testing.T) {
	engine, _, _, eventApp, notifier := newTestEngine()
	rule := makeRule(1, 1, 1, 0)

	handleResult(engine, rule, 100, normalResult())

	if len(eventApp.createdEvents) != 0 {
		t.Error("expected no events created")
	}
	if notifier.recoverCalls != 0 {
		t.Error("expected no recover notify")
	}
}

func Test9_MaxDurationAND(t *testing.T) {
	engine, _, _, _, _ := newTestEngine()
	rule := &entity.AlertRule{
		Condition: &entity.AlertCondition{
			Operator: "and",
			Items: []entity.ConditionItem{
				{Metric: "cpu_rate", Compare: "gt", Value: 80, Duration: 60},
				{Metric: "mem_rate", Compare: "gt", Value: 90, Duration: 120},
			},
		},
	}
	result := &service.EvalResult{
		Items: []service.ConditionEval{
			{Satisfied: true},
			{Satisfied: true},
		},
	}
	maxD := engine.getMaxDurationForResult(rule, result)
	if maxD != 120 {
		t.Errorf("expected max duration 120, got %d", maxD)
	}
}

func Test10_MaxDurationOR(t *testing.T) {
	engine, _, _, _, _ := newTestEngine()
	rule := &entity.AlertRule{
		Condition: &entity.AlertCondition{
			Operator: "or",
			Items: []entity.ConditionItem{
				{Metric: "cpu_rate", Compare: "gt", Value: 80, Duration: 60},
				{Metric: "mem_rate", Compare: "gt", Value: 90, Duration: 120},
			},
		},
	}
	result := &service.EvalResult{
		Items: []service.ConditionEval{
			{Satisfied: true},
			{Satisfied: false},
		},
	}
	maxD := engine.getMaxDurationForResult(rule, result)
	if maxD != 60 {
		t.Errorf("expected max duration 60, got %d", maxD)
	}
}

func Test11_MaxDurationNilCondition(t *testing.T) {
	engine, _, _, _, _ := newTestEngine()
	rule := &entity.AlertRule{Condition: nil}
	result := &service.EvalResult{}
	maxD := engine.getMaxDurationForResult(rule, result)
	if maxD != 0 {
		t.Errorf("expected 0 for nil condition, got %d", maxD)
	}
}

func Test12_GetTriggeredInfo(t *testing.T) {
	engine, _, _, _, _ := newTestEngine()
	result := &service.EvalResult{
		Items: []service.ConditionEval{
			{Metric: "cpu_rate", CurrentValue: 95.5, Threshold: 80, Compare: "gt", Satisfied: true},
		},
	}
	metric, val, threshold := engine.getTriggeredInfo(result)
	if metric != "cpu_rate" {
		t.Errorf("expected metric cpu_rate, got %s", metric)
	}
	if val != 95.5 {
		t.Errorf("expected value 95.5, got %v", val)
	}
	expected := "cpu_rate gt 80.00"
	if threshold != expected {
		t.Errorf("expected threshold %q, got %q", expected, threshold)
	}
}

func Test13_GetTriggeredInfoNoSatisfied(t *testing.T) {
	engine, _, _, _, _ := newTestEngine()
	result := &service.EvalResult{
		Items: []service.ConditionEval{
			{Metric: "cpu_rate", Satisfied: false},
		},
	}
	metric, val, threshold := engine.getTriggeredInfo(result)
	if metric != "" || val != 0 || threshold != "" {
		t.Errorf("expected empty result, got metric=%s val=%v threshold=%s", metric, val, threshold)
	}
}

func Test14_AddToGroup(t *testing.T) {
	engine, _, _, _, _ := newTestEngine()
	rule := makeRule(1, 1, 1, 0)
	event := &entity.AlertEvent{Id: 10, ResourceId: 100, Status: entity.AlertEventStatusFiring}

	engine.addToGroup(rule, event)

	key := groupKey{RuleId: 1, ResourceId: 100}
	group, ok := engine.pendingGroups[key]
	if !ok {
		t.Fatal("expected group to exist")
	}
	if len(group.Events) != 1 {
		t.Errorf("expected 1 event in group, got %d", len(group.Events))
	}
}

func Test15_FlushGroupsGroupWait(t *testing.T) {
	engine, _, _, _, notifier := newTestEngine()
	rule := makeRule(1, 1, 1, 0)
	rule.NotifyConfig = &entity.AlertNotifyConfig{GroupWait: 1} // 1秒等待

	event := &entity.AlertEvent{Id: 10, ResourceId: 100, Status: entity.AlertEventStatusFiring}
	engine.addToGroup(rule, event)

	// FirstFire 设为2秒前，满足 GroupWait=1 的条件
	key := groupKey{RuleId: 1, ResourceId: 100}
	engine.pendingGroups[key].FirstFire = time.Now().Add(-2 * time.Second)

	engine.flushGroups()

	if notifier.notifyCalls != 1 {
		t.Errorf("expected 1 notify call, got %d", notifier.notifyCalls)
	}
}

func Test16_FlushGroupsGroupWaitNotMet(t *testing.T) {
	engine, _, _, _, notifier := newTestEngine()
	rule := makeRule(1, 1, 1, 0)
	rule.NotifyConfig = &entity.AlertNotifyConfig{GroupWait: 60}

	event := &entity.AlertEvent{Id: 10, ResourceId: 100, Status: entity.AlertEventStatusFiring}
	engine.addToGroup(rule, event)

	engine.flushGroups()

	if notifier.notifyCalls != 0 {
		t.Errorf("expected 0 notify calls (group wait not met), got %d", notifier.notifyCalls)
	}
}

func Test17_FlushGroupsCleanup(t *testing.T) {
	engine, _, _, _, _ := newTestEngine()
	rule := makeRule(1, 1, 1, 0)
	rule.NotifyConfig = &entity.AlertNotifyConfig{GroupWait: 0}

	event := &entity.AlertEvent{Id: 10, ResourceId: 100, Status: entity.AlertEventStatusRecovered}
	engine.addToGroup(rule, event)

	engine.flushGroups()

	key := groupKey{RuleId: 1, ResourceId: 100}
	if _, ok := engine.pendingGroups[key]; ok {
		t.Error("expected group to be cleaned up after all events recovered")
	}
}

func Test18_RepeatIntervalThrottle(t *testing.T) {
	engine, _, _, _, notifier := newTestEngine()
	rule := makeRule(1, 1, 1, 0)
	rule.NotifyConfig = &entity.AlertNotifyConfig{RepeatInterval: 3600}

	now := time.Now()
	event := &entity.AlertEvent{
		Id:             10,
		ResourceId:     100,
		Status:         entity.AlertEventStatusFiring,
		NotifyCount:    1,
		LastNotifyTime: &now,
	}

	group := &pendingGroup{
		Events:    []*entity.AlertEvent{event},
		Rule:      rule,
		FirstFire: time.Now(),
	}

	engine.sendGroupNotification(group)

	if notifier.notifyCalls != 0 {
		t.Errorf("expected 0 notify calls (repeat interval), got %d", notifier.notifyCalls)
	}
}

func Test19_ShouldEvaluate(t *testing.T) {
	engine, _, _, _, _ := newTestEngine()
	rule := &entity.AlertRule{Id: 1, EvalInterval: 60}

	if !engine.shouldEvaluate(rule, time.Now()) {
		t.Error("first evaluation should return true")
	}

	if engine.shouldEvaluate(rule, time.Now().Add(10*time.Second)) {
		t.Error("should not evaluate within interval")
	}

	if !engine.shouldEvaluate(rule, time.Now().Add(61*time.Second)) {
		t.Error("should evaluate after interval")
	}
}

func Test20_GetGroupWaitDefault(t *testing.T) {
	engine, _, _, _, _ := newTestEngine()
	rule := &entity.AlertRule{NotifyConfig: nil}
	if engine.getGroupWait(rule) != 30 {
		t.Errorf("expected default GroupWait 30, got %d", engine.getGroupWait(rule))
	}
	rule.NotifyConfig = &entity.AlertNotifyConfig{GroupWait: 10}
	if engine.getGroupWait(rule) != 10 {
		t.Errorf("expected GroupWait 10, got %d", engine.getGroupWait(rule))
	}
}

func Test21_GetGroupIntervalDefault(t *testing.T) {
	engine, _, _, _, _ := newTestEngine()
	rule := &entity.AlertRule{NotifyConfig: nil}
	if engine.getGroupInterval(rule) != 300 {
		t.Errorf("expected default GroupInterval 300, got %d", engine.getGroupInterval(rule))
	}
}

func Test22_GetRepeatIntervalDefault(t *testing.T) {
	engine, _, _, _, _ := newTestEngine()
	rule := &entity.AlertRule{NotifyConfig: nil}
	if engine.getRepeatInterval(rule) != 3600 {
		t.Errorf("expected default RepeatInterval 3600, got %d", engine.getRepeatInterval(rule))
	}
}

// ==================== 缺陷回归用例 ====================

// Test23_DurationFirstBreachRealPath 覆盖"真实首次越界"路径（不预置越界记录）。
//
// 修复前 handleEvalResult 只把首次越界时间写入缓存、未同步局部变量，
// 导致 time.Since(firstBreach) 以零值时间为基准恒为巨大正值，
// 首次越界即满足 Duration，持续时间防抖完全失效。
// 旧用例 Test3 预置了首次越界时间，恰好绕开了这条真实路径，因此缺陷被单测掩盖。
func Test23_DurationFirstBreachRealPath(t *testing.T) {
	engine, _, _, eventApp, _ := newTestEngine()
	rule := makeRule(1, 1, 1, 300)

	handleResult(engine, rule, 100, breachedResult())
	if len(eventApp.createdEvents) != 0 {
		t.Fatalf("首次越界且 Duration=300s 时不得立即产生事件，实际 %d 条", len(eventApp.createdEvents))
	}

	// 持续越界超过 Duration 后才允许触发
	engine.breachTracker.SetFirstBreach(1, 100, time.Now().Add(-301*time.Second))
	handleResult(engine, rule, 100, breachedResult())
	if len(eventApp.createdEvents) != 1 {
		t.Fatalf("越界时长已达 Duration 时应产生 1 条事件，实际 %d 条", len(eventApp.createdEvents))
	}
}

// Test24_FirstBreachStableAcrossCycles 首个越界时间必须在后续周期保持稳定并被持续续期，
// 否则 Duration 会被反复重新计时而永不满足
func Test24_FirstBreachStableAcrossCycles(t *testing.T) {
	engine, tracker, _, _, _ := newTestEngine()
	rule := makeRule(1, 1, 1, 300)

	handleResult(engine, rule, 100, breachedResult())
	first := tracker.GetFirstBreach(1, 100)
	if first.IsZero() {
		t.Fatal("越界后必须记录首次越界时间")
	}

	handleResult(engine, rule, 100, breachedResult())
	handleResult(engine, rule, 100, breachedResult())
	if got := tracker.GetFirstBreach(1, 100); !got.Equal(first) {
		t.Errorf("首次越界时间应保持稳定，got %v want %v", got, first)
	}
}

// Test25_RecoveryClearsFirstBreach 恢复正常后必须清除越界记录，避免下次越界沿用旧时间
func Test25_RecoveryClearsFirstBreach(t *testing.T) {
	engine, tracker, _, _, _ := newTestEngine()
	rule := makeRule(1, 1, 1, 0)

	handleResult(engine, rule, 100, breachedResult())
	if tracker.GetFirstBreach(1, 100).IsZero() {
		t.Fatal("expected first breach recorded")
	}
	handleResult(engine, rule, 100, normalResult())
	if !tracker.GetFirstBreach(1, 100).IsZero() {
		t.Error("恢复后首次越界时间应被清除")
	}
}

// Test26_EventCarriesResourceName 新建事件必须冗余存储资源展示名，供前端列表与通知使用
func Test26_EventCarriesResourceName(t *testing.T) {
	engine, _, _, eventApp, _ := newTestEngine()
	rule := makeRule(1, 1, 1, 0)

	handleResult(engine, rule, 100, breachedResult())
	if len(eventApp.createdEvents) != 1 {
		t.Fatalf("expected 1 created event, got %d", len(eventApp.createdEvents))
	}
	if name := eventApp.createdEvents[0].ResourceName; name != "test-machine" {
		t.Errorf("expected resourceName %q, got %q", "test-machine", name)
	}
}

// Test27_TouchBackfillsResourceName 历史活跃事件缺失资源名时，触摸更新需顺带回填
func Test27_TouchBackfillsResourceName(t *testing.T) {
	engine, _, _, eventApp, _ := newTestEngine()
	rule := makeRule(1, 1, 1, 0)

	handleResult(engine, rule, 100, breachedResult())
	active := eventApp.createdEvents[0]
	active.ResourceName = ""

	handleResult(engine, rule, 100, breachedResult())
	if active.ResourceName != "test-machine" {
		t.Errorf("expected resourceName backfilled, got %q", active.ResourceName)
	}
}

// Test28_NoChannelKeepsNotifyCount 无可用渠道时不得虚增通知次数，
// 也不得更新通知时间（否则 RepeatInterval 窗口会被一次空发送消费）
func Test28_NoChannelKeepsNotifyCount(t *testing.T) {
	engine, _, _, _, notifier := newTestEngine()
	notifier.sentChannels = 0

	rule := makeRule(1, 1, 1, 0)
	event := &entity.AlertEvent{Id: 10, ResourceId: 100, Status: entity.AlertEventStatusFiring}
	engine.sendGroupNotification(&pendingGroup{
		Events:    []*entity.AlertEvent{event},
		Rule:      rule,
		FirstFire: time.Now(),
	})

	if event.NotifyCount != 0 {
		t.Errorf("expected NotifyCount 0, got %d", event.NotifyCount)
	}
	if event.LastNotifyTime != nil {
		t.Errorf("expected LastNotifyTime untouched, got %v", *event.LastNotifyTime)
	}
}

// Test29_NotifyErrorKeepsNotifyCount 通知报错同样不得累加通知次数
func Test29_NotifyErrorKeepsNotifyCount(t *testing.T) {
	engine, _, _, _, notifier := newTestEngine()
	notifier.notifyErr = fmt.Errorf("channel unavailable")

	rule := makeRule(1, 1, 1, 0)
	event := &entity.AlertEvent{Id: 10, ResourceId: 100, Status: entity.AlertEventStatusFiring}
	engine.sendGroupNotification(&pendingGroup{
		Events:    []*entity.AlertEvent{event},
		Rule:      rule,
		FirstFire: time.Now(),
	})

	if event.NotifyCount != 0 {
		t.Errorf("expected NotifyCount 0 on notify error, got %d", event.NotifyCount)
	}
}

// Test30_UnreliableResultKeepsCountersAligned 校验 Items 与条件项下标对齐的不变式：
// OR 条件下 Duration 必须取自真正满足的那个条件项，而非按结果下标误配
func Test30_UnreliableResultKeepsCountersAligned(t *testing.T) {
	engine, _, _, _, _ := newTestEngine()
	rule := &entity.AlertRule{
		Condition: &entity.AlertCondition{
			Operator: "or",
			Items: []entity.ConditionItem{
				{Metric: "ghost_metric", Compare: "gt", Value: 999999, Duration: 0},
				{Metric: "mem_rate", Compare: "gt", Value: 90, Duration: 600},
			},
		},
	}
	// 评估结果与条件项一一对应：第 0 项无法取值，第 1 项满足
	result := &service.EvalResult{
		Triggered: true,
		Evaluated: true,
		Items: []service.ConditionEval{
			{Metric: "ghost_metric", Satisfied: false, Err: "unknown metric"},
			{Metric: "mem_rate", CurrentValue: 95, Threshold: 90, Compare: "gt", Satisfied: true},
		},
	}
	if got := engine.getMaxDurationForResult(rule, result); got != 600 {
		t.Errorf("OR 条件应从满足项取 Duration，expected 600 got %d", got)
	}
	if metric, _, threshold := engine.getTriggeredInfo(result); metric != "mem_rate" || threshold != "mem_rate gt 90.00" {
		t.Errorf("触发信息应取自满足项, got metric=%s threshold=%s", metric, threshold)
	}
}

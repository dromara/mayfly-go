package application

import (
	"context"
	"fmt"
	"math"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/service"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/scheduler"
	"sync"
	"time"
)

// eventRetainDays 已终结事件（Recovered/Closed）保留天数，超过后由定时任务自动清理
const eventRetainDays = 30

// groupKey 分组键：rule_id + resource_id + 分组标签值，实现按标签分组
type groupKey struct {
	RuleId      uint64
	ResourceId  uint64
	GroupLabels string // 分组标签的序列化值，用于按标签分组
}

// pendingGroup 待发送的告警分组
type pendingGroup struct {
	Events     []*entity.AlertEvent
	Rule       *entity.AlertRule
	FirstFire  time.Time
	LastNotify time.Time
}

type AlertEngine interface {
	StartEvalLoop()
	Stop()
}

type alertEngineAppImpl struct {
	ruleApp          AlertRule                  `inject:"T"`
	eventApp         AlertEvent                 `inject:"T"`
	silenceApp       AlertSilence               `inject:"T"`
	escalationApp    AlertEscalation            `inject:"T"`
	inhibitionApp    AlertInhibition            `inject:"T"`
	notifyPolicyApp  AlertNotifyPolicy          `inject:"T"`
	notifier         service.AlertNotifier      `inject:"T"`
	tagTreeRelateApp tagapp.TagTreeRelate       `inject:"T"`
	tagTreeApp       tagapp.TagTree             `inject:"T"`
	breachTracker    service.BreachTracker      `inject:"T"`
	counter          service.ConsecutiveCounter `inject:"T"`

	// 分组聚合
	mu            sync.Mutex
	pendingGroups map[groupKey]*pendingGroup
	stopOnce      sync.Once

	// per-rule 评估时间追踪（支持 EvalInterval）
	evalMu       sync.Mutex
	lastEvalTime map[uint64]time.Time // ruleId -> last evaluation time

	// 评估并发限制：最多同时评估 10 条规则，防止 DB 连接池耗尽
	evalSem chan struct{}

	// 活跃事件缓存：每个评估周期刷新一次，避免 checkInhibition 重复查 DB
	activeEventsMu    sync.Mutex
	activeEventsCache []*entity.AlertEvent
	activeEventsAt    time.Time
}

var _ AlertEngine = (*alertEngineAppImpl)(nil)

func (a *alertEngineAppImpl) StartEvalLoop() {
	logx.Info("[alert] start alert evaluation loop")
	a.pendingGroups = make(map[groupKey]*pendingGroup)
	a.lastEvalTime = make(map[uint64]time.Time)
	a.evalSem = make(chan struct{}, 10) // 最多 10 个规则并发评估

	// 使用分布式锁调度，多实例部署时只有一个实例执行评估
	scheduler.AddFunByKeyWithLock("alert-eval", "@every 1m", 50*time.Second, func() {
		defer gox.Recover()
		a.evaluateAll()
	})

	// 分组刷新循环：使用分布式锁，每10秒检查一次待发送的分组
	scheduler.AddFunByKeyWithLock("alert-group-flush", "@every 10s", 8*time.Second, func() {
		defer gox.Recover()
		a.flushGroups()
	})

	// 已终结事件自动清理：每天凌晨执行，清理超过 30 天的 Recovered/Closed 事件
	scheduler.AddFunByKeyWithLock("alert-event-cleanup", "0 0 3 * * ?", 30*time.Minute, func() {
		defer gox.Recover()
		deleted, err := a.eventApp.CleanupTerminal(context.Background(), eventRetainDays)
		if err != nil {
			logx.Errorf("[alert] cleanup terminal events error: %s", err.Error())
		} else if deleted > 0 {
			logx.Infof("[alert] cleaned up %d terminal events (retained %d days)", deleted, eventRetainDays)
		}
	})
}

// Stop 优雅停止引擎，幂等安全
func (a *alertEngineAppImpl) Stop() {
	a.stopOnce.Do(func() {
		logx.Info("[alert] stopping alert evaluation loop")
		scheduler.RemoveByKey("alert-eval")
		scheduler.RemoveByKey("alert-group-flush")
		scheduler.RemoveByKey("alert-event-cleanup")
		// 同时停止升级扫描循环
		if a.escalationApp != nil {
			a.escalationApp.StopEscalationLoop()
		}
	})
}

func (a *alertEngineAppImpl) evaluateAll() {
	rules, err := a.ruleApp.ListEnabled()
	if err != nil {
		logx.Errorf("[alert] list enabled rules error: %s", err.Error())
		return
	}
	logx.Debugf("[alert] evaluateAll: found %d enabled rules", len(rules))

	now := time.Now()
	for _, rule := range rules {
		// 检查 EvalInterval：跳过未到评估时间的规则
		if !a.shouldEvaluate(rule, now) {
			continue
		}
		logx.Debugf("[alert] evaluateAll: evaluating rule[%d] name=%s", rule.Id, rule.Name)
		r := rule // capture loop variable
		// 信号量在 goroutine 外部获取，限制同时存在的 goroutine 数不超过信号量容量
		a.evalSem <- struct{}{}
		gox.Go(func() {
			defer func() { <-a.evalSem }() // 释放信号量
			a.evaluateRule(r)
		}, func(err error) {
			logx.ErrorTrace(fmt.Sprintf("[alert] evaluate rule[%d] error", r.Id), err)
		})
	}
}

// shouldEvaluate 检查规则是否到了评估时间（基于 EvalInterval）
func (a *alertEngineAppImpl) shouldEvaluate(rule *entity.AlertRule, now time.Time) bool {
	interval := rule.EvalInterval
	if interval <= 0 {
		interval = 60 // 默认60秒
	}

	a.evalMu.Lock()
	defer a.evalMu.Unlock()

	lastTime, exists := a.lastEvalTime[rule.Id]
	if !exists || now.Sub(lastTime) >= time.Duration(interval)*time.Second {
		a.lastEvalTime[rule.Id] = now
		return true
	}
	return false
}

func (a *alertEngineAppImpl) evaluateRule(rule *entity.AlertRule) {
	evaluator, ok := service.GetEvaluator(rule.ResourceType)
	if !ok {
		logx.Warnf("[alert] evaluateRule: no evaluator for resource type %d, rule[%d]", rule.ResourceType, rule.Id)
		return
	}

	ctx := context.Background()
	// 通过标签 code 查询资源 ID 的函数
	queryResourceIds := func(codes []string) []uint64 {
		return evaluator.ResourceIdsByCodes(codes)
	}
	resourceIds := expandResourceScope(ctx, a.tagTreeRelateApp, a.tagTreeApp, rule.ResourceType, rule.Id, rule.ScopeType, rule.ScopeValue, queryResourceIds)
	logx.Debugf("[alert] evaluateRule: rule[%d] expanded %d resources from scopeType=%d scopeValue=%s", rule.Id, len(resourceIds), rule.ScopeType, rule.ScopeValue)
	for _, rid := range resourceIds {
		// 静默检查
		if a.silenceApp.IsSilenced(ctx, rule, rid) {
			logx.Debugf("[alert] evaluateRule: rule[%d] resource[%d] is silenced", rule.Id, rid)
			continue
		}

		result, err := evaluator.Evaluate(ctx, &service.EvalParam{Rule: rule, ResourceId: rid})
		if err != nil {
			logx.Errorf("[alert] eval rule[%d] resource[%d] error: %s", rule.Id, rid, err.Error())
			continue
		}
		logx.Debugf("[alert] evaluateRule: rule[%d] resource[%d] evaluated=%v triggered=%v", rule.Id, rid, result.Evaluated, result.Triggered)
		// 本次判定不可靠（指标取不到且结论为未触发）：跳过状态迁移，
		// 既不能触发告警，也不能据此把进行中的告警误判为"已恢复"
		if !result.Evaluated {
			logx.Warnf("[alert] rule[%d] resource[%d] evaluation unreliable, skip state transition", rule.Id, rid)
			continue
		}
		a.handleEvalResult(ctx, rule, rid, evaluator, result)
	}
}

// breachCounterKey 生成越界计数器 key
func breachCounterKey(ruleId, resourceId uint64) string {
	return fmt.Sprintf("breach:%d:%d", ruleId, resourceId)
}

// recoverCounterKey 生成恢复计数器 key
func recoverCounterKey(ruleId, resourceId uint64) string {
	return fmt.Sprintf("recover:%d:%d", ruleId, resourceId)
}

// handleEvalResult 处理评估结果（告警状态机核心）
func (a *alertEngineAppImpl) handleEvalResult(ctx context.Context, rule *entity.AlertRule, rid uint64, evaluator service.AlertEvaluator, result *service.EvalResult) {
	breached := result.Triggered
	active, err := a.eventApp.FindActive(rule.Id, rid)
	if err != nil {
		logx.Errorf("[alert] find active event error: rule[%d] resource[%d] %s", rule.Id, rid, err.Error())
		return
	}

	if breached {
		// 越界：重置恢复计数器
		a.counter.Reset(recoverCounterKey(rule.Id, rid))

		// 新越界周期：记录首次越界时间并重置越界计数器。
		// 必须把首次越界时间同步到局部变量 firstBreach，否则下方 Duration 判断会拿
		// 零值时间做差（time.Since(零值) 为巨大正值），导致首次越界即满足 Duration，
		// 使持续时间防抖完全失效。
		firstBreach := a.breachTracker.GetFirstBreach(rule.Id, rid)
		if firstBreach.IsZero() {
			firstBreach = time.Now()
			a.counter.Reset(breachCounterKey(rule.Id, rid))
		}
		// 每个越界周期都续写首次越界时间以刷新缓存 TTL：
		// 若仅在活跃事件期间续期，Duration 大于 TTL 的规则会在等待中丢失首次越界时间
		// 并重新开始计时，导致告警永远无法满足 Duration 条件
		a.breachTracker.SetFirstBreach(rule.Id, rid, firstBreach)

		// 累加越界计数器（每次评估都递增，保证 Duration 和 TriggerCount 独立判断）
		consecutiveBreaches := a.counter.Incr(breachCounterKey(rule.Id, rid))

		// Duration 防抖：持续越界时间需达到阈值
		maxDuration := a.getMaxDurationForResult(rule, result)
		if maxDuration > 0 && time.Since(firstBreach) < time.Duration(maxDuration)*time.Second {
			return
		}

		// TriggerCount 防抖：连续越界次数需达到阈值
		triggerCount := rule.TriggerCount
		if triggerCount <= 0 {
			triggerCount = 1
		}
		if consecutiveBreaches < int64(triggerCount) {
			return
		}

		metric, currentValue, threshold := a.getTriggeredInfo(result)
		resourceName := evaluator.ResourceName(rid)

		if active == nil {
			// 原子创建，防止并发重复创建
			created, exists, err := a.eventApp.CreateFiringEvent(ctx, rule, rid, resourceName, metric, currentValue, threshold)
			if err != nil {
				logx.Errorf("[alert] create firing event error: %s", err.Error())
				return
			}
			if exists {
				// 已有活跃事件（并发竞争），直接触摸更新
				if err := a.eventApp.TouchActiveEvent(ctx, created, metric, currentValue); err != nil {
					logx.Errorf("[alert] touch active event error: %s", err.Error())
				}
				// 如果事件还没被通知过，添加到分组等待通知
				if created.NotifyCount == 0 {
					a.addToGroup(rule, created)
				}
			} else if created != nil {
				a.addToGroup(rule, created)
			}
		} else {
			// 资源名为后补字段，历史活跃事件可能为空：在触摸更新前回填，复用同一次写库
			if active.ResourceName == "" {
				active.ResourceName = resourceName
			}
			if err := a.eventApp.TouchActiveEvent(ctx, active, metric, currentValue); err != nil {
				logx.Errorf("[alert] touch active event error: %s", err.Error())
			}
			// 如果事件还没被通知过，添加到分组等待通知
			if active.NotifyCount == 0 {
				a.addToGroup(rule, active)
			}
		}
	} else {
		// 未越界：重置越界计数器和 Duration 追踪
		a.breachTracker.DelFirstBreach(rule.Id, rid)
		a.counter.Reset(breachCounterKey(rule.Id, rid))

		if active != nil {
			// RecoveryCount 防抖：连续恢复次数需达到阈值
			recoveryCount := rule.RecoveryCount
			if recoveryCount <= 0 {
				recoveryCount = 1
			}
			consecutiveRecovers := a.counter.Incr(recoverCounterKey(rule.Id, rid))
			if consecutiveRecovers < int64(recoveryCount) {
				return
			}

			// 恢复计数器复位
			a.counter.Reset(recoverCounterKey(rule.Id, rid))

			if err := a.eventApp.Recover(ctx, active); err != nil {
				logx.Errorf("[alert] recover event error: %s", err.Error())
				return
			}
			// 恢复通知发送（带超时控制，走路由确保与触发通知相同目标）
			notifyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			recoverTarget := a.resolveNotifyTarget(rule, active)
			if _, err := a.notifier.NotifyRecoverWithTarget(notifyCtx, active, rule, recoverTarget); err != nil {
				logx.Errorf("[alert] notify recover error: %s", err.Error())
			}
		}
	}
}

// checkInhibition 检查事件是否被其他活跃告警抑制（P2）
// 遍历所有活跃告警事件，检查是否存在抑制关系。
// 使用评估周期级缓存（60s TTL），避免同一周期内重复查 DB。
func (a *alertEngineAppImpl) checkInhibition(event *entity.AlertEvent) bool {
	activeEvents := a.getActiveFiringEvents()
	if len(activeEvents) == 0 {
		return false
	}

	for _, sourceEvent := range activeEvents {
		if sourceEvent.Id == event.Id {
			continue
		}
		if a.inhibitionApp.IsInhibited(sourceEvent, event) {
			logx.Infof("[alert] event[%d] inhibited by event[%d] rule[%d]", event.Id, sourceEvent.Id, sourceEvent.RuleId)
			return true
		}
	}
	return false
}

// getActiveFiringEvents 获取活跃告警事件列表（带 60s TTL 缓存）
func (a *alertEngineAppImpl) getActiveFiringEvents() []*entity.AlertEvent {
	a.activeEventsMu.Lock()
	defer a.activeEventsMu.Unlock()

	if a.activeEventsCache != nil && time.Since(a.activeEventsAt) < 60*time.Second {
		return a.activeEventsCache
	}

	events, err := a.eventApp.ListActiveFiring()
	if err != nil {
		logx.Errorf("[alert] list active firing events error: %s", err.Error())
		return nil
	}
	a.activeEventsCache = events
	a.activeEventsAt = time.Now()
	return events
}

// addToGroup 将告警事件加入分组缓冲（按事件 ID 去重，避免同一事件被重复通知）
func (a *alertEngineAppImpl) addToGroup(rule *entity.AlertRule, event *entity.AlertEvent) {
	// 抑制检查（P2）：如果事件被其他活跃告警抑制，则不加入分组
	if a.checkInhibition(event) {
		logx.Infof("[alert] event[%d] rule[%d] suppressed by inhibition rule, skip notify", event.Id, rule.Id)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	key := a.computeGroupKey(rule, event)
	group, ok := a.pendingGroups[key]
	if !ok {
		group = &pendingGroup{
			Rule:      rule,
			FirstFire: time.Now(),
		}
		a.pendingGroups[key] = group
	}
	// 按事件 ID 去重：同一事件只保留一份，避免 flushGroups 重复发送通知
	for _, e := range group.Events {
		if e.Id == event.Id {
			return
		}
	}
	group.Events = append(group.Events, event)
}

// computeGroupKey 计算分组键：按 rule_id + resource_id 分组。
// 分组标签（GroupByLabels）已迁移至通知策略层，此处不再处理。
func (a *alertEngineAppImpl) computeGroupKey(rule *entity.AlertRule, event *entity.AlertEvent) groupKey {
	return groupKey{
		RuleId:     rule.Id,
		ResourceId: event.ResourceId,
	}
}

// flushGroups 刷新分组缓冲，发送满足条件的通知
func (a *alertEngineAppImpl) flushGroups() {
	a.mu.Lock()
	// 收集需要发送的分组，释放锁后再发送，避免阻塞其他操作
	var toNotify []*pendingGroup

	now := time.Now()
	logx.Debugf("[alert] flushGroups: checking %d pending groups", len(a.pendingGroups))
	for key, group := range a.pendingGroups {
		groupWait := a.getGroupWait(group.Rule)
		groupInterval := a.getGroupInterval(group.Rule)

		shouldNotify := false
		if group.LastNotify.IsZero() {
			// 首次通知：等待 GroupWait
			elapsed := now.Sub(group.FirstFire)
			if elapsed >= time.Duration(groupWait)*time.Second {
				shouldNotify = true
				logx.Debugf("[alert] flushGroups: group[rule=%d,res=%d] first fire elapsed %v >= groupWait %ds, should notify", key.RuleId, key.ResourceId, elapsed, groupWait)
			} else {
				logx.Debugf("[alert] flushGroups: group[rule=%d,res=%d] first fire elapsed %v < groupWait %ds, waiting", key.RuleId, key.ResourceId, elapsed, groupWait)
			}
		} else {
			// 后续通知：等待 GroupInterval
			elapsed := now.Sub(group.LastNotify)
			if elapsed >= time.Duration(groupInterval)*time.Second {
				shouldNotify = true
				logx.Debugf("[alert] flushGroups: group[rule=%d,res=%d] last notify elapsed %v >= groupInterval %ds, should notify", key.RuleId, key.ResourceId, elapsed, groupInterval)
			} else {
				logx.Debugf("[alert] flushGroups: group[rule=%d,res=%d] last notify elapsed %v < groupInterval %ds, waiting", key.RuleId, key.ResourceId, elapsed, groupInterval)
			}
		}

		if shouldNotify {
			toNotify = append(toNotify, group)
			// 注意：不在这里更新 group.LastNotify，等 sendGroupNotification 实际发送后再更新
			// 避免事件因 RepeatInterval 被跳过时，group.LastNotify 已更新导致下次等待过久
		}

		// 清理已恢复或已关闭的事件
		var remaining []*entity.AlertEvent
		for _, e := range group.Events {
			if e.Status == entity.AlertEventStatusFiring || e.Status == entity.AlertEventStatusAcknowledged {
				remaining = append(remaining, e)
			}
		}
		group.Events = remaining

		if len(group.Events) == 0 {
			delete(a.pendingGroups, key)
		}
	}
	a.mu.Unlock()

	logx.Debugf("[alert] flushGroups: %d groups to notify", len(toNotify))
	// 在锁外发送通知
	for _, group := range toNotify {
		a.sendGroupNotification(group)
	}
}

// sendGroupNotification 发送分组通知
func (a *alertEngineAppImpl) sendGroupNotification(group *pendingGroup) {
	if len(group.Events) == 0 {
		return
	}
	repeatInterval := a.getRepeatInterval(group.Rule)
	now := time.Now()
	notified := false // 跟踪是否有事件实际发送成功

	for _, event := range group.Events {
		// 跳过已恢复或已关闭的事件（锁外竞态保护）
		if event.Status != entity.AlertEventStatusFiring && event.Status != entity.AlertEventStatusAcknowledged {
			continue
		}

		// RepeatInterval 限速：距上次通知未达间隔则跳过
		if event.NotifyCount > 0 && event.LastNotifyTime != nil {
			if now.Sub(*event.LastNotifyTime) < time.Duration(repeatInterval)*time.Second {
				continue
			}
		}

		// 解析通知目标（支持路由 P0）
		target := a.resolveNotifyTarget(group.Rule, event)

		notifyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		sent, err := a.notifier.NotifyWithTarget(notifyCtx, event, group.Rule, target)
		cancel()
		if err != nil {
			logx.Errorf("[alert] group notify error: event[%d] %s", event.Id, err.Error())
			continue
		}
		if sent == 0 {
			// 无任何可用渠道：既不能递增 NotifyCount（统计虚高），
			// 也不能更新 LastNotifyTime，否则 RepeatInterval 窗口会被一次空发送消费掉
			logx.Warnf("[alert] event[%d] skipped: rule[%d] has no available channel", event.Id, group.Rule.Id)
			continue
		}
		// 已投递给渠道，递增 NotifyCount 并记录通知时间
		event.NotifyCount++
		event.LastNotifyTime = &now
		if err := a.eventApp.UpdateById(context.Background(), event); err != nil {
			logx.Errorf("[alert] update event notify info error: event[%d] %s", event.Id, err.Error())
		}
		notified = true
	}

	// 只有实际发送了通知才更新 group.LastNotify，避免事件被跳过时 group 等待过久
	if notified {
		group.LastNotify = now
	}
}

// resolveNotifyTarget 通过通知策略解析通知目标。
// 查找所有标签匹配事件的通知策略，合并所有匹配策略的渠道和接收人。
// 无匹配策略时返回空目标（不发送通知）。
func (a *alertEngineAppImpl) resolveNotifyTarget(rule *entity.AlertRule, event *entity.AlertEvent) service.NotifyTarget {
	eventLabels := parseRuleLabels(event.Labels)

	policies, err := a.notifyPolicyApp.MatchPolicies(context.Background(), eventLabels)
	if err != nil {
		logx.Errorf("[alert] resolve notify target: match policies error: %s", err.Error())
		return service.NotifyTarget{}
	}
	if len(policies) == 0 {
		return service.NotifyTarget{}
	}

	// 合并所有匹配策略的渠道和接收人（去重）
	channelSet := make(map[uint64]struct{})
	receiverSet := make(map[int64]struct{})
	for _, p := range policies {
		for _, ch := range p.ChannelIds {
			channelSet[ch] = struct{}{}
		}
		for _, r := range p.ReceiverIds {
			receiverSet[r] = struct{}{}
		}
	}

	channelIds := make([]uint64, 0, len(channelSet))
	for ch := range channelSet {
		channelIds = append(channelIds, ch)
	}
	receiverIds := make([]int64, 0, len(receiverSet))
	for r := range receiverSet {
		receiverIds = append(receiverIds, r)
	}

	return service.NotifyTarget{
		ChannelIds:  channelIds,
		ReceiverIds: receiverIds,
	}
}

func (a *alertEngineAppImpl) getGroupWait(rule *entity.AlertRule) int {
	if rule.NotifyConfig != nil && rule.NotifyConfig.GroupWait > 0 {
		return rule.NotifyConfig.GroupWait
	}
	return 30 // 默认30秒
}

func (a *alertEngineAppImpl) getGroupInterval(rule *entity.AlertRule) int {
	if rule.NotifyConfig != nil && rule.NotifyConfig.GroupInterval > 0 {
		return rule.NotifyConfig.GroupInterval
	}
	return 300 // 默认5分钟
}

func (a *alertEngineAppImpl) getRepeatInterval(rule *entity.AlertRule) int {
	if rule.NotifyConfig != nil && rule.NotifyConfig.RepeatInterval > 0 {
		return rule.NotifyConfig.RepeatInterval
	}
	return 3600 // 默认1小时
}

// getMaxDurationForResult 根据评估结果确定应使用的 Duration。
// AND 条件：取所有条件项的最大 Duration（所有条件都要满足才触发）。
// OR 条件：取已满足条件项的最小 Duration（任一条件满足即触发，等待时间最短的条件先触发）。
func (a *alertEngineAppImpl) getMaxDurationForResult(rule *entity.AlertRule, result *service.EvalResult) int {
	if rule.Condition == nil {
		return 0
	}
	if rule.Condition.Operator == "or" {
		minD := math.MaxInt
		hasSatisfied := false
		for i, item := range rule.Condition.Items {
			if i < len(result.Items) && result.Items[i].Satisfied {
				hasSatisfied = true
				if item.Duration < minD {
					minD = item.Duration
				}
			}
		}
		if !hasSatisfied {
			return 0
		}
		return minD
	}
	// AND（默认）：所有条件项都要满足，取最大 Duration（所有条件都要达到时间门槛）
	maxDuration := 0
	for _, item := range rule.Condition.Items {
		if item.Duration > maxDuration {
			maxDuration = item.Duration
		}
	}
	return maxDuration
}

func (a *alertEngineAppImpl) getTriggeredInfo(result *service.EvalResult) (string, float64, string) {
	for _, item := range result.Items {
		if item.Satisfied {
			threshold := fmt.Sprintf("%s %s %.2f", item.Metric, item.Compare, item.Threshold)
			return item.Metric, item.CurrentValue, threshold
		}
	}
	return "", 0, ""
}

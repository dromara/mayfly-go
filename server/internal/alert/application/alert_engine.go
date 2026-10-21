package application

import (
	"context"
	"fmt"
	"math"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/service"
	pkgconsts "mayfly-go/internal/pkg/consts"
	opsevent "mayfly-go/internal/pkg/event"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/taskx"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// eventRetainDays 已终结事件（Recovered/Closed）保留天数，超过后由定时任务自动清理
const eventRetainDays = 30

// 评估调度粒度。
//
// shouldEvaluate 只能在调度点上判定「规则是否到期」，永远无法比调度更密。因此调度粒度必须
// 不粗于规则允许的最小评估间隔（minEvalInterval 直接取自它），否则用户填的更小间隔会被
// 静默拉长到调度粒度——界面写着「每隔 N 秒评估一次」、实际每 60 秒才评估一次，
// 配置项就成了摆设。改动本组常量时必须同步核对 minEvalInterval 与前端校验下限
const (
	evalTickSeconds = 10
	evalTickCron    = "@every 10s"
	// 锁 TTL 必须覆盖单轮最坏耗时：基座契约就写在这两个地方（pkg/taskx/cron.go:28、
	// pkg/scheduler/scheduler.go:110）——「lockDuration 应大于 run 最大执行时间」。
	//
	// 锁在 run 结束时由 `defer lock.UnLock()` 显式释放（scheduler.go:123），所以把 TTL 拉长**不会**
	// 挡住本实例或别的实例的下一个触发点；反过来 TTL 短于单轮耗时才是真事故：一轮跑到一半锁先过期，
	// 另一实例在下一个 10s 点抢到同一把锁并发跑同一轮，而 lastEvalTime 与 pendingGroups 都是
	// **进程内**状态（各自一份、不共享），于是同一告警被评估两遍、通知也发两遍。
	// TTL 同时是「实例崩溃后全集群停摆多久」的上限，所以取够覆盖单轮、又不长期占位的值
	evalTickLockDuration = 60 * time.Second

	// flush 锁同一个契约：flushGroups 里要真发 IM/邮件（含 30s 发送超时），比评估更容易超过一个调度点
	groupFlushLockDuration = 60 * time.Second
)

// groupKey 分组键：rule_id + resource_id。
// 按标签分组已迁移至通知策略层（AlertNotifyPolicy 标签路由），分组维度收敛为规则+资源
type groupKey struct {
	RuleId     uint64
	ResourceId uint64
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

	// 评估自旋守卫：调度粒度可能细于单次 evaluateAll 耗时（evalSem 满时派发循环会阻塞），
	// 而 cron 不会跳过仍在运行的任务，无守卫会让多轮评估叠加、重复扫描规则表
	evalRunning atomic.Bool

	// 活跃事件缓存：每个评估周期刷新一次，避免 checkInhibition 重复查 DB
	activeEventsMu    sync.Mutex
	activeEventsCache []*entity.AlertEvent
	activeEventsAt    time.Time

	// 通知策略缓存（周期级 TTL）：逐次通知匹配策略需两次全量查库（策略列表 + 标签绑定），
	// 缓存后单周期只查一次；查询失败时沿用上次缓存，避免 DB 抖动中断通知路由
	policyMu      sync.Mutex
	policyCache   []*entity.AlertNotifyPolicy
	policyCacheAt time.Time
}

// policyCacheTTL 通知策略缓存时长，与活跃事件缓存一致；策略变更最长该时长后生效
const policyCacheTTL = 60 * time.Second

var _ AlertEngine = (*alertEngineAppImpl)(nil)

func (a *alertEngineAppImpl) StartEvalLoop() {
	logx.Info("[alert] start alert evaluation loop")
	a.pendingGroups = make(map[groupKey]*pendingGroup)
	a.lastEvalTime = make(map[uint64]time.Time)
	a.evalSem = make(chan struct{}, 10) // 最多 10 个规则并发评估

	// 使用分布式锁调度，多实例部署时只有一个实例执行评估
	_ = taskx.BindCronTaskWithLock("alert-eval", evalTickCron, evalTickLockDuration, true, func() {
		defer gox.Recover()
		a.evaluateAll()
	})

	// 分组刷新循环：使用分布式锁，每10秒检查一次待发送的分组
	_ = taskx.BindCronTaskWithLock("alert-group-flush", "@every 10s", groupFlushLockDuration, true, func() {
		defer gox.Recover()
		a.flushGroups()
	})

	// 已终结事件自动清理：每天凌晨执行，清理超过 30 天的 Recovered/Closed 事件
	_ = taskx.BindCronTaskWithLock("alert-event-cleanup", "0 0 3 * * ?", 30*time.Minute, true, func() {
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
		taskx.UnbindCronTask("alert-eval")
		taskx.UnbindCronTask("alert-group-flush")
		taskx.UnbindCronTask("alert-event-cleanup")
		// 同时停止升级扫描循环
		if a.escalationApp != nil {
			a.escalationApp.StopEscalationLoop()
		}
	})
}

func (a *alertEngineAppImpl) evaluateAll() {
	// 上一轮未跑完就跳过本轮：到期的规则不会丢，下一轮（一个调度粒度之后）仍会被 shouldEvaluate 放行
	if !a.evalRunning.CompareAndSwap(false, true) {
		logx.Debugf("[alert] evaluateAll skipped: previous round still running")
		return
	}
	defer a.evalRunning.Store(false)

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
	active, err := a.eventApp.FindActive(ctx, rule.Id, rid)
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
				a.addToGroup(rule, created)
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
			// 不区分是否已通知过：分组缓冲是**纯进程内内存**，重启即清零，旧写法用
			// 「NotifyCount == 0」当门，导致重启后所有已通知过的活跃事件再也进不了分组，
			// 重复告警通知从此静默停摆（真机实测：规则配了 repeatInterval/groupInterval=300，
			// 事件持续越界 50 分钟、trigger_count 从 19 跑到 33，notify_count 始终为 1）。
			// 重复节奏由已持久化的 LastNotifyTime 播种（见 addToGroup），因此补入组不会造成重启即轰炸
			a.addToGroup(rule, active)
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
			FirstFire: event.FirstTriggerTime,
		}
		// 用事件里已持久化的时间戳播种分组时钟：进程重启后把旧事件重新纳入分组时，
		// 间隔从「它原来进到的位置」继续算，而不是当成首次通知立刻发一轮，
		// 也不会把 groupWait 从头重算。零值（历史脏数据可缺）才回退到当前时间
		if event.LastNotifyTime != nil && !event.LastNotifyTime.IsZero() {
			group.LastNotify = *event.LastNotifyTime
		}
		if group.FirstFire.IsZero() {
			group.FirstFire = time.Now()
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

		// 重复通知的两道闸门（仅对已通知过的事件生效）：
		//  1. RepeatInterval 限速：距上次通知未达间隔则跳过；
		//  2. 新鲜度校验：上次通知之后指标未被重新确认为越限（LastTriggerTime 未前移）则跳过。
		// 第 2 道闸门用于阻断"陈旧告警刷屏"：当指标采集持续不可靠（Evaluated=false）时，
		// 引擎会跳过状态迁移，事件长期停留在「告警中」却没有任何新的越限证据，
		// 此时分组刷新循环不应再反复推送通知，直到重新采到越限数据或确认恢复。
		if event.NotifyCount > 0 && event.LastNotifyTime != nil {
			if now.Sub(*event.LastNotifyTime) < time.Duration(repeatInterval)*time.Second {
				continue
			}
			if !event.LastTriggerTime.After(*event.LastNotifyTime) {
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
		// 已投递给渠道，递增 NotifyCount 并记录通知时间。
		// 必须走专用列更新：group 内 event 是旧快照，全量写回会覆盖评估链路
		// 并发写入的 CurrentValue/LastTriggerTime（破坏新鲜度闸门）
		event.NotifyCount++
		event.LastNotifyTime = &now
		if err := a.eventApp.UpdateNotifyInfo(context.Background(), event); err != nil {
			logx.Errorf("[alert] update event notify info error: event[%d] %s", event.Id, err.Error())
		}
		notified = true

		// 通知成功后发布中立运维触发事件（fail-open）：agentops 订阅后自动派生只读诊断 Run，
		// 发布异常/无订阅者都不影响告警主链路。维持依赖不变式——alert 不依赖 ai/agentops，
		// 仅经 internal/pkg/event 中立契约 + global.EventBus 与之解耦通信。
		a.publishOpsTrigger(event)
	}

	// 只有实际发送了通知才更新 group.LastNotify，避免事件被跳过时 group 等待过久
	if notified {
		group.LastNotify = now
	}
}

// publishOpsTrigger 把已成功通知的告警事件归一化为中立 OpsTriggerEvent 并发布到全局事件总线。
//
// fail-open：Publish 本身无返回值、订阅方（agentops）异步消费；此处再以 recover 兜底，确保
// 归一化/发布的任何异常都不会回流影响告警通知主链路。维持依赖不变式——alert 不依赖
// ai/agentops，仅经 internal/pkg/event 中立契约 + global.EventBus 与之解耦通信。
func (a *alertEngineAppImpl) publishOpsTrigger(ev *entity.AlertEvent) {
	// 恢复语义统一走 gox.Recover（本包其余异步派发同此）：自写 recover 会让「是否记日志、
	// 日志格式」在各调用点漂移；回调里补上事件 id，否则只能看到一条笼统的 panic
	defer gox.Recover(func(err error) {
		logx.Errorf("[alert] publish ops trigger event failed (ignored): event[%d] %v", ev.Id, err)
	})

	// OccurredAt 取最近触发时间：同一事件再次越限（LastTriggerTime 前移）即视为新事件、可再派生，
	// 未前移则被上游新鲜度闸门拦下不会重复通知（也就不会重复发布）
	occurredAt := ev.LastTriggerTime.Unix()
	if ev.LastTriggerTime.IsZero() {
		occurredAt = time.Now().Unix()
	}
	evt := &opsevent.OpsTriggerEvent{
		Source:       opsevent.OpsTriggerSourceAlert,
		SourceRef:    strconv.FormatUint(ev.Id, 10),
		ResourceType: opsTriggerResourceType(ev.ResourceType),
		ResourceId:   ev.ResourceId,
		ResourceName: ev.ResourceName,
		Severity:     ev.Priority.Code(),
		Metric:       ev.Metric,
		Value:        ev.CurrentValue,
		OccurredAt:   occurredAt,
		Payload: map[string]any{
			"ruleId":       ev.RuleId,
			"ruleName":     ev.RuleName,
			"threshold":    ev.Threshold,
			"labels":       ev.Labels,
			"status":       int8(ev.Status),
			"triggerCount": ev.TriggerCount,
		},
	}
	global.EventBus.Publish(context.Background(), opsevent.EventTopicOpsTrigger, evt)
	logx.Debugf("[alert] published ops trigger event: event[%d] resource=%s/%d severity=%s", ev.Id, evt.ResourceType, ev.ResourceId, evt.Severity)
}

// opsTriggerResourceType 把告警资源类型枚举映射为中立事件里的资源类型字符串（供触发器匹配/诊断上下文）。
//
// 键名唯一真源在 consts.resourceTypeKeys（与 AI 工具入参说明、资源可见面判定同源），本函数不再自写 switch。
// 未登记键名的类型（凭证/库名等非资源主体）回退为数值串，保证信息不丢失；新增「可被触发器按类型
// 匹配」的资源类型时必须去那张表登记，否则这里的数字串与触发器条件里的类型名永远对不上
func opsTriggerResourceType(rt int8) string {
	if key, ok := pkgconsts.ResourceTypeKey(rt); ok {
		return key
	}
	return strconv.Itoa(int(rt))
}

// listPoliciesWithLabels 获取启用通知策略列表（带 TTL 缓存，已回填各策略 MatchLabels）。
// 缓存消除了逐次通知的两轮全量查库；查询失败时沿用上次缓存，DB 抖动不中断通知路由
func (a *alertEngineAppImpl) listPoliciesWithLabels() []*entity.AlertNotifyPolicy {
	a.policyMu.Lock()
	defer a.policyMu.Unlock()

	if a.policyCache != nil && time.Since(a.policyCacheAt) < policyCacheTTL {
		return a.policyCache
	}

	policies, err := a.notifyPolicyApp.ListEnabledWithLabels(context.Background())
	if err != nil {
		logx.Errorf("[alert] list notify policies error: %s", err.Error())
		return a.policyCache
	}
	// 按 Id 升序固定策略遍历顺序：合并去重与「首个声明者归因」的确定性
	// 不能依赖 SQL 未定义的返回行序
	sort.Slice(policies, func(i, j int) bool { return policies[i].Id < policies[j].Id })
	a.policyCache = policies
	a.policyCacheAt = time.Now()
	return policies
}

// resolveNotifyTarget 通过通知策略解析通知目标。
// 对缓存中的启用策略做标签子集匹配，合并所有命中策略的渠道和接收人（去重，
// 按策略声明顺序保持确定性），并登记渠道→策略归因供通知日志使用。
// 无命中策略时返回空目标（不发送通知）。
func (a *alertEngineAppImpl) resolveNotifyTarget(rule *entity.AlertRule, event *entity.AlertEvent) service.NotifyTarget {
	eventLabels := parseRuleLabels(event.Labels)

	target := service.NotifyTarget{}
	channelSeen := make(map[uint64]struct{})
	receiverSeen := make(map[int64]struct{})
	for _, p := range a.listPoliciesWithLabels() {
		if !matchRuleLabelsMap(parseRuleLabels(p.MatchLabels), eventLabels) {
			continue
		}
		// 按策略声明顺序去重追加，保证合并结果确定性
		for _, ch := range p.ChannelIds {
			if _, ok := channelSeen[ch]; !ok {
				channelSeen[ch] = struct{}{}
				target.ChannelIds = append(target.ChannelIds, ch)
			}
		}
		for _, r := range p.ReceiverIds {
			if _, ok := receiverSeen[r]; !ok {
				receiverSeen[r] = struct{}{}
				target.ReceiverIds = append(target.ReceiverIds, r)
			}
		}
		// 渠道→策略归因：多策略命中同一渠道时保留首个声明者
		target = target.WithNotifyPolicyBinding(p.Id, p.ChannelIds)
	}
	if len(target.ChannelIds) == 0 && len(target.ReceiverIds) == 0 {
		return service.NotifyTarget{}
	}
	return target
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

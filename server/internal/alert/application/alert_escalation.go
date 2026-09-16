package application

import (
	"context"
	"encoding/json"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/repository"
	"mayfly-go/internal/alert/domain/service"
	"mayfly-go/internal/alert/imsg"
	labelapp "mayfly-go/internal/label/application"
	labelentity "mayfly-go/internal/label/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/scheduler"
	"sort"
	"strings"
	"sync"
	"time"
)

type AlertEscalation interface {
	base.App[*entity.AlertEscalation]

	SaveAlertEscalation(ctx context.Context, escalation *entity.AlertEscalation) error
	GetAlertEscalationList(condition *entity.AlertEscalationQuery, orderBy ...string) (*model.PageResult[*entity.AlertEscalation], error)
	DeleteEscalation(ctx context.Context, id uint64) error
	StartEscalationLoop()
	StopEscalationLoop()
}

type alertEscalationAppImpl struct {
	base.AppImpl[*entity.AlertEscalation, repository.AlertEscalation]

	eventApp        AlertEvent            `inject:"T"`
	notifier        service.AlertNotifier `inject:"T"`
	ruleApp         AlertRule             `inject:"T"`
	labelBindingApp labelapp.LabelBinding `inject:"T"`

	// 事件级锁，防止多个升级策略并发修改同一事件的 EscalationLvl
	eventMu sync.Map // map[uint64]*sync.Mutex

	// 升级扫描并发限制
	escSem chan struct{}
}

var _ AlertEscalation = (*alertEscalationAppImpl)(nil)

func (a *alertEscalationAppImpl) SaveAlertEscalation(ctx context.Context, escalation *entity.AlertEscalation) error {
	if err := validateAlertEscalation(ctx, escalation); err != nil {
		return err
	}
	// 保存前按 DelayMinutes 递增排序，确保升级链有序
	sort.Slice(escalation.Rules, func(i, j int) bool {
		return escalation.Rules[i].DelayMinutes < escalation.Rules[j].DelayMinutes
	})

	// 提取 matchLabels 用于保存标签绑定，然后清空实体中的瞬态字段
	matchLabelsJSON := escalation.MatchLabels
	escalation.MatchLabels = ""

	if escalation.Id == 0 {
		if err := a.Insert(ctx, escalation); err != nil {
			return err
		}
	} else {
		if err := a.UpdateById(ctx, escalation); err != nil {
			return err
		}
	}

	// 仅当 MatchLabels 显式提供时才更新标签绑定（避免 ChangeStatus 等操作误清空）
	if matchLabelsJSON != "" {
		if matchLabelsJSON != "{}" {
			var labels map[string]string
			if err := json.Unmarshal([]byte(matchLabelsJSON), &labels); err == nil && len(labels) > 0 {
				if err := a.labelBindingApp.SaveBindingsByLabels(ctx, labelentity.LabelTargetAlertEscalation, escalation.Id, labels); err != nil {
					return err
				}
			}
		} else {
			if err := a.labelBindingApp.DeleteByTarget(ctx, labelentity.LabelTargetAlertEscalation, escalation.Id); err != nil {
				return err
			}
		}
	}

	// 回填 MatchLabels 以便 API 响应包含标签信息
	escalation.MatchLabels = matchLabelsJSON
	return nil
}

func (a *alertEscalationAppImpl) DeleteEscalation(ctx context.Context, id uint64) error {
	if _, err := a.GetById(id); err != nil {
		return errorx.NewBizI(ctx, imsg.ErrEscalationNotFound, "id", id)
	}
	// 清理标签绑定记录
	if err := a.labelBindingApp.DeleteByTarget(ctx, labelentity.LabelTargetAlertEscalation, id); err != nil {
		return err
	}
	return a.DeleteById(ctx, id)
}

// validateAlertEscalation 升级策略保存校验。
//
// 延迟分钟数为 0 会让第一级升级在事件创建后的首次扫描立即触发（等于跳过升级语义）；
// 某级别既无渠道又无接收人则该级别永远不会产生通知，但升级级别仍会推进，
// 导致后续级别被误判为"已升级过"，因此两者都必须拒绝。
func validateAlertEscalation(ctx context.Context, escalation *entity.AlertEscalation) error {
	if strings.TrimSpace(escalation.Name) == "" {
		return errorx.NewBizI(ctx, imsg.ErrEscalationNameRequired)
	}
	if escalation.Status != entity.AlertEscalationStatusEnable && escalation.Status != entity.AlertEscalationStatusDisable {
		return errorx.NewBizI(ctx, imsg.ErrRuleStatusInvalid)
	}
	if len(escalation.Rules) == 0 {
		return errorx.NewBizI(ctx, imsg.ErrEscalationRulesRequired)
	}
	for i, rule := range escalation.Rules {
		index := i + 1
		if rule.DelayMinutes <= 0 {
			return errorx.NewBizI(ctx, imsg.ErrEscalationRuleDelayInvalid, "index", index)
		}
		if len(rule.ChannelIds) == 0 && len(rule.ReceiverIds) == 0 {
			return errorx.NewBizI(ctx, imsg.ErrEscalationRuleReceiverRequired, "index", index)
		}
	}
	if escalation.MatchLabels != "" && escalation.MatchLabels != "{}" {
		var labels map[string]string
		if err := json.Unmarshal([]byte(escalation.MatchLabels), &labels); err != nil {
			return errorx.NewBizI(ctx, imsg.ErrEscalationMatchLabelsInvalid)
		}
	}
	return nil
}

func (a *alertEscalationAppImpl) GetAlertEscalationList(condition *entity.AlertEscalationQuery, orderBy ...string) (*model.PageResult[*entity.AlertEscalation], error) {
	return a.GetRepo().GetAlertEscalationList(condition, orderBy...)
}

func (a *alertEscalationAppImpl) StartEscalationLoop() {
	logx.Info("[alert] start escalation scan loop")
	// 初始化升级扫描信号量（最多同时扫描 5 个策略）
	if a.escSem == nil {
		a.escSem = make(chan struct{}, 5)
	}
	// 使用分布式锁调度，多实例部署时只有一个实例执行升级扫描
	scheduler.AddFunByKeyWithLock("alert-escalation", "@every 1m", 50*time.Second, func() {
		defer gox.Recover()
		a.scanEscalations()
	})
}

func (a *alertEscalationAppImpl) StopEscalationLoop() {
	scheduler.RemoveByKey("alert-escalation")
}

func (a *alertEscalationAppImpl) scanEscalations() {
	escalations, err := a.GetRepo().ListEnabled()
	if err != nil {
		logx.Errorf("[alert] list escalations error: %s", err.Error())
		return
	}
	if len(escalations) == 0 {
		return
	}

	// 一次性加载所有启用规则（避免每个策略重复查询）
	rules, err := a.ruleApp.ListEnabled()
	if err != nil {
		logx.Errorf("[alert] scan escalations list rules error: %s", err.Error())
		return
	}

	// 批量加载所有升级策略的标签绑定（一次查询代替 N 次）
	escIds := make([]uint64, len(escalations))
	for i, e := range escalations {
		escIds[i] = e.Id
	}
	allLabels, err := a.labelBindingApp.ListByTargets(context.Background(), labelentity.LabelTargetAlertEscalation, escIds)
	if err != nil {
		logx.Errorf("[alert] scan escalations load labels error: %s", err.Error())
		return
	}

	for _, esc := range escalations {
		e := esc // capture loop variable
		escLabels := bindingsToMap(allLabels[e.Id])
		a.escSem <- struct{}{}
		gox.Go(func() {
			defer func() { <-a.escSem }()
			a.processEscalation(e, escLabels, rules)
		}, func(err error) {
			logx.ErrorTrace("[alert] process escalation error", err)
		})
	}
}

func (a *alertEscalationAppImpl) processEscalation(esc *entity.AlertEscalation, escLabels map[string]string, rules []*entity.AlertRule) {
	for _, rule := range rules {
		// 标签匹配：升级策略的 matchLabels 为空则匹配所有规则，非空则要求规则标签包含升级策略的所有标签
		if !matchRuleLabelsMap(escLabels, parseRuleLabels(rule.Labels)) {
			continue
		}

		activeEvents, err := a.eventApp.ListByCond(model.NewCond().
			Eq("rule_id", rule.Id).
			In("status", []entity.AlertEventStatus{
				entity.AlertEventStatusFiring,
				entity.AlertEventStatusAcknowledged,
			}))
		if err != nil {
			logx.Errorf("[alert] escalation list events error: rule[%d] %s", rule.Id, err.Error())
			continue
		}

		for _, event := range activeEvents {
			a.checkAndEscalate(esc, rule, event)
		}
	}
}

// lockEvent 获取事件级互斥锁
func (a *alertEscalationAppImpl) lockEvent(eventId uint64) *sync.Mutex {
	actual, _ := a.eventMu.LoadOrStore(eventId, &sync.Mutex{})
	return actual.(*sync.Mutex)
}

// cleanupEventLock 清理已完成事件的锁，防止内存泄漏
func (a *alertEscalationAppImpl) cleanupEventLock(eventId uint64) {
	a.eventMu.Delete(eventId)
}

// checkAndEscalate 检查并执行升级通知
// 通过 event.EscalationLevel 追踪已升级到的级别，避免重复升级
// 使用事件级锁防止并发修改
func (a *alertEscalationAppImpl) checkAndEscalate(esc *entity.AlertEscalation, rule *entity.AlertRule, event *entity.AlertEvent) {
	mu := a.lockEvent(event.Id)
	mu.Lock()

	// 重新从 DB 读取最新状态，避免使用过期的 EscalationLvl
	freshEvent, err := a.eventApp.GetById(event.Id)
	if err != nil {
		mu.Unlock()
		logx.Errorf("[alert] escalation re-read event error: event[%d] %s", event.Id, err.Error())
		return
	}
	// 仅处理仍然活跃的事件
	if freshEvent.Status != entity.AlertEventStatusFiring && freshEvent.Status != entity.AlertEventStatusAcknowledged {
		// 事件已恢复/关闭，先解锁再清理防止 TOCTOU 竞态
		mu.Unlock()
		a.cleanupEventLock(event.Id)
		return
	}

	elapsed := time.Since(freshEvent.FirstTriggerTime).Minutes()
	needSave := false

	for i, escRule := range esc.Rules {
		if elapsed < float64(escRule.DelayMinutes) {
			continue
		}
		if freshEvent.EscalationLvl > i {
			continue
		}
		if freshEvent.EscalationLvl == i {
			if len(escRule.ChannelIds) > 0 || len(escRule.ReceiverIds) > 0 {
				target := service.NotifyTarget{
					ChannelIds:  escRule.ChannelIds,
					ReceiverIds: escRule.ReceiverIds,
				}

				notifyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				sent, err := a.notifier.NotifyWithTarget(notifyCtx, freshEvent, rule, target)
				cancel()
				if err != nil {
					logx.Errorf("[alert] escalation notify error: esc[%s] rule[%d] event[%d] level[%d] %s",
						esc.Name, rule.Id, freshEvent.Id, i, err.Error())
					// 通知失败时不推进级别，下次扫描重试当前级别
					break
				}
				if sent == 0 {
					logx.Warnf("[alert] escalation[%s] level[%d] has no available channel for event[%d]",
						esc.Name, i, freshEvent.Id)
					// 无可用渠道时不推进级别，下次扫描重试当前级别
					break
				}
			}
			freshEvent.EscalationLvl = i + 1
			needSave = true
		}
	}

	if needSave {
		if err := a.eventApp.UpdateById(context.Background(), freshEvent); err != nil {
			logx.Errorf("[alert] save escalation level error: event[%d] %s", freshEvent.Id, err.Error())
		}
	}
	mu.Unlock()
}

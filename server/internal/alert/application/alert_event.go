package application

import (
	"context"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/repository"
	"mayfly-go/internal/alert/imsg"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"time"
)

// AlertEvent 告警事件应用层接口
type AlertEvent interface {
	base.App[*entity.AlertEvent]

	// GetAlertEventList 分页获取告警事件列表
	GetAlertEventList(condition *entity.AlertEventQuery, orderBy ...string) (*model.PageResult[*entity.AlertEvent], error)

	// FindActive 查找活跃事件
	FindActive(ruleId, resourceId uint64) (*entity.AlertEvent, error)

	// CreateFiringEvent 创建告警中事件（原子操作，防止并发重复创建）
	// 返回 (创建的事件, 是否已存在活跃事件, error)
	CreateFiringEvent(ctx context.Context, rule *entity.AlertRule, resourceId uint64, resourceName, metric string, currentValue float64, threshold string) (*entity.AlertEvent, bool, error)

	// TouchActiveEvent 更新活跃事件（累加触发次数）
	TouchActiveEvent(ctx context.Context, event *entity.AlertEvent, metric string, currentValue float64) error

	// Recover 恢复事件
	Recover(ctx context.Context, event *entity.AlertEvent) error

	// Ack 确认事件（仅允许确认"告警中"的事件）
	Ack(ctx context.Context, eventId uint64, userId int64) error

	// Close 关闭事件（仅允许关闭未终结的事件）
	Close(ctx context.Context, eventId uint64) error

	// DeleteEvent 删除已终结的事件（Recovered/Closed），活跃事件须先关闭
	DeleteEvent(ctx context.Context, eventId uint64) error

	// CloseByRuleId 级联关闭指定规则下所有活跃事件（规则删除/禁用时调用）
	CloseByRuleId(ctx context.Context, ruleId uint64) error

	// CleanupTerminal 清理已终结超过 retainDays 天的事件
	CleanupTerminal(ctx context.Context, retainDays int) (int64, error)

	// CountGroupByStatus 按状态统计数量
	CountGroupByStatus() (map[entity.AlertEventStatus]int64, error)

	// ---- 概览统计方法 ----

	// GetAvgRecoveryTime 计算已恢复事件的平均恢复时间（秒）
	GetAvgRecoveryTime() (int64, error)
	// CountTodayNotifications 统计今日有通知的事件数
	CountTodayNotifications(since time.Time) (int64, error)
	// CountPolicyMatched 统计匹配到通知策略的事件数
	CountPolicyMatched() (int64, error)
	// CountUnmatched 统计活跃但未匹配策略的事件数
	CountUnmatched() (int64, error)
	// CountEscalated 统计已升级的事件数
	CountEscalated() (int64, error)
	// GetMaxEscalationLevel 获取已达到的最高升级级别
	GetMaxEscalationLevel() (int64, error)
	// GetTopByTriggerCount 获取触发次数最多的事件
	GetTopByTriggerCount(limit int) ([]*entity.AlertEvent, error)
	// GetTopByDuration 获取持续时间最长的事件
	GetTopByDuration(limit int) ([]*entity.AlertEvent, error)
	// GetTopByNotifyCount 获取通知次数最多的事件
	GetTopByNotifyCount(limit int) ([]*entity.AlertEvent, error)

	// ListActiveFiring 获取所有告警中的事件（用于抑制检查）
	ListActiveFiring() ([]*entity.AlertEvent, error)
}

type alertEventAppImpl struct {
	base.AppImpl[*entity.AlertEvent, repository.AlertEvent]
}

var _ AlertEvent = (*alertEventAppImpl)(nil)

func (a *alertEventAppImpl) GetAlertEventList(condition *entity.AlertEventQuery, orderBy ...string) (*model.PageResult[*entity.AlertEvent], error) {
	return a.GetRepo().GetAlertEventList(condition, orderBy...)
}

func (a *alertEventAppImpl) FindActive(ruleId, resourceId uint64) (*entity.AlertEvent, error) {
	return a.GetRepo().FindActive(ruleId, resourceId)
}

func (a *alertEventAppImpl) CreateFiringEvent(ctx context.Context, rule *entity.AlertRule, resourceId uint64, resourceName, metric string, currentValue float64, threshold string) (*entity.AlertEvent, bool, error) {
	now := time.Now()
	event := &entity.AlertEvent{
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
		FirstTriggerTime: now,
		LastTriggerTime:  now,
		TriggerCount:     1,
		Labels:           rule.Labels,
	}
	// 设置基础信息（创建时间、创建者等）
	event.FillBaseInfo(model.IdGenTypeNone, nil)
	return a.GetRepo().CreateIfNotActive(event)
}

func (a *alertEventAppImpl) TouchActiveEvent(ctx context.Context, event *entity.AlertEvent, metric string, currentValue float64) error {
	event.LastTriggerTime = time.Now()
	event.TriggerCount++
	event.Metric = metric
	event.CurrentValue = currentValue
	return a.UpdateById(ctx, event)
}

func (a *alertEventAppImpl) Recover(ctx context.Context, event *entity.AlertEvent) error {
	now := time.Now()
	event.Status = entity.AlertEventStatusRecovered
	event.RecoverTime = &now
	return a.UpdateById(ctx, event)
}

// Ack 确认告警事件。
// 仅「告警中」事件可被确认：已恢复/已关闭事件若被置回「已确认」，
// 会重新变成活跃事件并污染概览统计与升级扫描。
func (a *alertEventAppImpl) Ack(ctx context.Context, eventId uint64, userId int64) error {
	event, err := a.GetById(eventId)
	if err != nil {
		return errorx.NewBizI(ctx, imsg.ErrEventNotFound)
	}
	if event.Status != entity.AlertEventStatusFiring {
		return errorx.NewBizI(ctx, imsg.ErrEventAckInvalidStatus)
	}
	now := time.Now()
	event.Status = entity.AlertEventStatusAcknowledged
	event.AckUserId = userId
	event.AckTime = &now
	return a.UpdateById(ctx, event)
}

// Close 关闭告警事件，已关闭的事件幂等拒绝
func (a *alertEventAppImpl) Close(ctx context.Context, eventId uint64) error {
	event, err := a.GetById(eventId)
	if err != nil {
		return errorx.NewBizI(ctx, imsg.ErrEventNotFound)
	}
	if event.Status == entity.AlertEventStatusClosed {
		return errorx.NewBizI(ctx, imsg.ErrEventCloseInvalidStatus)
	}
	event.Status = entity.AlertEventStatusClosed
	return a.UpdateById(ctx, event)
}

// DeleteEvent 删除已终结的事件。
// 活跃事件（Firing/Acknowledged）不允许直接删除，避免丢失正在处理的告警。
func (a *alertEventAppImpl) DeleteEvent(ctx context.Context, eventId uint64) error {
	event, err := a.GetById(eventId)
	if err != nil {
		return errorx.NewBizI(ctx, imsg.ErrEventNotFound)
	}
	if event.Status == entity.AlertEventStatusFiring || event.Status == entity.AlertEventStatusAcknowledged {
		return errorx.NewBizI(ctx, imsg.ErrEventDeleteActive)
	}
	return a.DeleteById(ctx, eventId)
}

// CloseByRuleId 级联关闭指定规则下所有活跃事件。
// 规则删除或禁用时调用，防止活跃事件变成孤儿。
func (a *alertEventAppImpl) CloseByRuleId(ctx context.Context, ruleId uint64) error {
	repo := a.GetRepo()
	events, err := repo.FindActiveByRuleId(ruleId)
	if err != nil {
		return err
	}
	for _, event := range events {
		event.Status = entity.AlertEventStatusClosed
		if err := repo.UpdateById(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// CleanupTerminal 清理已终结（Recovered/Closed）超过 retainDays 天的事件
func (a *alertEventAppImpl) CleanupTerminal(ctx context.Context, retainDays int) (int64, error) {
	return a.GetRepo().CleanupTerminal(retainDays)
}

func (a *alertEventAppImpl) CountGroupByStatus() (map[entity.AlertEventStatus]int64, error) {
	return a.GetRepo().CountGroupByStatus()
}

func (a *alertEventAppImpl) GetAvgRecoveryTime() (int64, error) {
	return a.GetRepo().GetAvgRecoveryTime()
}

func (a *alertEventAppImpl) CountTodayNotifications(since time.Time) (int64, error) {
	return a.GetRepo().CountTodayNotifications(since)
}

func (a *alertEventAppImpl) CountPolicyMatched() (int64, error) {
	return a.GetRepo().CountPolicyMatched()
}

func (a *alertEventAppImpl) CountUnmatched() (int64, error) {
	return a.GetRepo().CountUnmatched()
}

func (a *alertEventAppImpl) CountEscalated() (int64, error) {
	return a.GetRepo().CountEscalated()
}

func (a *alertEventAppImpl) GetMaxEscalationLevel() (int64, error) {
	return a.GetRepo().GetMaxEscalationLevel()
}

func (a *alertEventAppImpl) GetTopByTriggerCount(limit int) ([]*entity.AlertEvent, error) {
	return a.GetRepo().GetTopByTriggerCount(limit)
}

func (a *alertEventAppImpl) GetTopByDuration(limit int) ([]*entity.AlertEvent, error) {
	return a.GetRepo().GetTopByDuration(limit)
}

func (a *alertEventAppImpl) GetTopByNotifyCount(limit int) ([]*entity.AlertEvent, error) {
	return a.GetRepo().GetTopByNotifyCount(limit)
}

// ListActiveFiring 获取所有告警中的事件（用于抑制检查）
func (a *alertEventAppImpl) ListActiveFiring() ([]*entity.AlertEvent, error) {
	return a.GetRepo().SelectByCond(model.NewCond().Eq("status", entity.AlertEventStatusFiring))
}

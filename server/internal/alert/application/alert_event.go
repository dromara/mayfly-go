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
	FindActive(ctx context.Context, ruleId, resourceId uint64) (*entity.AlertEvent, error)

	// CreateFiringEvent 创建告警中事件（原子操作，防止并发重复创建）
	// 返回 (创建的事件, 是否已存在活跃事件, error)
	CreateFiringEvent(ctx context.Context, rule *entity.AlertRule, resourceId uint64, resourceName, metric string, currentValue float64, threshold string) (*entity.AlertEvent, bool, error)

	// TouchActiveEvent 更新活跃事件（累加触发次数）
	TouchActiveEvent(ctx context.Context, event *entity.AlertEvent, metric string, currentValue float64) error

	// UpdateNotifyInfo 仅更新事件的通知计数字段（notify_count/last_notify_time）。
	// 通知链路与评估链路并发操作同一事件：通知侧若全量写回旧快照，
	// 会覆盖评估侧刚写入的 CurrentValue/LastTriggerTime（破坏新鲜度闸门），因此必须限定列
	UpdateNotifyInfo(ctx context.Context, event *entity.AlertEvent) error

	// UpdateEscalationLvl 仅更新事件的升级级别列（escalation_lvl）。
	// 升级链路发送通知期间（最长 30s）评估/通知链路可能已更新同一事件的其他字段，
	// 全量写回旧快照会覆盖它们，因此同样限定列
	UpdateEscalationLvl(ctx context.Context, event *entity.AlertEvent) error

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
	CountGroupByStatus(ctx context.Context) (map[entity.AlertEventStatus]int64, error)

	// ---- 概览统计方法 ----

	// GetAvgRecoveryTime 计算已恢复事件的平均恢复时间（秒）
	GetAvgRecoveryTime(ctx context.Context) (int64, error)
	// CountTodayNotifications 统计今日有通知的事件数
	CountTodayNotifications(ctx context.Context, since time.Time) (int64, error)
	// CountPolicyMatched 统计匹配到通知策略的事件数
	CountPolicyMatched(ctx context.Context) (int64, error)
	// CountUnmatched 统计活跃但未匹配策略的事件数
	CountUnmatched(ctx context.Context) (int64, error)
	// CountEscalated 统计已升级的事件数
	CountEscalated(ctx context.Context) (int64, error)
	// GetMaxEscalationLevel 获取已达到的最高升级级别
	GetMaxEscalationLevel(ctx context.Context) (int64, error)
	// GetTopByTriggerCount 获取触发次数最多的事件
	GetTopByTriggerCount(ctx context.Context, limit int) ([]*entity.AlertEvent, error)
	// GetTopByDuration 获取持续时间最长的事件
	GetTopByDuration(ctx context.Context, limit int) ([]*entity.AlertEvent, error)
	// GetTopByNotifyCount 获取通知次数最多的事件
	GetTopByNotifyCount(ctx context.Context, limit int) ([]*entity.AlertEvent, error)

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

func (a *alertEventAppImpl) FindActive(ctx context.Context, ruleId, resourceId uint64) (*entity.AlertEvent, error) {
	return a.GetRepo().FindActive(ctx, ruleId, resourceId)
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
	return a.GetRepo().CreateIfNotActive(ctx, event)
}

func (a *alertEventAppImpl) TouchActiveEvent(ctx context.Context, event *entity.AlertEvent, metric string, currentValue float64) error {
	event.LastTriggerTime = time.Now()
	event.TriggerCount++
	event.Metric = metric
	event.CurrentValue = currentValue
	// 只写本次触摸真正改动的列（与 UpdateNotifyInfo/UpdateEscalationLvl 同一口径）。
	// 整行写回会用本轮开头的旧快照覆盖并发写入的 notify_count/last_notify_time
	//（通知刷新循环与评估循环是两个独立 cron），表现为重复通知节奏偏移、已通知次数倒退；
	// resource_name 必须带上：评估链路在「历史活跃事件资源名为空」时先回填再触摸，靠的就是这一写
	return a.GetRepo().UpdateById(ctx, event, "last_trigger_time", "trigger_count", "metric", "current_value", "resource_name")
}

// UpdateNotifyInfo 仅更新通知计数字段，避免全量写回旧快照覆盖评估链路并发写入的指标值
func (a *alertEventAppImpl) UpdateNotifyInfo(ctx context.Context, event *entity.AlertEvent) error {
	return a.GetRepo().UpdateById(ctx, event, "notify_count", "last_notify_time")
}

// UpdateEscalationLvl 仅更新升级级别列，避免升级链路全量写回旧快照覆盖并发写入
func (a *alertEventAppImpl) UpdateEscalationLvl(ctx context.Context, event *entity.AlertEvent) error {
	return a.GetRepo().UpdateById(ctx, event, "escalation_lvl")
}

func (a *alertEventAppImpl) Recover(ctx context.Context, event *entity.AlertEvent) error {
	now := time.Now()
	event.Status = entity.AlertEventStatusRecovered
	event.RecoverTime = &now
	// 同样只写状态列：整行写回会把恢复前旧快照的 notify_count/metric 等写回去，
	// 而恢复通知与升级扫描可能刚更新过它们
	return a.GetRepo().UpdateById(ctx, event, "status", "recover_time")
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
	// 按列更新：与上面几个写入口同一口径，不让用户侧动作把评估/通知链路刚写的列带回去
	return a.GetRepo().UpdateById(ctx, event, "status", "ack_user_id", "ack_time")
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
	return a.GetRepo().UpdateById(ctx, event, "status")
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
	events, err := repo.FindActiveByRuleId(ctx, ruleId)
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
	return a.GetRepo().CleanupTerminal(ctx, retainDays)
}

func (a *alertEventAppImpl) CountGroupByStatus(ctx context.Context) (map[entity.AlertEventStatus]int64, error) {
	return a.GetRepo().CountGroupByStatus(ctx)
}

func (a *alertEventAppImpl) GetAvgRecoveryTime(ctx context.Context) (int64, error) {
	return a.GetRepo().GetAvgRecoveryTime(ctx)
}

func (a *alertEventAppImpl) CountTodayNotifications(ctx context.Context, since time.Time) (int64, error) {
	return a.GetRepo().CountTodayNotifications(ctx, since)
}

func (a *alertEventAppImpl) CountPolicyMatched(ctx context.Context) (int64, error) {
	return a.GetRepo().CountPolicyMatched(ctx)
}

func (a *alertEventAppImpl) CountUnmatched(ctx context.Context) (int64, error) {
	return a.GetRepo().CountUnmatched(ctx)
}

func (a *alertEventAppImpl) CountEscalated(ctx context.Context) (int64, error) {
	return a.GetRepo().CountEscalated(ctx)
}

func (a *alertEventAppImpl) GetMaxEscalationLevel(ctx context.Context) (int64, error) {
	return a.GetRepo().GetMaxEscalationLevel(ctx)
}

func (a *alertEventAppImpl) GetTopByTriggerCount(ctx context.Context, limit int) ([]*entity.AlertEvent, error) {
	return a.GetRepo().GetTopByTriggerCount(ctx, limit)
}

func (a *alertEventAppImpl) GetTopByDuration(ctx context.Context, limit int) ([]*entity.AlertEvent, error) {
	return a.GetRepo().GetTopByDuration(ctx, limit)
}

func (a *alertEventAppImpl) GetTopByNotifyCount(ctx context.Context, limit int) ([]*entity.AlertEvent, error) {
	return a.GetRepo().GetTopByNotifyCount(ctx, limit)
}

// ListActiveFiring 获取所有告警中的事件（用于抑制检查）
func (a *alertEventAppImpl) ListActiveFiring() ([]*entity.AlertEvent, error) {
	return a.GetRepo().SelectByCond(model.NewCond().Eq("status", entity.AlertEventStatusFiring))
}

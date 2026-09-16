package application

import (
	"context"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"time"
)

// 概览 Top N 常量（避免魔法数字）
const (
	overviewTopN           = 5
	overviewRecentPageSize = 20
	overviewTrendHours     = 23
)

// AlertOverview 告警概览应用层接口
type AlertOverview interface {
	// GetOverview 获取告警概览数据
	GetOverview(ctx context.Context) (*entity.AlertOverviewData, error)
}

type alertOverviewAppImpl struct {
	base.AppImpl[*entity.AlertEvent, repository.AlertEvent]
	eventApp        AlertEvent                `inject:"T"`
	ruleApp         AlertRule                 `inject:"T"`
	silenceApp      AlertSilence              `inject:"T"`
	inhibitionApp   AlertInhibition           `inject:"T"`
	escalationApp   AlertEscalation           `inject:"T"`
	notifyPolicyApp AlertNotifyPolicy         `inject:"T"`
	notifyLogRepo   repository.AlertNotifyLog `inject:"T"`
}

var _ AlertOverview = (*alertOverviewAppImpl)(nil)

func (a *alertOverviewAppImpl) GetOverview(ctx context.Context) (*entity.AlertOverviewData, error) {
	data := &entity.AlertOverviewData{}

	// 1. 事件状态统计
	statusCounts, err := a.eventApp.CountGroupByStatus()
	if err != nil {
		return nil, err
	}
	data.FiringCount = statusCounts[entity.AlertEventStatusFiring]
	data.AcknowledgedCount = statusCounts[entity.AlertEventStatusAcknowledged]
	data.RecoveredCount = statusCounts[entity.AlertEventStatusRecovered]
	data.ClosedCount = statusCounts[entity.AlertEventStatusClosed]

	// 2. MTTR 计算
	data.AvgRecoveryTime, _ = a.eventApp.GetAvgRecoveryTime()

	// 3. 今日通知统计
	today := time.Now().Truncate(24 * time.Hour)
	data.TodayNotifyCount, _ = a.eventApp.CountTodayNotifications(today)

	// 4. 规则统计
	data.RuleStats = a.calculateRuleStats()

	// 5. 通知统计（阶段一 + 阶段二）
	data.NotifyStats = a.calculateNotifyStats(statusCounts, today)

	// 6. 静默统计
	data.SilenceStats = a.calculateSilenceStats()

	// 7. 抑制统计
	data.InhibitionStats = a.calculateInhibitionStats()

	// 8. 升级统计
	data.EscalationStats = a.calculateEscalationStats()

	// 9. Top 告警
	data.TopFrequent, _ = a.eventApp.GetTopByTriggerCount(overviewTopN)
	data.TopLongest, _ = a.eventApp.GetTopByDuration(overviewTopN)
	data.TopNotified, _ = a.eventApp.GetTopByNotifyCount(overviewTopN)

	// 10. 最近事件
	recentEvents, err := a.eventApp.GetAlertEventList(&entity.AlertEventQuery{
		PageParam: model.PageParam{PageNum: 1, PageSize: overviewRecentPageSize},
	}, "last_trigger_time desc")
	if err != nil {
		recentEvents = &model.PageResult[*entity.AlertEvent]{Total: 0, List: []*entity.AlertEvent{}}
	}
	data.RecentEvents = &model.PageResult[*entity.AlertEvent]{
		Total: recentEvents.Total,
		List:  recentEvents.List,
	}

	// 11. 阶段三：通知趋势（最近 24 小时，通过 repository 一条 SQL 完成）
	since := time.Now().Add(-overviewTrendHours * time.Hour).Truncate(time.Hour)
	trend, trendErr := a.notifyLogRepo.GetHourlyTrend(since)
	if trendErr != nil {
		logx.Warnf("[alert] get notify trend error: %v", trendErr)
		data.NotifyTrend = []entity.NotifyTrendPoint{}
	} else if trend != nil {
		data.NotifyTrend = make([]entity.NotifyTrendPoint, 0, len(trend))
		for _, p := range trend {
			data.NotifyTrend = append(data.NotifyTrend, *p)
		}
	} else {
		data.NotifyTrend = []entity.NotifyTrendPoint{}
	}

	// 12. 阶段三：通知质量报告
	data.QualityReport = a.calculateQualityReport()

	return data, nil
}

// calculateRuleStats 计算规则统计（通过 ruleApp repository）
func (a *alertOverviewAppImpl) calculateRuleStats() entity.RuleStats {
	stats := entity.RuleStats{
		ByPriority:     make(map[string]int64),
		ByResourceType: make(map[int8]int64),
	}

	stats.Total, stats.Enabled, stats.Disabled, _ = a.ruleApp.GetRuleStats()

	byPriority, err := a.ruleApp.CountGroupByPriority()
	if err == nil && byPriority != nil {
		stats.ByPriority = byPriority
	}

	byResourceType, err := a.ruleApp.CountGroupByResourceType()
	if err == nil && byResourceType != nil {
		stats.ByResourceType = byResourceType
	}

	return stats
}

// calculateNotifyStats 计算通知统计（阶段一 + 阶段二）
func (a *alertOverviewAppImpl) calculateNotifyStats(statusCounts map[entity.AlertEventStatus]int64, today time.Time) entity.NotifyStats {
	stats := entity.NotifyStats{
		ChannelDistribution: make(map[string]int64),
		ByChannel:           make(map[string]entity.ChannelStat),
	}

	// ===== 阶段一：基础统计（通过 eventApp repository） =====

	stats.TodaySent, _ = a.eventApp.CountTodayNotifications(today)

	// 总事件数
	stats.TotalEvents = statusCounts[entity.AlertEventStatusFiring] +
		statusCounts[entity.AlertEventStatusAcknowledged] +
		statusCounts[entity.AlertEventStatusRecovered] +
		statusCounts[entity.AlertEventStatusClosed]

	stats.PolicyMatched, _ = a.eventApp.CountPolicyMatched()
	stats.UnmatchedCount, _ = a.eventApp.CountUnmatched()

	// 策略匹配率
	effectiveEvents := stats.TotalEvents - stats.UnmatchedCount
	if effectiveEvents > 0 {
		stats.SuccessRate = float64(stats.PolicyMatched) / float64(effectiveEvents) * 100
	}

	// 渠道分布（通过 notifyPolicyApp repository）
	dist, err := a.notifyPolicyApp.GetChannelDistribution()
	if err == nil && dist != nil {
		stats.ChannelDistribution = dist
	}

	// ===== 阶段二：精确统计（通过 notifyLogRepo） =====
	todaySent, todaySuccess, todayFailed, err := a.notifyLogRepo.GetTodayStats()
	if err != nil {
		logx.Warnf("[alert] get today notify stats error: %v", err)
	} else {
		stats.TodaySent = todaySent
		stats.TodaySuccess = todaySuccess
		stats.TodayFailed = todayFailed
	}

	// 使用日志表的真实发送成功率覆盖 successRate
	if todaySent > 0 {
		stats.SuccessRate = float64(todaySuccess) / float64(todaySent) * 100
	}

	// 按渠道统计
	byChannel, err := a.notifyLogRepo.GetByChannelStats(today)
	if err != nil {
		logx.Warnf("[alert] get by-channel stats error: %v", err)
	} else if byChannel != nil {
		for name, stat := range byChannel {
			stats.ByChannel[name] = *stat
		}
	}

	return stats
}

// calculateSilenceStats 计算静默统计（通过 silenceApp CountByCond）
func (a *alertOverviewAppImpl) calculateSilenceStats() entity.SilenceStats {
	stats := entity.SilenceStats{}
	stats.Total = a.silenceApp.CountByCond("is_deleted = 0")
	now := time.Now()
	stats.Active = a.silenceApp.CountByCond(
		model.NewCond().Eq("status", entity.AlertSilenceStatusEnable).
			And("start_time <= ?", now).
			And("end_time >= ?", now),
	)
	return stats
}

// calculateInhibitionStats 计算抑制统计（通过 inhibitionApp CountByCond）
func (a *alertOverviewAppImpl) calculateInhibitionStats() entity.InhibitionStats {
	stats := entity.InhibitionStats{}
	stats.Total = a.inhibitionApp.CountByCond("is_deleted = 0")
	stats.Active = a.inhibitionApp.CountByCond(
		model.NewCond().Eq("status", entity.AlertInhibitionStatusEnable),
	)
	return stats
}

// calculateEscalationStats 计算升级统计（通过 escalationApp + eventApp repository）
func (a *alertOverviewAppImpl) calculateEscalationStats() entity.EscalationStats {
	stats := entity.EscalationStats{}
	stats.Total = a.escalationApp.CountByCond("is_deleted = 0")
	stats.Active = a.escalationApp.CountByCond(
		model.NewCond().Eq("status", entity.AlertEscalationStatusEnable),
	)
	stats.EscalatedCount, _ = a.eventApp.CountEscalated()
	stats.MaxLevelReached, _ = a.eventApp.GetMaxEscalationLevel()
	return stats
}

// calculateQualityReport 计算通知质量报告（阶段三，全部通过 repository）
func (a *alertOverviewAppImpl) calculateQualityReport() entity.NotifyQualityReport {
	report := entity.NotifyQualityReport{
		TopFailReasons: []entity.FailReason{},
	}

	// 发送延迟
	avgDelay, maxDelay, err := a.notifyLogRepo.GetSendDelay()
	if err != nil {
		logx.Warnf("[alert] get send delay error: %v", err)
	}
	report.AvgSendDelay = avgDelay
	report.MaxSendDelay = maxDelay

	// 本周成功率
	weekAgo := time.Now().AddDate(0, 0, -7)
	weeklyRate, err := a.notifyLogRepo.GetSuccessRateSince(weekAgo)
	if err != nil {
		logx.Warnf("[alert] get weekly success rate error: %v", err)
	}
	report.WeeklySuccess = weeklyRate

	// 本月成功率
	monthAgo := time.Now().AddDate(0, -1, 0)
	monthlyRate, err := a.notifyLogRepo.GetSuccessRateSince(monthAgo)
	if err != nil {
		logx.Warnf("[alert] get monthly success rate error: %v", err)
	}
	report.MonthlySuccess = monthlyRate

	// 主要失败原因
	failReasons, err := a.notifyLogRepo.GetFailReasons(overviewTopN)
	if err != nil {
		logx.Warnf("[alert] get fail reasons error: %v", err)
	} else if failReasons != nil {
		report.TopFailReasons = make([]entity.FailReason, 0, len(failReasons))
		for _, fr := range failReasons {
			report.TopFailReasons = append(report.TopFailReasons, *fr)
		}
	}

	return report
}

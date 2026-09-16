package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

// AlertOverviewData 告警概览数据
type AlertOverviewData struct {
	// 事件状态统计
	FiringCount       int64 `json:"firingCount"`
	AcknowledgedCount int64 `json:"acknowledgedCount"`
	RecoveredCount    int64 `json:"recoveredCount"`
	ClosedCount       int64 `json:"closedCount"`

	// MTTR (平均恢复时间，单位秒)
	AvgRecoveryTime int64 `json:"avgRecoveryTime"`

	// 今日统计
	TodayNotifyCount int64 `json:"todayNotifyCount"`

	// 规则统计
	RuleStats RuleStats `json:"ruleStats"`

	// 通知统计（阶段一 + 阶段二 + 阶段三）
	NotifyStats NotifyStats `json:"notifyStats"`

	// 静默/抑制/升级统计
	SilenceStats    SilenceStats    `json:"silenceStats"`
	InhibitionStats InhibitionStats `json:"inhibitionStats"`
	EscalationStats EscalationStats `json:"escalationStats"`

	// Top 告警
	TopFrequent []*AlertEvent `json:"topFrequent"` // 最频繁触发
	TopLongest  []*AlertEvent `json:"topLongest"`  // 持续时间最长
	TopNotified []*AlertEvent `json:"topNotified"` // 通知次数最多

	// 最近事件
	RecentEvents *model.PageResult[*AlertEvent] `json:"recentEvents"`

	// 阶段三：历史趋势（最近 24 小时）
	NotifyTrend []NotifyTrendPoint `json:"notifyTrend"`

	// 阶段三：通知质量报告
	QualityReport NotifyQualityReport `json:"qualityReport"`
}

// RuleStats 规则统计
type RuleStats struct {
	Total          int64            `json:"total"`
	Enabled        int64            `json:"enabled"`
	Disabled       int64            `json:"disabled"`
	ByPriority     map[string]int64 `json:"byPriority"`     // P0/P1/P2/P3 分布
	ByResourceType map[int8]int64   `json:"byResourceType"` // 按资源类型分布
}

// NotifyStats 通知统计（阶段一 + 阶段二）
type NotifyStats struct {
	// 阶段一：基础统计
	TodaySent      int64   `json:"todaySent"`      // 今日发送通知数
	SuccessRate    float64 `json:"successRate"`    // 成功率 (0-100)
	PolicyMatched  int64   `json:"policyMatched"`  // 匹配到策略的事件数
	TotalEvents    int64   `json:"totalEvents"`    // 总事件数
	UnmatchedCount int64   `json:"unmatchedCount"` // 未匹配策略的事件数

	// 阶段一：渠道分布（基于策略配置）
	ChannelDistribution map[string]int64 `json:"channelDistribution"` // 渠道名 -> 覆盖事件数

	// 阶段二：精确统计（基于日志表）
	TodaySuccess int64                  `json:"todaySuccess"` // 今日成功数
	TodayFailed  int64                  `json:"todayFailed"`  // 今日失败数
	ByChannel    map[string]ChannelStat `json:"byChannel"`    // 渠道维度统计
}

// ChannelStat 渠道统计（阶段二）
type ChannelStat struct {
	Sent    int64   `json:"sent"`    // 发送总数
	Success int64   `json:"success"` // 成功数
	Failed  int64   `json:"failed"`  // 失败数
	Rate    float64 `json:"rate"`    // 成功率
}

// NotifyTrendPoint 通知趋势点（阶段三）
type NotifyTrendPoint struct {
	Hour    string `json:"hour"`    // 小时 (00:00, 01:00, ...)
	Sent    int64  `json:"sent"`    // 发送数
	Success int64  `json:"success"` // 成功数
	Failed  int64  `json:"failed"`  // 失败数
}

// NotifyQualityReport 通知质量报告（阶段三）
type NotifyQualityReport struct {
	AvgSendDelay   int64        `json:"avgSendDelay"`   // 平均发送延迟（秒）
	MaxSendDelay   int64        `json:"maxSendDelay"`   // 最大发送延迟（秒）
	QueueLength    int64        `json:"queueLength"`    // 当前待发送队列长度
	WeeklySuccess  float64      `json:"weeklySuccess"`  // 本周成功率
	MonthlySuccess float64      `json:"monthlySuccess"` // 本月成功率
	TopFailReasons []FailReason `json:"topFailReasons"` // 主要失败原因
}

// FailReason 失败原因统计（阶段三）
type FailReason struct {
	Reason string `json:"reason"` // 失败原因
	Count  int64  `json:"count"`  // 出现次数
}

// SilenceStats 静默统计
type SilenceStats struct {
	Total         int64 `json:"total"`         // 静默规则总数
	Active        int64 `json:"active"`        // 当前活跃的静默规则数
	SilencedCount int64 `json:"silencedCount"` // 被静默的事件数（估算）
}

// InhibitionStats 抑制统计
type InhibitionStats struct {
	Total          int64 `json:"total"`          // 抑制规则总数
	Active         int64 `json:"active"`         // 当前活跃的抑制规则数
	InhibitedCount int64 `json:"inhibitedCount"` // 被抑制的事件数（估算）
}

// EscalationStats 升级统计
type EscalationStats struct {
	Total           int64 `json:"total"`           // 升级策略总数
	Active          int64 `json:"active"`          // 当前活跃的升级策略数
	EscalatedCount  int64 `json:"escalatedCount"`  // 已升级的事件数
	MaxLevelReached int64 `json:"maxLevelReached"` // 达到的最高升级级别
}

// AlertNotifyLog 通知发送日志（阶段二新增）
type AlertNotifyLog struct {
	Id          uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	EventId     uint64    `json:"eventId" gorm:"not null;index:idx_notify_log_event;comment:事件 ID"`
	PolicyId    uint64    `json:"policyId" gorm:"comment:匹配的策略 ID"`
	ChannelId   uint64    `json:"channelId" gorm:"not null;index:idx_notify_log_channel;comment:渠道 ID"`
	ChannelName string    `json:"channelName" gorm:"size:100;comment:渠道名称"`
	Status      int8      `json:"status" gorm:"not null;comment:1:成功 2:失败"`
	ErrorMsg    string    `json:"errorMsg" gorm:"size:500;comment:失败原因"`
	SendTime    time.Time `json:"sendTime" gorm:"not null;index:idx_notify_log_send_time;comment:发送时间"`
	CreatedAt   time.Time `json:"createdAt" gorm:"autoCreateTime"`
}

func (a *AlertNotifyLog) TableName() string {
	return "t_alert_notify_log"
}

// FillBaseInfo 实现 model.ModelI 接口（通知日志不需要填充基础信息）
func (a *AlertNotifyLog) FillBaseInfo(idGenType model.IdGenType, account *model.LoginAccount) {}

// SetId 实现 model.ModelI 接口
func (a *AlertNotifyLog) SetId(id uint64) {
	a.Id = id
}

// IsCreate 实现 model.ModelI 接口
func (a *AlertNotifyLog) IsCreate() bool {
	return a.Id == 0
}

// LogicDelete 实现 model.ModelI 接口（通知日志不使用逻辑删除）
func (a *AlertNotifyLog) LogicDelete() bool {
	return false
}

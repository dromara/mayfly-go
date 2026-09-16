package repository

import (
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
	"time"
)

// AlertRule 告警规则仓库接口
type AlertRule interface {
	base.Repo[*entity.AlertRule]

	GetAlertRuleList(condition *entity.AlertRuleQuery, orderBy ...string) (*model.PageResult[*entity.AlertRule], error)
	ListEnabled() ([]*entity.AlertRule, error)

	// GetRuleStats 返回 (总数, 启用数, 禁用数, error)
	GetRuleStats() (total, enabled, disabled int64, err error)
	// CountGroupByPriority 按优先级分组统计，返回 map["P0"]count
	CountGroupByPriority() (map[string]int64, error)
	// CountGroupByResourceType 按资源类型分组统计
	CountGroupByResourceType() (map[int8]int64, error)
}

// AlertEvent 告警事件仓库接口
type AlertEvent interface {
	base.Repo[*entity.AlertEvent]

	GetAlertEventList(condition *entity.AlertEventQuery, orderBy ...string) (*model.PageResult[*entity.AlertEvent], error)
	FindActive(ruleId, resourceId uint64) (*entity.AlertEvent, error)
	// CreateIfNotActive 原子地检查并创建活跃事件（防止 TOCTOU 竞态）
	// 返回 (创建的事件, 是否已存在活跃事件, error)
	CreateIfNotActive(event *entity.AlertEvent) (*entity.AlertEvent, bool, error)
	// FindActiveByRuleId 查找指定规则下所有活跃事件（规则删除/禁用时级联关闭用）
	FindActiveByRuleId(ruleId uint64) ([]*entity.AlertEvent, error)
	// CleanupTerminal 删除已终结超过 retainDays 天的事件，返回删除行数
	CleanupTerminal(retainDays int) (int64, error)
	// CountGroupByStatus 按状态统计数量
	CountGroupByStatus() (map[entity.AlertEventStatus]int64, error)

	// ---- 概览统计方法 ----

	// GetAvgRecoveryTime 计算已恢复事件的平均恢复时间（秒）
	GetAvgRecoveryTime() (int64, error)
	// CountTodayNotifications 统计今日有通知的事件数
	CountTodayNotifications(since time.Time) (int64, error)
	// CountPolicyMatched 统计匹配到通知策略的事件数（notify_count > 0）
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
}

// AlertSilence 静默规则仓库接口
type AlertSilence interface {
	base.Repo[*entity.AlertSilence]

	GetAlertSilenceList(condition *entity.AlertSilenceQuery, orderBy ...string) (*model.PageResult[*entity.AlertSilence], error)
	// ListActive 获取当前生效的静默规则
	ListActive() ([]*entity.AlertSilence, error)
}

// AlertEscalation 升级策略仓库接口
type AlertEscalation interface {
	base.Repo[*entity.AlertEscalation]

	GetAlertEscalationList(condition *entity.AlertEscalationQuery, orderBy ...string) (*model.PageResult[*entity.AlertEscalation], error)
	ListEnabled() ([]*entity.AlertEscalation, error)
}

// AlertInhibition 抑制规则仓库接口
type AlertInhibition interface {
	base.Repo[*entity.AlertInhibition]

	GetAlertInhibitionList(condition *entity.AlertInhibitionQuery, orderBy ...string) (*model.PageResult[*entity.AlertInhibition], error)
	ListEnabled() ([]*entity.AlertInhibition, error)
}

// AlertNotifyPolicy 通知策略仓库接口
type AlertNotifyPolicy interface {
	base.Repo[*entity.AlertNotifyPolicy]

	GetAlertNotifyPolicyList(condition *entity.AlertNotifyPolicyQuery, orderBy ...string) (*model.PageResult[*entity.AlertNotifyPolicy], error)
	ListEnabled() ([]*entity.AlertNotifyPolicy, error)

	// GetChannelDistribution 按渠道统计关联的启用策略数
	GetChannelDistribution() (map[string]int64, error)
}

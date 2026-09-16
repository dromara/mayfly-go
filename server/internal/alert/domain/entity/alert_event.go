package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

// AlertEvent 告警事件
type AlertEvent struct {
	model.Model
	model.ExtraData // 扩展字段：告警详情快照、上下文信息等

	RuleId       uint64        `json:"ruleId" gorm:"not null;index:idx_alert_event_rule;comment:规则ID"`
	RuleName     string        `json:"ruleName" gorm:"size:100;comment:规则名称"`
	Priority     AlertPriority `json:"priority" gorm:"not null;comment:优先级"`
	ResourceType int8          `json:"resourceType" gorm:"not null;comment:资源类型"`
	ResourceId   uint64        `json:"resourceId" gorm:"not null;index:idx_alert_event_resource;comment:资源ID"`
	ResourceName string        `json:"resourceName" gorm:"size:100;comment:资源名称"`

	Status       AlertEventStatus `json:"status" gorm:"not null;default:1;index:idx_alert_event_status;comment:告警状态"`
	Metric       string           `json:"metric" gorm:"size:50;comment:触发指标"`
	CurrentValue float64          `json:"currentValue" gorm:"comment:当前值"`
	Threshold    string           `json:"threshold" gorm:"size:200;comment:阈值描述"`

	FirstTriggerTime time.Time  `json:"firstTriggerTime" gorm:"not null;comment:首次触发时间"`
	LastTriggerTime  time.Time  `json:"lastTriggerTime" gorm:"not null;index:idx_alert_event_last;comment:最近触发时间"`
	RecoverTime      *time.Time `json:"recoverTime" gorm:"comment:恢复时间"`
	AckUserId        int64      `json:"ackUserId" gorm:"comment:确认人"`
	AckTime          *time.Time `json:"ackTime" gorm:"comment:确认时间"`

	TriggerCount   int        `json:"triggerCount" gorm:"default:0;comment:累计触发次数"`
	NotifyCount    int        `json:"notifyCount" gorm:"default:0;comment:已通知次数"`
	LastNotifyTime *time.Time `json:"lastNotifyTime" gorm:"comment:最近通知时间"`
	EscalationLvl  int        `json:"escalationLevel" gorm:"default:0;comment:已升级到的级别索引"`
	Labels         string     `json:"labels" gorm:"type:json;comment:标签(用于分组/路由)"`
}

func (a *AlertEvent) TableName() string {
	return "t_alert_event"
}

// AlertEventStatus 告警事件状态
type AlertEventStatus int8

const (
	AlertEventStatusFiring       AlertEventStatus = 1 // 告警中
	AlertEventStatusAcknowledged AlertEventStatus = 2 // 已确认
	AlertEventStatusRecovered    AlertEventStatus = 3 // 已恢复
	AlertEventStatusClosed       AlertEventStatus = 4 // 已关闭
)

// AlertEventQuery 告警事件查询条件
//
// Priority 使用指针：P0 的枚举值为 0，若用值类型则"未筛选"与"筛选 P0"无法区分，
// 且 model.QueryCond.Eq 会忽略零值导致按 P0 筛选静默失效。
type AlertEventQuery struct {
	model.PageParam

	RuleId       uint64           `json:"ruleId" form:"ruleId"`
	ResourceType int8             `json:"resourceType" form:"resourceType"`
	ResourceId   uint64           `json:"resourceId" form:"resourceId"`
	Status       AlertEventStatus `json:"status" form:"status"`
	Priority     *AlertPriority   `json:"priority" form:"priority"`
	Keyword      string           `json:"keyword" form:"keyword"`
}

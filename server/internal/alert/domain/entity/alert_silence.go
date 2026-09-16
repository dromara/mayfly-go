package entity

import (
	"mayfly-go/pkg/model"
)

// AlertSilence 静默规则
// 通过 MatchLabels 匹配告警规则的标签，在指定时间范围内静默匹配的告警通知。
// 与 AlertRule 的 Labels 字段做子集匹配：静默的 matchLabels 为空则匹配所有告警。
type AlertSilence struct {
	model.Model

	Name        string `json:"name" gorm:"size:100;not null;comment:静默名称"`
	Status      int8   `json:"status" gorm:"not null;default:1;index:idx_alert_silence_status;comment:状态 1:启用 -1:禁用"`
	MatchLabels string `json:"matchLabels" gorm:"-"` // 不存 DB，由 t_label_binding 管理

	// 使用 model.JsonTime 而非裸 time.Time：前后端约定的时间格式为 "2006-01-02 15:04:05"（本地时区），
	// 裸 time.Time 只接受 RFC3339，前端日期选择器提交的格式会导致 JSON 解析失败并返回 500
	StartTime model.JsonTime `json:"startTime" gorm:"not null;index:idx_alert_silence_time;comment:生效开始时间"`
	EndTime   model.JsonTime `json:"endTime" gorm:"not null;comment:生效结束时间"`
	Remark    string         `json:"remark" gorm:"size:500;comment:备注"`
}

func (a *AlertSilence) TableName() string {
	return "t_alert_silence"
}

const (
	AlertSilenceStatusEnable  int8 = 1
	AlertSilenceStatusDisable int8 = -1
)

type AlertSilenceQuery struct {
	model.PageParam

	Name    string `json:"name" form:"name"`
	Status  int8   `json:"status" form:"status"`
	Keyword string `json:"keyword" form:"keyword"`
}

package entity

import (
	"mayfly-go/pkg/model"
)

// LabelBinding 标签绑定关系表
// 通过 label_id 关联 t_label，绑定到各类目标实体（资源、告警规则等）。
type LabelBinding struct {
	model.Model

	LabelId    uint64 `json:"labelId" gorm:"not null;index:idx_lb_label;uniqueIndex:idx_lb_unique;comment:标签 ID"`
	TargetType string `json:"targetType" gorm:"size:50;not null;uniqueIndex:idx_lb_unique;comment:目标类型"`
	TargetId   uint64 `json:"targetId" gorm:"not null;index:idx_lb_target;uniqueIndex:idx_lb_unique;comment:目标 ID"`
}

func (l *LabelBinding) TableName() string {
	return "t_label_binding"
}

// 目标类型常量
const (
	LabelTargetMachine               = "machine"
	LabelTargetDbInstance            = "db_instance"
	LabelTargetRedis                 = "redis"
	LabelTargetMongo                 = "mongo"
	LabelTargetAlertRule             = "alert_rule"
	LabelTargetAlertSilence          = "alert_silence"
	LabelTargetAlertInhibition       = "alert_inhibition"
	LabelTargetAlertInhibitionSource = "alert_inhibition_source"
	LabelTargetAlertInhibitionTarget = "alert_inhibition_target"
	LabelTargetAlertEscalation       = "alert_escalation"
	LabelTargetAlertNotifyPolicy     = "alert_notify_policy"
)

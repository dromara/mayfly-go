package entity

import (
	"mayfly-go/pkg/model"
)

// AlertInhibition 告警抑制规则
// 当源告警（SourceMatch）触发时，抑制目标告警（TargetMatch）的通知。
// Equal 标签用于限定抑制范围：只有源和目标在这些标签上值相同时才抑制。
//
// 示例：集群不可达时抑制该集群下所有实例的告警
//
//	SourceMatch: {"alertname": "ClusterDown"}
//	TargetMatch: {"alertname": "InstanceDown"}
//	Equal:       ["cluster"]
type AlertInhibition struct {
	model.Model

	Name   string `json:"name" gorm:"size:100;not null;comment:抑制规则名称"`
	Status int8   `json:"status" gorm:"not null;default:1;comment:状态 1:启用 -1:禁用"`
	Remark string `json:"remark" gorm:"size:500;comment:备注"`

	// 源告警标签匹配器：当源告警匹配这些标签时触发抑制
	SourceMatch string `json:"sourceMatch" gorm:"type:json;comment:源告警标签匹配(JSON map)"`

	// 目标告警标签匹配器：被抑制的告警必须匹配这些标签
	TargetMatch string `json:"targetMatch" gorm:"type:json;comment:目标告警标签匹配(JSON map)"`

	// 相等标签：源和目标在这些标签上值必须相等才抑制
	// 为空表示只要源匹配就抑制所有目标（全局抑制）
	Equal string `json:"equal" gorm:"type:json;comment:相等标签(JSON array)"`
}

func (a *AlertInhibition) TableName() string {
	return "t_alert_inhibition"
}

const (
	AlertInhibitionStatusEnable  int8 = 1  // 启用
	AlertInhibitionStatusDisable int8 = -1 // 禁用
)

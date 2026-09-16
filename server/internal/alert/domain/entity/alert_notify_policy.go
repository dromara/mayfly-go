package entity

import "mayfly-go/pkg/model"

// AlertNotifyPolicy 通知策略
// 独立于告警规则的通知路由实体，通过标签匹配将告警事件路由到对应的通知渠道和接收人。
// 多条策略可叠加：一个事件匹配的所有策略的渠道/接收人合并发送。
type AlertNotifyPolicy struct {
	model.Model

	Name           string   `json:"name" gorm:"size:100;not null;comment:策略名称"`
	Status         int8     `json:"status" gorm:"not null;default:1;index:idx_alert_np_status;comment:状态 1:启用 -1:禁用"`
	MatchLabels    string   `json:"matchLabels" gorm:"-"` // 不存 DB，由 t_label_binding 管理
	ChannelIds     []uint64 `json:"channelIds" gorm:"type:json;serializer:json;comment:通知渠道ID列表"`
	ReceiverIds    []int64  `json:"receiverIds" gorm:"type:json;serializer:json;comment:通知接收人ID列表"`
	RepeatInterval int      `json:"repeatInterval" gorm:"default:300;comment:重复通知间隔(秒)"`
	Remark         string   `json:"remark" gorm:"size:500;comment:备注"`
}

func (a *AlertNotifyPolicy) TableName() string {
	return "t_alert_notify_policy"
}

const (
	AlertNotifyPolicyStatusEnable  int8 = 1
	AlertNotifyPolicyStatusDisable int8 = -1
)

type AlertNotifyPolicyQuery struct {
	model.PageParam

	Name    string `json:"name" form:"name"`
	Status  int8   `json:"status" form:"status"`
	Keyword string `json:"keyword" form:"keyword"`
}

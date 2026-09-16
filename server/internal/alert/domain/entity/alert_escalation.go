package entity

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"mayfly-go/pkg/model"
)

// AlertEscalation 升级策略
// 通过 MatchLabels 匹配告警规则的标签，对匹配的告警事件按时间逐级升级通知。
// MatchLabels 为空则匹配所有告警规则。
type AlertEscalation struct {
	model.Model

	Name        string             `json:"name" gorm:"size:100;not null;comment:策略名称"`
	Status      int8               `json:"status" gorm:"not null;default:1;index:idx_alert_esc_status;comment:状态 1:启用 -1:禁用"`
	MatchLabels string             `json:"matchLabels" gorm:"-"` // 不存 DB，由 t_label_binding 管理
	Rules       EscalationRuleList `json:"rules" gorm:"type:json;serializer:json;comment:升级规则链(JSON)"`
	Remark      string             `json:"remark" gorm:"size:500;comment:备注"`
}

func (a *AlertEscalation) TableName() string {
	return "t_alert_escalation"
}

// EscalationRuleList 升级规则链（按延迟时间递增排序）
type EscalationRuleList []EscalationRule

// EscalationRule 单级升级规则
type EscalationRule struct {
	DelayMinutes int      `json:"delayMinutes"` // 延迟分钟数（从首次触发开始计算）
	ChannelIds   []uint64 `json:"channelIds"`   // 该级别的通知渠道
	ReceiverIds  []int64  `json:"receiverIds"`  // 该级别的额外接收人
}

const (
	AlertEscalationStatusEnable  int8 = 1
	AlertEscalationStatusDisable int8 = -1
)

type AlertEscalationQuery struct {
	model.PageParam

	Name    string `json:"name" form:"name"`
	Status  int8   `json:"status" form:"status"`
	Keyword string `json:"keyword" form:"keyword"`
}

// ---- EscalationRuleList JSON 序列化支持 ----

func (e EscalationRuleList) Value() (driver.Value, error) {
	if len(e) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("marshal EscalationRuleList: %w", err)
	}
	return string(b), nil
}

func (e *EscalationRuleList) Scan(val interface{}) error {
	if val == nil {
		*e = EscalationRuleList{}
		return nil
	}
	var b []byte
	switch v := val.(type) {
	case string:
		b = []byte(v)
	case []byte:
		b = v
	default:
		return fmt.Errorf("unsupported EscalationRuleList scan type: %T", val)
	}
	if len(b) == 0 {
		*e = EscalationRuleList{}
		return nil
	}
	return json.Unmarshal(b, e)
}

package entity

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"mayfly-go/pkg/model"
)

// AlertRule 告警规则
type AlertRule struct {
	model.Model

	Name     string        `json:"name" gorm:"size:100;not null;comment:规则名称"`
	Status   int8          `json:"status" gorm:"not null;default:1;index:idx_alert_rule_status;comment:状态 1:启用 -1:禁用"`
	Priority AlertPriority `json:"priority" gorm:"not null;default:2;comment:优先级 0:P0紧急 1:P1重要 2:P2一般 3:P3提示"`
	Remark   string        `json:"remark" gorm:"size:500;comment:备注"`

	// ---- 资产关联 ----
	ResourceType int8           `json:"resourceType" gorm:"not null;index:idx_alert_rule_rt;comment:资源类型，复用 consts.ResourceType*"`
	ScopeType    AlertScopeType `json:"scopeType" gorm:"not null;default:1;comment:关联范围类型 1:指定资源 2:标签路径"`
	ScopeValue   string         `json:"scopeValue" gorm:"size:2000;comment:关联范围值"`

	// 告警条件配置 (JSON) —— 列名用 alert_condition 避开 MySQL 保留字 condition
	Condition *AlertCondition `json:"condition" gorm:"column:alert_condition;type:json;serializer:json"`

	// 通知配置
	NotifyConfig *AlertNotifyConfig `json:"notifyConfig" gorm:"type:json;serializer:json"`

	// 高级配置
	EvalInterval  int `json:"evalInterval" gorm:"default:60;comment:评估间隔(秒)"`
	TriggerCount  int `json:"triggerCount" gorm:"default:1;comment:连续触发N次后才真正告警（防抖动）"`
	RecoveryCount int `json:"recoveryCount" gorm:"default:1;comment:连续恢复N次后才发恢复通知"`

	// 标签（用于静默规则 MatchLabels 匹配）
	Labels string `json:"labels" gorm:"type:json;comment:规则标签(JSON map)"`
}

func (a *AlertRule) TableName() string {
	return "t_alert_rule"
}

// AlertScopeType 关联范围类型
type AlertScopeType int8

const (
	// AlertScopeAll 该资源类型下的全部资源。
	// 仅静默规则允许（静默本就支持"整类资源静默"）；告警规则必须显式指定评估对象
	AlertScopeAll AlertScopeType = 0
	// AlertScopeResource 指定资源：ScopeValue 为资源ID
	AlertScopeResource AlertScopeType = 1
	// AlertScopeTagPath 标签路径：ScopeValue 为 codePath JSON 数组
	AlertScopeTagPath AlertScopeType = 2
)

// IsValidRuleScopeType 告警规则可用的范围类型（不含 AlertScopeAll）
func IsValidRuleScopeType(scopeType AlertScopeType) bool {
	return scopeType == AlertScopeResource || scopeType == AlertScopeTagPath
}

// AlertCondition 告警条件 - 支持多条件 AND/OR 组合
type AlertCondition struct {
	Operator string          `json:"operator"` // and / or
	Items    []ConditionItem `json:"items"`
}

// ConditionItem 条件项
type ConditionItem struct {
	Metric   string  `json:"metric"`   // 指标名: cpu_rate / mem_rate / disk_usage / load1 / status
	Compare  string  `json:"compare"`  // 比较方式: gt / gte / lt / lte / eq / neq
	Value    float64 `json:"value"`    // 阈值
	Duration int     `json:"duration"` // 持续时间(秒)，持续多久才触发
}

// 条件组合方式
const (
	ConditionOperatorAnd = "and"
	ConditionOperatorOr  = "or"
)

// ValidCompareOperators 合法的比较方式集合。
// 请求侧取值必须与该集合完全一致：求值时未知比较方式恒为 false，会导致规则静默不告警。
var ValidCompareOperators = []string{"gt", "gte", "lt", "lte", "eq", "neq"}

// IsValidCompare 比较方式是否合法
func IsValidCompare(compare string) bool {
	for _, c := range ValidCompareOperators {
		if c == compare {
			return true
		}
	}
	return false
}

// IsValidConditionOperator 组合方式是否合法
func IsValidConditionOperator(operator string) bool {
	return operator == ConditionOperatorAnd || operator == ConditionOperatorOr
}

// AlertNotifyConfig 通知配置（仅保留分组/重试时序参数，通知路由由独立的通知策略 AlertNotifyPolicy 负责）
type AlertNotifyConfig struct {
	GroupWait      int `json:"groupWait"`      // 分组等待时间(秒)
	GroupInterval  int `json:"groupInterval"`  // 分组发送间隔(秒)
	RepeatInterval int `json:"repeatInterval"` // 重复告警发送间隔(秒)
}

// AlertPriority 告警优先级
type AlertPriority int8

const (
	AlertPriorityCritical AlertPriority = 0 // P0 紧急
	AlertPriorityHigh     AlertPriority = 1 // P1 重要
	AlertPriorityMedium   AlertPriority = 2 // P2 一般
	AlertPriorityLow      AlertPriority = 3 // P3 提示
)

// IsValid 优先级是否落在 P0~P3 区间内
func (p AlertPriority) IsValid() bool {
	return p >= AlertPriorityCritical && p <= AlertPriorityLow
}

// Code 返回优先级的语言中立标识（P0~P3），用于通知内容与跨端展示，避免中文文案硬编码
func (p AlertPriority) Code() string {
	return fmt.Sprintf("P%d", int(p))
}

// AlertRuleStatus 告警规则状态
const (
	AlertRuleStatusEnable  int8 = 1  // 启用
	AlertRuleStatusDisable int8 = -1 // 禁用
)

// ---- AlertCondition JSON 序列化支持 ----

func (c AlertCondition) Value() (driver.Value, error) {
	if c.Operator == "" && len(c.Items) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("marshal AlertCondition: %w", err)
	}
	return string(b), nil
}

func (c *AlertCondition) Scan(val interface{}) error {
	if val == nil {
		*c = AlertCondition{}
		return nil
	}
	var b []byte
	switch v := val.(type) {
	case string:
		b = []byte(v)
	case []byte:
		b = v
	default:
		return fmt.Errorf("unsupported AlertCondition scan type: %T", val)
	}
	if len(b) == 0 {
		*c = AlertCondition{}
		return nil
	}
	return json.Unmarshal(b, c)
}

// ---- AlertNotifyConfig JSON 序列化支持 ----

func (n AlertNotifyConfig) Value() (driver.Value, error) {
	if n.GroupWait == 0 && n.GroupInterval == 0 && n.RepeatInterval == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(n)
	if err != nil {
		return nil, fmt.Errorf("marshal AlertNotifyConfig: %w", err)
	}
	return string(b), nil
}

func (n *AlertNotifyConfig) Scan(val interface{}) error {
	if val == nil {
		*n = AlertNotifyConfig{}
		return nil
	}
	var b []byte
	switch v := val.(type) {
	case string:
		b = []byte(v)
	case []byte:
		b = v
	default:
		return fmt.Errorf("unsupported AlertNotifyConfig scan type: %T", val)
	}
	if len(b) == 0 {
		*n = AlertNotifyConfig{}
		return nil
	}
	return json.Unmarshal(b, n)
}

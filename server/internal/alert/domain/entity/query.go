package entity

import (
	"mayfly-go/pkg/model"
)

// AlertRuleQuery 告警规则查询条件
type AlertRuleQuery struct {
	model.PageParam

	Name         string `json:"name" form:"name"`
	Status       int8   `json:"status" form:"status"`
	ResourceType int8   `json:"resourceType" form:"resourceType"`
	Keyword      string `json:"keyword" form:"keyword"`
}

// AlertInhibitionQuery 抑制规则查询条件
type AlertInhibitionQuery struct {
	model.PageParam

	Name   string `json:"name" form:"name"`
	Status int8   `json:"status" form:"status"`
}

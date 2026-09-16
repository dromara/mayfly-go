package entity

import "mayfly-go/pkg/model"

// LabelQuery 标签查询条件
type LabelQuery struct {
	model.PageParam

	LabelKey string `json:"labelKey" query:"labelKey"` // 标签键过滤
}

package entity

import (
	"mayfly-go/pkg/model"
)

// 标签资源关联
type TagResource struct {
	model.Model

	TagId        uint64 `json:"tagId"`
	TagPath      string `json:"tagPath"`      // 标签路径
	ResourceCode string `json:"resourceCode"` // 资源标识
	ResourceType int8   `json:"resourceType"` // 资源类型
}

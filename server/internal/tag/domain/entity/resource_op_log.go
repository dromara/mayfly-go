package entity

import "mayfly-go/pkg/model"

// 资源操作日志记录
type ResourceOpLog struct {
	model.CreateModel

	CodePath     string `json:"codePath"`     // 标签路径
	ResourceCode string `json:"resourceCode"` // 资源编号
	ResourceType int8   `json:"relateType"`   // 资源类型
}

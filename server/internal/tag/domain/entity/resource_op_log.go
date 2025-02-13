package entity

import "mayfly-go/pkg/model"

// 资源操作日志记录
type ResourceOpLog struct {
	model.CreateModel

	CodePath     string `json:"codePath" gorm:"size:255;not null;"`    // 标签路径
	ResourceCode string `json:"resourceCode" gorm:"size:50;not null;"` // 资源编号
	ResourceType int8   `json:"relateType" gorm:"not null;"`           // 资源类型
}

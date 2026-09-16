package entity

import (
	"mayfly-go/pkg/model"
)

// Label 全局标签注册表
// 扁平化键值对标签，用于告警路由匹配、资源分类标记等场景。
// 与 TagTree（层级资源分组）完全独立，无数据关联。
type Label struct {
	model.Model
	model.ExtraData // 扩展字段：color、description 等

	LabelKey   string `json:"labelKey" gorm:"size:100;not null;uniqueIndex:idx_label_kv;comment:标签键"`
	LabelValue string `json:"labelValue" gorm:"size:200;not null;uniqueIndex:idx_label_kv;comment:标签值"`
}

func (l *Label) TableName() string {
	return "t_label"
}

// ExtraData 中的扩展字段 key
const (
	LabelExtraKeyColor       = "color"       // 显示颜色
	LabelExtraKeyDescription = "description" // 描述
)

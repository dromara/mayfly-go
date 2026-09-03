package entity

import (
	"mayfly-go/pkg/model"
)

// Memory 长期记忆实体（对齐 tokhub t_memory，按本系统记忆域词汇裁剪）
//
// 记忆由系统（LLM 提取器 / 记忆工具）按用户维度读写：
//   - user_id 为隔离维度，跨会话、跨实例共享
//   - tags 存 JSON 数组字符串，标签过滤经 LIKE '%"tag"%' 匹配
//   - 检索排序依赖 update_time（近期记忆优先），无 embedding 列，
//     语义检索后续可经 extra 扩展或独立向量介质叠加
type Memory struct {
	model.Model

	// UserId 所属用户 ID（隔离维度）
	UserId string `gorm:"column:user_id;size:32;not null;comment:用户ID" json:"userId"`
	// Type 记忆类型：preference/fact/skill/experience
	Type string `gorm:"column:type;size:32;not null;default:fact;comment:记忆类型" json:"type"`
	// Content 记忆内容（自然语言描述）
	Content string `gorm:"column:content;type:text;not null;comment:记忆内容" json:"content"`
	// Tags 标签 JSON 数组字符串，如 ["editor","preference"]
	Tags string `gorm:"column:tags;size:512;comment:标签JSON数组" json:"tags"`

	model.ExtraData
}

func (m *Memory) TableName() string {
	return "t_ai_memory"
}

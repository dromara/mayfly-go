package memory

import (
	"context"
	"time"
)

// DefaultStore 默认记忆存储。
// 多实例部署由 application 层装配为 DB 后端实现（t_ai_memory 表，跨实例共享）；
// 未装配时由 agent 装配器降级为本地 JSONL 存储（仅适用于单实例部署/单测）
var DefaultStore Store

// MemoryItem 记忆项（长期记忆）
type MemoryItem struct {
	ID        string            `json:"id"`                 // 唯一标识
	UserID    string            `json:"userId"`             // 用户ID
	Type      string            `json:"type"`               // 记忆类型: preference/fact/skill/experience
	Content   string            `json:"content"`            // 记忆内容（自然语言描述）
	Tags      []string          `json:"tags"`               // 标签，用于分类和检索
	CreatedAt time.Time         `json:"createdAt"`          // 创建时间
	UpdatedAt time.Time         `json:"updatedAt"`          // 更新时间
	Metadata  map[string]string `json:"metadata,omitempty"` // 元数据（来源会话ID、提取时间等）
}

// Store 记忆存储接口
type Store interface {
	// 长期记忆操作
	GetByUser(ctx context.Context, userID string, tags []string) ([]*MemoryItem, error)
	Save(ctx context.Context, items []*MemoryItem) error
	Delete(ctx context.Context, userID string, ids []string) error

	// 向量检索（用于语义搜索）
	Search(ctx context.Context, userID string, query string, limit int) ([]*MemoryItem, error)
}

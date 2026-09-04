package session

import (
	"mayfly-go/pkg/utils/collx"
	"time"
)

// SessionMeta 会话元数据
type SessionMeta struct {
	UserId     string    `json:"userId"`
	Key        string    `json:"key"` // 会话唯一标识
	Summary    string    `json:"summary"`
	Count      int       `json:"count"`      // 消息总数
	TokenCount int       `json:"tokenCount"` // 总 token 数
	Skip       int       `json:"skip"`
	Extra      collx.M   `json:"extra,omitempty"` // 扩展字段
	CreatedAt  time.Time `json:"createdAt"`       // 创建时间戳
	UpdatedAt  time.Time `json:"updatedAt"`       // 最后更新时间戳
}

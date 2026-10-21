package session

import (
	"context"
	"errors"
)

var (
	// DefaultSessionStore 默认会话存储
	DefaultSessionStore Store

	// ErrStoreUnsupported 当前后端不支持该能力（显式失败，不静默返回空结果）。
	//
	// 典型：JSONL 后端是单文件历史，MessageQuery 里没有 sessionKey，无法定位到具体会话；
	// 旧实现用空 key 读历史后返回空切片——调用方看到「查不到」而不是「不支持」，
	// 审批/补全的 resume 信息落不了库却查不出原因。生产走 DB 后端，此错误只会在兜底后端出现
	ErrStoreUnsupported = errors.New("session store backend does not support this operation")
)

type MessageQuery struct {
	ActionId    string
	TurnId      string
	ToolCallId  string
	MessageType string
}

// Store 会话相关信息存储
type Store interface {
	// ============= 消息操作  =============

	// AppendMsgs 追加消息到会话末尾
	AppendMsgs(ctx context.Context, sessionKey string, msgs ...*Message) error
	// GetHistory 获取会话历史消息（按时间正序排列）
	//
	// 语义：返回**未被摘要覆盖**的最近 limit 条消息（limit<=0 表示不限）；Skip 的截断由 Store
	// 自己完成——两个后端必须同一语义，Manager 不得再用元数据里的 Count 做算术
	GetHistory(ctx context.Context, sessionKey string, limit int) ([]*Message, error)
	// CountUnsummarized 返回尚未被摘要覆盖的消息条数（与 GetHistory 的展开口径一致：
	// message 计 1、tool_call 计 2，internal 等不进模型上下文的不计）。
	//
	// 为何需要它：元数据里的 Count 是 AppendMsgs 的累加快照，与 TurnItem 展开出的真实条数
	// 并不相等（中断派生消息、历史数据、未走 AppendMsgs 的写入路径）——拿它当读取上界会
	// 静默截断上下文（实测某会话 160 条只读回 5 条），Count<=Skip 时甚至直接返回空历史
	CountUnsummarized(ctx context.Context, sessionKey string) (int, error)
	// ClearHistory 清空会话历史消息
	ClearHistory(ctx context.Context, sessionKey string) error

	// GetMessage 根据查询条件获取单条消息
	GetMessage(ctx context.Context, query *MessageQuery) ([]*Message, error)
	// UpdateMessage 更新单条消息
	UpdateMessage(ctx context.Context, msg *Message) error

	// ============= 元数据操作  =============

	// ListMetas 列出所有会话元信息
	ListMetas(ctx context.Context) ([]*SessionMeta, error)
	// GetMeta 获取会话元信息
	GetMeta(ctx context.Context, sessionKey string) (*SessionMeta, error)
	// SaveMeta 保存会话元信息
	SaveMeta(ctx context.Context, meta *SessionMeta) error
	// DeleteMeta 删除会话元信息
	DeleteMeta(ctx context.Context, sessionKey string) error
}

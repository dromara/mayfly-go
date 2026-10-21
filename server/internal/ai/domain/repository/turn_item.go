package repository

import (
	"context"
	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/pkg/base"
)

type TurnItem interface {
	base.Repo[*entity.TurnItem]

	// SelectByTurnId 按 turn 查询 item 列表
	SelectByTurnId(ctx context.Context, conversationId uint64, turnId string) ([]*entity.TurnItem, error)

	// SelectRecentByConvId 查询会话最近的 items（排除不产生消息的 reasoning 与 turn_notice，
	// 按有序 item_id 倒序取 limit 条）
	// 供 session 存储重建 LLM 上下文。注意：留下的 item 并不都是 1 条消息——tool_call 会展开成
	// 两条（助手调用 + 工具结果），所以取 limit 条转换后只会不少于而不会恰好，裁剪必须按消息组对齐；
	// 而完全产不出消息的类型必须在本查询里排除，否则它们会白占窗口槽位、挤掉真实上下文
	SelectRecentByConvId(ctx context.Context, conversationId uint64, limit int) ([]*entity.TurnItem, error)

	// SelectTurnGroups 按 turn 分组加载（分页），返回每个 turn 的 item 列表
	SelectTurnGroups(ctx context.Context, conversationId uint64, beforeTurnId string, limit int) ([]*TurnGroup, error)

	// BatchInsert 批量插入 TurnItem
	BatchInsert(ctx context.Context, items []*entity.TurnItem) error
}

// TurnGroup turn 分组，包含 turn 内所有 item
type TurnGroup struct {
	TurnId string             `json:"turnId"`
	Items  []*entity.TurnItem `json:"items"`
}

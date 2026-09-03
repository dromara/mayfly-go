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

	// SelectRecentByConvId 查询会话最近的 items（排除 reasoning，按 id 倒序取 limit 条）
	// 供 session 存储重建 LLM 上下文：每个非 reasoning item 至少产生一条消息，
	// 取最近 limit 个 item 转换后必然不少于 limit 条消息，可安全裁剪
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

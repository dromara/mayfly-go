package application

import (
	"context"
	"fmt"

	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/domain/repository"
	"mayfly-go/internal/ai/protocol"
	"mayfly-go/pkg/errorx"
)

// TurnItem TurnItem 应用接口
type TurnItem interface {
	// BatchSaveTurnItems 批量保存 TurnItem
	BatchSaveTurnItems(ctx context.Context, items []*entity.TurnItem) error

	// SelectByTurnId 查询指定 turn 的 item 列表
	SelectByTurnId(ctx context.Context, conversationId uint64, turnId string) ([]*entity.TurnItem, error)

	// UpdateResumedToolCallItem 恢复路径将挂起的 tool_call item 更新至真实终态
	// （update_item_status：按原 item 定位同一行，status/payload 回写为
	// 真实执行结果，extra 仅 merge interrupt.resume，保留挂起时的 kind/request_id 等）
	UpdateResumedToolCallItem(ctx context.Context, conversationId uint64, item *entity.TurnItem, resume *protocol.InterruptResume) error
}

// turnItemAppImpl TurnItem 应用实现
type turnItemAppImpl struct {
	turnItemRepo repository.TurnItem `inject:"T"`
}

var _ TurnItem = (*turnItemAppImpl)(nil)

// BatchSaveTurnItems 批量保存 TurnItem
func (t *turnItemAppImpl) BatchSaveTurnItems(ctx context.Context, items []*entity.TurnItem) error {
	return t.turnItemRepo.BatchInsert(ctx, items)
}

// SelectByTurnId 查询指定 turn 的 item 列表
func (t *turnItemAppImpl) SelectByTurnId(ctx context.Context, conversationId uint64, turnId string) ([]*entity.TurnItem, error) {
	return t.turnItemRepo.SelectByTurnId(ctx, conversationId, turnId)
}

// UpdateResumedToolCallItem 恢复路径更新挂起的 tool_call item 至真实终态
func (t *turnItemAppImpl) UpdateResumedToolCallItem(ctx context.Context, conversationId uint64, item *entity.TurnItem, resume *protocol.InterruptResume) error {
	row := &entity.TurnItem{ConversationId: conversationId, ItemId: item.ItemId}
	if err := t.turnItemRepo.GetByCond(row); err != nil {
		return err
	}
	if row.Id == 0 {
		return errorx.NewBiz(fmt.Sprintf("resumed tool_call item not found, itemId=%s", item.ItemId))
	}

	row.Status = item.Status
	row.Payload = item.Payload

	// extra merge_patch 语义：仅回填 interrupt.resume，
	// 保留挂起时写入的 kind/request_id/metadata 等字段
	if resume != nil {
		info := interruptInfoOf(row)
		if info == nil {
			info = &protocol.InterruptInfo{Resume: resume}
			if row.Extra == nil {
				row.Extra = make(map[string]any)
			}
			row.Extra["interrupt"] = info
		} else if info.Resume == nil {
			info.Resume = resume
			row.Extra["interrupt"] = info
		}
	}
	return t.turnItemRepo.UpdateById(ctx, row, "status", "payload", "extra")
}

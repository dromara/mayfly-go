package persistence

import (
	"context"
	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/domain/repository"
	"mayfly-go/pkg/base"
)

type turnItemRepoImpl struct {
	base.RepoImpl[*entity.TurnItem]
}

var _ repository.TurnItem = (*turnItemRepoImpl)(nil)

func newTurnItemRepo() repository.TurnItem {
	return &turnItemRepoImpl{}
}

func (t *turnItemRepoImpl) SelectByTurnId(ctx context.Context, conversationId uint64, turnId string) ([]*entity.TurnItem, error) {
	var items []*entity.TurnItem
	// 按 item_id 排序而非自增 id：item_id 是触发时刻生成的有序 UUID（SortableUUID），
	// 字典序 == 触发时序；落库顺序（写入序）与触发顺序解耦，不能作为排序依据
	if err := t.SelectBySql("SELECT * FROM t_ai_turn_item WHERE conversation_id = ? AND turn_id = ? AND is_deleted = 0 ORDER BY item_id ASC", &items, conversationId, turnId); err != nil {
		return nil, err
	}
	return items, nil
}

func (t *turnItemRepoImpl) SelectRecentByConvId(ctx context.Context, conversationId uint64, limit int) ([]*entity.TurnItem, error) {
	var items []*entity.TurnItem
	sql := "SELECT * FROM t_ai_turn_item WHERE conversation_id = ? AND item_type != ? AND is_deleted = 0 ORDER BY item_id DESC LIMIT ?"
	if err := t.SelectBySql(sql, &items, conversationId, entity.ItemTypeReasoning, limit); err != nil {
		return nil, err
	}
	// 倒序取出的结果反转为时间正序
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	return items, nil
}

func (t *turnItemRepoImpl) SelectTurnGroups(ctx context.Context, conversationId uint64, beforeTurnId string, limit int) ([]*repository.TurnGroup, error) {
	// 使用有序 UUID（SortableUUID），turn_id 字典序 == 时间序，可直接 ORDER BY
	var turnIds []string
	sql := "SELECT turn_id FROM t_ai_turn_item WHERE conversation_id = ? AND is_deleted = 0 GROUP BY turn_id"
	args := []any{conversationId}

	if beforeTurnId != "" {
		// 有序 UUID 可直接字符串比较：查找比 beforeTurnId 更早的 turn
		sql += " HAVING MIN(turn_id) < ?"
		args = append(args, beforeTurnId)
	}

	sql += " ORDER BY MIN(turn_id) DESC LIMIT ?"
	args = append(args, limit)

	if err := t.SelectBySql(sql, &turnIds, args...); err != nil {
		return nil, err
	}

	if len(turnIds) == 0 {
		return []*repository.TurnGroup{}, nil
	}

	// 按 turn_id 查询每个 turn 的 items（按 item_id ASC 保证 turn 内触发顺序）
	var groups []*repository.TurnGroup
	for _, turnId := range turnIds {
		items, err := t.SelectByTurnId(ctx, conversationId, turnId)
		if err != nil {
			return nil, err
		}
		groups = append(groups, &repository.TurnGroup{
			TurnId: turnId,
			Items:  items,
		})
	}

	return groups, nil
}

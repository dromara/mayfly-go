package persistence

import (
	"context"
	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/domain/repository"
	"mayfly-go/pkg/base"
	"strings"
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
	if err := t.SelectBySQL(ctx, "SELECT * FROM t_ai_turn_item WHERE conversation_id = ? AND turn_id = ? AND is_deleted = 0 ORDER BY item_id ASC", &items, conversationId, turnId); err != nil {
		return nil, err
	}
	return items, nil
}

// nonMessageItemTypes 完全产不出 LLM 消息的 item 类型：reasoning 是过路推理，turn_notice 是
// 纯展示元数据（「本轮被人为停止」），context_compaction 是压缩记账。
//
// 三者都进不了 application/session_store.go 的 itemToMessages 任何分支，但仍会被历史查询取回来
// 并各占一个 limit 窗口槽位——表现是「多停几次、多压几次，模型逐渐忘掉前面的对话」，极难反查回一条 SQL。
// 新登记一个不产消息的类型只需往本列表加一项，守卫会对着它断言 SQL 里的占位符个数
var nonMessageItemTypes = []string{
	entity.ItemTypeReasoning,
	entity.ItemTypeTurnNotice,
	entity.ItemTypeCompaction,
}

// nonMessagePlaceholders 按排除集长度生成 `?, ?, ?` 形式的占位符串，与实参个数天然一致
func nonMessagePlaceholders() string {
	marks := make([]string, 0, len(nonMessageItemTypes))
	for range nonMessageItemTypes {
		marks = append(marks, "?")
	}
	return strings.Join(marks, ", ")
}

func (t *turnItemRepoImpl) SelectRecentByConvId(ctx context.Context, conversationId uint64, limit int) ([]*entity.TurnItem, error) {
	var items []*entity.TurnItem
	// 参数与占位符逐个对应：会话 + 排除集 + limit（展开后的实参不能靠人手工对齐）
	args := make([]any, 0, len(nonMessageItemTypes)+2)
	args = append(args, conversationId)
	for _, itemType := range nonMessageItemTypes {
		args = append(args, itemType)
	}
	args = append(args, limit)

	sql := "SELECT * FROM t_ai_turn_item WHERE conversation_id = ? AND item_type NOT IN (" + nonMessagePlaceholders() + ") AND is_deleted = 0 ORDER BY item_id DESC LIMIT ?"
	if err := t.SelectBySQL(ctx, sql, &items, args...); err != nil {
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

	if err := t.SelectBySQL(ctx, sql, &turnIds, args...); err != nil {
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

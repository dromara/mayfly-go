package application

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/domain/repository"
)

// fakeHistoryTurnItemRepo 只提供 GetHistory/CountUnsummarized 用到的方法（其余方法由嵌入接口兜底，
// 接口漂移时编译期即报错）
type fakeHistoryTurnItemRepo struct {
	repository.TurnItem
	items []*entity.TurnItem
}

func (f *fakeHistoryTurnItemRepo) SelectRecentByConvId(context.Context, uint64, int) ([]*entity.TurnItem, error) {
	return f.items, nil
}

// SelectByCond 供 CountUnsummarized 按真实 item 统计上下文消息数（仅用到 item_type 列）
func (f *fakeHistoryTurnItemRepo) SelectByCond(any, ...string) ([]*entity.TurnItem, error) {
	return f.items, nil
}

// fakeConvRepoMeta 只回答会话元数据（Skip/摘要），供 CountUnsummarized 取已摘要偏移。
// found=false 时模拟「会话行不存在」：GetMeta 返回 (nil,nil)，口径等同 skip=0
type fakeConvRepoMeta struct {
	repository.Conversation
	skip  int
	found bool
}

func (f *fakeConvRepoMeta) GetByCond(cond any) error {
	if !f.found {
		return errFakeConvNotFound
	}
	if conv, ok := cond.(*entity.Conversation); ok {
		conv.CompactionMessageCount = f.skip
		// 故意与真实条数不一致：这正是生产里的漂移形态（count 累加快照不等于 item 展开数）
		conv.TotalMessageCount = 0
	}
	return nil
}

var errFakeConvNotFound = fmt.Errorf("fake conversation not found")

// histMsgItem 构造一条 message item（role: user / assistant）
func histMsgItem(id uint64, turnId, itemId, role, text string) *entity.TurnItem {
	return &entity.TurnItem{
		Id: id, TurnId: turnId, ItemId: itemId, ItemType: entity.ItemTypeMessage,
		Payload: fmt.Sprintf(`{"type":"message","id":%q,"role":%q,"content":[{"type":"output_text","text":%q}]}`,
			itemId, role, text),
	}
}

// histToolItem 构造一条 tool_call item —— 重建 LLM 上下文时会展开成两条消息
// （助手 tool_calls + 工具结果），这正是「按条数裁剪」会切坏的形态
func histToolItem(id uint64, turnId, itemId, callId, toolName string) *entity.TurnItem {
	return &entity.TurnItem{
		Id: id, TurnId: turnId, ItemId: itemId, ItemType: entity.ItemTypeToolCall,
		Payload: fmt.Sprintf(`{"type":"tool_call","id":%q,"tool_name":%q,"tool_call_id":%q,"arguments":"{}","output":"ok","status":"success"}`,
			itemId, toolName, callId),
	}
}

// TestGetHistoryNeverStartsWithOrphanToolResult 真实 GetHistory 路径不得产出孤立工具结果开头。
//
// 这是把「裁剪按消息组对齐」钉在**生产函数**上，而不是只钉在辅助函数上：
// 上一轮只测函数的变异存活过（有人把调用点改回 msgs[n-limit:] 时函数用例照样全绿）。
// 仓储接口注释里那句「每个非 reasoning item 至少产生一条消息……可安全裁剪」正是错的判断，
// tool_call item 产生两条，按条数硬切会留下没有前置助手调用的 tool 消息
func TestGetHistoryNeverStartsWithOrphanToolResult(t *testing.T) {
	items := []*entity.TurnItem{
		histMsgItem(1, "t1", "i1", "user", "问题一"),
		histToolItem(2, "t1", "i2", "call-1", "DbQueryTables"),
		histToolItem(3, "t1", "i3", "call-2", "DbQueryTables"),
		histMsgItem(4, "t1", "i4", "assistant", "答案一"),
	}
	store := &sessionStoreImpl{turnItemRepo: &fakeHistoryTurnItemRepo{items: items}, conversationRepo: &fakeConvRepoMeta{}}

	// 逐个 limit 扫：任何窗口大小下，历史首条都不能是 tool 角色（否则就是非法消息序列）
	for limit := 1; limit <= 7; limit++ {
		msgs, err := store.GetHistory(context.Background(), "conv:1", limit)
		require.NoError(t, err, "limit=%d", limit)
		require.NotEmpty(t, msgs, "limit=%d 不应返回空历史", limit)
		require.NotEqual(t, schema.Tool, msgs[0].Role,
			"limit=%d 时历史以孤立工具结果开头，模型 API 会拒或语义断裂", limit)
	}
}

// TestGetHistoryKeepsToolCallWithItsResult 保留工具结果时必须连着它的助手调用一起保留。
func TestGetHistoryKeepsToolCallWithItsResult(t *testing.T) {
	items := []*entity.TurnItem{
		histMsgItem(1, "t1", "i1", "user", "问题一"),
		histToolItem(2, "t1", "i2", "call-1", "DbQueryTables"),
		histToolItem(3, "t1", "i3", "call-2", "MachineCommandExec"),
	}
	store := &sessionStoreImpl{turnItemRepo: &fakeHistoryTurnItemRepo{items: items}, conversationRepo: &fakeConvRepoMeta{}}

	// 5 条消息（1 user + 2×2），limit=2 会切到 call-2 的助手调用上
	msgs, err := store.GetHistory(context.Background(), "conv:1", 2)
	require.NoError(t, err)
	require.Len(t, msgs, 2, "切点本就落在成对位置时按 limit 精确裁剪")
	require.Equal(t, schema.Assistant, msgs[0].Role)
	require.NotEmpty(t, msgs[0].ToolCalls, "助手调用必须带 tool_calls")
	require.Equal(t, schema.Tool, msgs[1].Role)
	require.Equal(t, msgs[0].ToolCalls[0].ID, msgs[1].ToolCallId, "工具结果必须与前面的助手调用配对")
}

// TestGetHistoryMissingToolCallIdSkipped arguments/toolCallId 缺失的历史脏数据不得产出半条消息。
func TestGetHistoryMissingToolCallIdSkipped(t *testing.T) {
	items := []*entity.TurnItem{
		histMsgItem(1, "t1", "i1", "user", "问题一"),
		{Id: 2, TurnId: "t1", ItemId: "i2", ItemType: entity.ItemTypeToolCall,
			Payload: `{"type":"tool_call","id":"i2","tool_name":"X","tool_call_id":"","arguments":"{}"}`},
	}
	store := &sessionStoreImpl{turnItemRepo: &fakeHistoryTurnItemRepo{items: items}, conversationRepo: &fakeConvRepoMeta{}}

	msgs, err := store.GetHistory(context.Background(), "conv:1", 10)
	require.NoError(t, err)
	require.Len(t, msgs, 1, "无 toolCallId 的脏 item 整条跳过，不留半条消息")
	require.Equal(t, schema.User, msgs[0].Role)
}

// TestCountUnsummarizedUsesItemTruthNotMetaCount 未摘要条数必须以 TurnItem 的展开数为准，
// 不能用元数据里的 Count（AppendMsgs 的累加快照）。
//
// 真机证据（本机库）：多个会话 total_message_count=0 却有 15–40 条消息；某会话
// count=128/skip=123 而实际展开 160 条——旧读路径按 Count-Skip 只取 5 条、静默丢 32 条，
// 而 count=0<=skip=0 时 Manager 直接返回空历史（模型完全失忆）。
func TestCountUnsummarizedUsesItemTruthNotMetaCount(t *testing.T) {
	// 展开后 4 条上下文消息：user 1 + tool_call 2 + assistant 1
	items := []*entity.TurnItem{
		histMsgItem(1, "t1", "i1", "user", "问题一"),
		histToolItem(2, "t1", "i2", "call-1", "DbQueryTables"),
		histMsgItem(3, "t1", "i3", "assistant", "答案一"),
	}

	// 元数据 Count=0（漂移形态）、Skip=1
	store := &sessionStoreImpl{
		turnItemRepo:     &fakeHistoryTurnItemRepo{items: items},
		conversationRepo: &fakeConvRepoMeta{skip: 1, found: true},
	}

	n, err := store.CountUnsummarized(context.Background(), "conv:1")
	require.NoError(t, err)
	require.Equal(t, 3, n, "4 条展开消息被摘要掉 1 条后应剩 3 条；不得用 Count=0 算成 0")

	// limit=0（不限）时必须读回全部未摘要消息，而不是因 Count<=Skip 返回空
	msgs, err := store.GetHistory(context.Background(), "conv:1", 0)
	require.NoError(t, err)
	require.Len(t, msgs, 3, "未摘要历史不得因元数据漂移而被清空")
	require.NotEqual(t, schema.Tool, msgs[0].Role, "裁剪仍须按消息组对齐，不能以孤立工具结果开头")

	// 全部已摘要（skip == 总条数）时返回空，由 Manager 前置摘要
	all := &sessionStoreImpl{
		turnItemRepo:     &fakeHistoryTurnItemRepo{items: items},
		conversationRepo: &fakeConvRepoMeta{skip: 4, found: true},
	}
	n, err = all.CountUnsummarized(context.Background(), "conv:1")
	require.NoError(t, err)
	require.Zero(t, n)
	msgs, err = all.GetHistory(context.Background(), "conv:1", 0)
	require.NoError(t, err)
	require.Empty(t, msgs)
}

// TestInterruptedToolItemRebuildsWithNeutralMarker 中断且未执行的历史项重建时必须给中性标记。
//
// 旧实现把中断说明当 output；现在 output 为空，重建若直接透传空串，部分网关会拒绝空 tool 内容，
// 而写回「请审批」又会让模型重复发起。中性标记 [interrupted] 同时避开这两个坑。
func TestInterruptedToolItemRebuildsWithNeutralMarker(t *testing.T) {
	item := &entity.TurnItem{
		Id: 9, TurnId: "t1", ItemId: "i9", ItemType: entity.ItemTypeToolCall,
		Status: "interrupted",
		Payload: `{"type":"tool_call","id":"i9","tool_name":"ExecSql","tool_call_id":"call-9",` +
			`"arguments":"{}","output":"","status":"interrupted"}`,
	}
	store := &sessionStoreImpl{
		turnItemRepo:     &fakeHistoryTurnItemRepo{items: []*entity.TurnItem{item}},
		conversationRepo: &fakeConvRepoMeta{},
	}
	msgs, err := store.GetHistory(context.Background(), "conv:1", 10)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	require.Equal(t, "[interrupted]", msgs[1].Content)
}

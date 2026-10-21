package application

import (
	"context"
	"testing"

	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/protocol"

	"github.com/stretchr/testify/require"
)

// fakeNoticeTurnItemApp 只实现 recorder 收尾真正会调的两个方法，其余由嵌入接口兜底
// （接口漂移时编译期即报错，不靠 grep 找实现点）
type fakeNoticeTurnItemApp struct {
	TurnItem

	saved []*entity.TurnItem
}

func (f *fakeNoticeTurnItemApp) BatchSaveTurnItems(_ context.Context, items []*entity.TurnItem) error {
	f.saved = append(f.saved, items...)
	return nil
}

// TestCollectTurnNoticePersistsDisplayOnlyItem 用户停止要作为一条 turn_notice item 落库。
//
// 真机实测到的缺陷形态：停止提示此前只由服务端推一个事件给当前客户端、前端写进内存，
// 零持久化 —— 刷新后那一轮只剩一段在句子中途断开的回答（截断处还留着未闭合的 `**`），
// 与模型正常答完完全同形，回看的人会当成模型自己写崩了
func TestCollectTurnNoticePersistsDisplayOnlyItem(t *testing.T) {
	app := &fakeNoticeTurnItemApp{}
	recorder := NewTurnRecorder(app, 77, "turn-1", func(*protocol.EventMsg) {})

	recorder.CollectTurnNotice("turn-1", entity.TurnNoticeKindStopped)
	recorder.Flush(context.Background())

	require.Len(t, app.saved, 1, "停止提示必须真的被写出去")
	item := app.saved[0]
	require.Equal(t, entity.ItemTypeTurnNotice, item.ItemType)
	require.Equal(t, "turn-1", item.TurnId)
	require.EqualValues(t, 77, item.ConversationId)
	// 提示本身是一条已完成的记录事实，不参与状态机：不得用 active/pending，否则历史侧要再猜一次
	require.Equal(t, entity.ItemStatusSuccess, item.Status)
	require.NotEmpty(t, item.ItemId, "item_id 是历史排序键（有序 UUID），缺失会让提示落位随机")

	// 种类要能从 payload 复原：前端据此选文案，服务端不入库句子
	restored, err := protocol.FromPayload(item.ItemType, item.ItemId, item.Payload)
	require.NoError(t, err)
	require.Equal(t, entity.TurnNoticeKindStopped, restored.NoticeKind)
}

// TestTurnNoticeItemProducesNoModelMessage 停止提示绝不能变成喂回模型的一条消息。
//
// 它落进同一张 t_ai_turn_item、也会被历史重建读到；一旦 itemToMessages 给它配上分支，
// 下一轮请求里就多出一条内容为空的「助手发言」，模型会把它当成自己说过的话接着编
func TestTurnNoticeItemProducesNoModelMessage(t *testing.T) {
	store := &sessionStoreImpl{}
	item := &entity.TurnItem{
		Id: 1, TurnId: "turn-1", ItemId: "n-1", ItemType: entity.ItemTypeTurnNotice,
		Payload: (&protocol.TurnItem{NoticeKind: entity.TurnNoticeKindStopped}).PayloadJSON(),
	}

	require.Nil(t, store.itemToMessages(context.Background(), item),
		"turn_notice 是纯展示元数据，产不出消息才对；产出任何消息都会污染下一轮的模型上下文")
}

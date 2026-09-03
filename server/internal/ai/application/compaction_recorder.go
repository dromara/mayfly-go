package application

import (
	"context"

	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/domain/repository"
	"mayfly-go/internal/ai/protocol"
	"mayfly-go/internal/ai/session"
	"mayfly-go/pkg/eventbus"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/stringx"
)

// compactionRecorder 压缩事件项记录器
// 订阅 session 摘要完成事件，将每次上下文压缩以 context_compaction 类型的
// TurnItem 落库到当前 turn 时间线（对齐 tokhub 的 ContextCompaction item），
// 前端据此渲染"上下文已压缩"节点；payload 不参与 LLM 上下文重建
type compactionRecorder struct {
	turnItemRepo repository.TurnItem `inject:"T"`
}

// registerCompactionRecorder 注册压缩事件项订阅（幂等：固定 subId 覆盖注册）
func registerCompactionRecorder(recorder *compactionRecorder) {
	session.EventBus.SubscribeAsync(session.EventTopicSummarized, "AiCompactionRecorder",
		func(ctx context.Context, event *eventbus.Event[any]) error {
			evt, ok := event.Val.(*session.SummarizedEvent)
			if !ok {
				return nil
			}

			convId, err := parseConvId(evt.SessionKey)
			if err != nil {
				// 非 conv: 前缀的会话键（CLI/测试等场景）无 TurnItem 时间线，跳过
				return nil
			}

			if err := recorder.record(ctx, convId, evt); err != nil {
				logx.WarnfContext(ctx, "[CompactionRecorder] record compaction item failed, convId=%d, err=%v", convId, err)
			}
			// 落库失败不阻塞事件总线（fail-open）
			return nil
		}, false)
}

// record 构建并落库压缩事件项，归属到会话最新的 turn
func (r *compactionRecorder) record(ctx context.Context, convId uint64, evt *session.SummarizedEvent) error {
	// 取会话最新 item 的 turnId：压缩由当前轮上下文积累触发，事件归属当前 turn
	items, err := r.turnItemRepo.SelectRecentByConvId(ctx, convId, 1)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		logx.WarnfContext(ctx, "[CompactionRecorder] no turn item found, skip compaction record, convId=%d", convId)
		return nil
	}

	itemId := stringx.SortableUUID()
	compaction := protocol.NewCompactionTurnItem(itemId, evt.OriginalTokens, evt.CompressedTokens, evt.CompressedMessageCount)

	item := &entity.TurnItem{
		ConversationId: convId,
		TurnId:         items[0].TurnId,
		ItemType:       entity.ItemTypeCompaction,
		ItemId:         itemId,
		Payload:        compaction.PayloadJSON(),
		Status:         protocol.TurnItemStatusSuccess,
	}
	return r.turnItemRepo.BatchInsert(ctx, []*entity.TurnItem{item})
}

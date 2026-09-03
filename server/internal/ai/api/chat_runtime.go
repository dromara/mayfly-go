package api

import (
	"context"
	"errors"
	"sync"

	"mayfly-go/internal/ai/api/vo"
	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/protocol"
)

// ==================== Turn 运行时（事件生产者与 WS 连接解耦） ====================
//
// 对齐 tokhub 的 turn registry + replay 模式：
//   - agent Run 在独立 goroutine 中执行，事件 publish 到 turn 级事件总线（全量缓冲）
//   - WS 连接只是订阅者，写失败/断连不再影响 agent 运行（刷新页面 turn 继续跑）
//   - 重连后 attach 该 turn：先回放缓冲快照，再持续订阅实时事件，流式输出无缝续上
//   - 显式 stop 消息触发 ctx 取消，是唯一真正中断 agent 的途径

// errTurnRunning 该会话已有运行中 turn
var errTurnRunning = errors.New("conversation has a running turn")

// turnEventBus 单个运行中 turn 的事件广播器：全量缓冲（供 attach 回放）+ 多订阅者实时投递
//
// 并发协议（保证回放与实时事件无重叠无丢失）：
//   - Subscribe 在锁内完成「快照 buf + 注册订阅者」，此后 Publish 的事件只进订阅者 channel；
//   - Publish 在锁内完成「append buf + 投递订阅者」。
//   - 两操作互斥，快照与 channel 内事件天然不重叠。
type turnEventBus struct {
	mu          sync.Mutex
	buf         []*protocol.EventMsg
	subscribers map[chan *protocol.EventMsg]struct{}
	done        chan struct{} // 关闭表示 turn 结束（收尾事件已全部发布）
	doneClosed  bool
}

// subscriberChan 订阅者 channel 容量：慢消费者缓冲上限，超出则被移除（断连重连后 attach 恢复）
const subscriberChanSize = 1024

func newTurnEventBus() *turnEventBus {
	return &turnEventBus{
		subscribers: make(map[chan *protocol.EventMsg]struct{}),
		done:        make(chan struct{}),
	}
}

// Publish 发布事件：追加缓冲 + 投递全部订阅者（非阻塞，满则移除慢订阅者）
func (b *turnEventBus) Publish(evt *protocol.EventMsg) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.doneClosed {
		return
	}
	b.buf = append(b.buf, evt)
	for ch := range b.subscribers {
		select {
		case ch <- evt:
		default:
			// 慢消费者：移除并关闭其 channel，泵感知后断连触发重连 attach
			delete(b.subscribers, ch)
			close(ch)
		}
	}
}

// Subscribe 订阅实时事件并获取缓冲快照（锁内原子完成，回放与实时事件无重叠无丢失）
func (b *turnEventBus) Subscribe() (<-chan *protocol.EventMsg, []*protocol.EventMsg, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan *protocol.EventMsg, subscriberChanSize)
	b.subscribers[ch] = struct{}{}
	replay := make([]*protocol.EventMsg, len(b.buf))
	copy(replay, b.buf)
	return ch, replay, func() { b.unsubscribe(ch) }
}

func (b *turnEventBus) unsubscribe(ch chan *protocol.EventMsg) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.subscribers[ch]; ok {
		delete(b.subscribers, ch)
		close(ch)
	}
}

// Done 返回 turn 结束信号（收尾事件已全部发布）
func (b *turnEventBus) Done() <-chan struct{} {
	return b.done
}

// Finish 终结 bus：turn 收尾完成，不再接收发布
func (b *turnEventBus) Finish() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.doneClosed {
		return
	}
	b.doneClosed = true
	close(b.done)
}

// ==================== 会话级运行中 turn 注册表 ====================

// runningTurn 一个运行中 turn 的运行时句柄
type runningTurn struct {
	turnId string
	convId uint64
	userId string // 发起用户（运行中列表按用户过滤）
	bus    *turnEventBus
	ctx    context.Context    // turn ctx（Start 时由 WithCancel(baseCtx) 派生，runTurn 使用）
	cancel context.CancelFunc // 显式 stop 的取消入口（唯一真正中断 agent 的途径）

	itemsMu sync.Mutex
	items   []*entity.TurnItem // 主循环预收集的 TurnItem（如用户消息），供 runTurn 收尾合并持久化
}

// Subscribe 订阅该 turn 事件流（attached 事件将在回放后、实时事件前写入）
func (rt *runningTurn) Subscribe(attached *protocol.EventMsg) *turnSubscription {
	ch, replay, unsub := rt.bus.Subscribe()
	return &turnSubscription{
		replay:   replay,
		ch:       ch,
		done:     rt.bus.Done(),
		unsub:    unsub,
		attached: attached,
	}
}

// Stop 请求取消该 turn（ctx 取消传播到 LLM 请求与工具执行）
func (rt *runningTurn) Stop() {
	if rt.cancel != nil {
		rt.cancel()
	}
}

// Publish 发布事件到该 turn 的事件总线
func (rt *runningTurn) Publish(evt *protocol.EventMsg) {
	rt.bus.Publish(evt)
}

// CollectItems 主循环预收集的 TurnItem（如用户消息）
func (rt *runningTurn) CollectItems(items ...*entity.TurnItem) {
	rt.itemsMu.Lock()
	defer rt.itemsMu.Unlock()
	rt.items = append(rt.items, items...)
}

// SnapshotItems 取出预收集的 TurnItem（runTurn 收尾合并持久化时调用）
func (rt *runningTurn) SnapshotItems() []*entity.TurnItem {
	rt.itemsMu.Lock()
	defer rt.itemsMu.Unlock()
	items := rt.items
	rt.items = nil
	return items
}

// turnSubscription 单个 WS 连接对某 turn 的事件订阅（泵消费）
type turnSubscription struct {
	replay   []*protocol.EventMsg // 注册时点的缓冲快照
	ch       <-chan *protocol.EventMsg
	done     <-chan struct{}
	unsub    func()
	attached *protocol.EventMsg // 回放后写入的事件（turn_attached）
}

// chatRuntime 会话级运行中 turn 注册表（api 包级单例，单会话同时仅一个 turn）
type chatRuntime struct {
	mu    sync.Mutex
	turns map[uint64]*runningTurn
}

func newChatRuntime() *chatRuntime {
	return &chatRuntime{turns: make(map[uint64]*runningTurn)}
}

// Start 注册运行中 turn（会话级互斥：已有运行中 turn 时返回 errTurnRunning）。
// turn ctx 在此创建（WithCancel(baseCtx)），cancel 由本层持有，避免与 runTurn 产生数据竞争
func (r *chatRuntime) Start(convId uint64, turnId, userId string, baseCtx context.Context) (*runningTurn, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if old, ok := r.turns[convId]; ok {
		// done 已关闭（收尾中）则允许覆盖新 turn
		select {
		case <-old.bus.Done():
		default:
			return nil, errTurnRunning
		}
	}
	turnCtx, cancel := context.WithCancel(baseCtx)
	rt := &runningTurn{
		turnId: turnId,
		convId: convId,
		userId: userId,
		bus:    newTurnEventBus(),
		ctx:    turnCtx,
		cancel: cancel,
	}
	r.turns[convId] = rt
	return rt, nil
}

// GetRunning 查询该会话运行中的 turn（无则返回 nil）
func (r *chatRuntime) GetRunning(convId uint64) *runningTurn {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.turns[convId]
}

// Stop 取消该会话运行中的 turn，返回是否存在
func (r *chatRuntime) Stop(convId uint64) bool {
	r.mu.Lock()
	rt, ok := r.turns[convId]
	r.mu.Unlock()
	if !ok {
		return false
	}
	rt.Stop()
	return true
}

// Remove turn 收尾完成，移除注册
func (r *chatRuntime) Remove(convId uint64, turnId string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rt, ok := r.turns[convId]; ok && rt.turnId == turnId {
		delete(r.turns, convId)
	}
}

// ListRunning 列出指定用户运行中的 turn（会话列表执行中指示器数据源；userId 为空返回全部）
func (r *chatRuntime) ListRunning(userId string) []vo.RunningTurnVO {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]vo.RunningTurnVO, 0, len(r.turns))
	for _, rt := range r.turns {
		if userId != "" && rt.userId != userId {
			continue
		}
		result = append(result, vo.RunningTurnVO{
			ConversationId: rt.convId,
			TurnId:         rt.turnId,
		})
	}
	return result
}

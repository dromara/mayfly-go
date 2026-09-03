package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"mayfly-go/internal/ai/protocol"
	"mayfly-go/pkg/utils/jsonx"

	"github.com/gorilla/websocket"
)

func newTestEvent() *protocol.EventMsg {
	return &protocol.EventMsg{Type: "evt"}
}

// TestTurnEventBus_ReplaySubscribeNoLoss 订阅回放与实时事件无重叠无丢失
func TestTurnEventBus_ReplaySubscribeNoLoss(t *testing.T) {
	bus := newTurnEventBus()
	for i := 0; i < 3; i++ {
		bus.Publish(newTestEvent())
	}

	ch, replay, unsub := bus.Subscribe()
	defer unsub()
	if len(replay) != 3 {
		t.Fatalf("replay = %d, want 3", len(replay))
	}

	// 订阅后再发布的事件只进 channel，不重复出现在 replay
	bus.Publish(newTestEvent())
	bus.Publish(newTestEvent())
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("channel should deliver post-subscribe events")
	}
	// 快照不可变：后续 publish 不影响已获取的 replay
	if len(replay) != 3 {
		t.Fatalf("replay mutated: %d, want 3", len(replay))
	}
}

// TestTurnEventBus_ConcurrentPublishSubscribe 并发发布下订阅者收全全部事件（回放 + channel 合计）
func TestTurnEventBus_ConcurrentPublishSubscribe(t *testing.T) {
	bus := newTurnEventBus()
	const total = 500

	// 先发布一部分再订阅，覆盖「回放 + 实时」混合路径
	for i := 0; i < total/2; i++ {
		bus.Publish(newTestEvent())
	}
	ch, replay, unsub := bus.Subscribe()
	defer unsub()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < total/2; i++ {
			bus.Publish(newTestEvent())
		}
	}()
	wg.Wait()

	received := len(replay)
	timeout := time.After(2 * time.Second)
	for received < total {
		select {
		case _, ok := <-ch:
			if !ok {
				t.Fatal("subscriber channel closed unexpectedly")
			}
			received++
		case <-timeout:
			t.Fatalf("received %d events, want %d (lost %d)", received, total, total-received)
		}
	}
}

// TestTurnEventBus_SlowSubscriberRemoved 慢消费者缓冲溢出被移除，不阻塞发布方
func TestTurnEventBus_SlowSubscriberRemoved(t *testing.T) {
	bus := newTurnEventBus()
	ch, _, unsub := bus.Subscribe()

	// 超过订阅者缓冲容量后该订阅者被移除并关闭 channel（先排空缓冲内残留事件）
	for i := 0; i < subscriberChanSize+10; i++ {
		bus.Publish(newTestEvent())
	}
	received := 0
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				if received < subscriberChanSize {
					t.Fatalf("closed after only %d events", received)
				}
				unsub() // 幂等：已移除的订阅者重复取消不应 panic
				return
			}
			received++
		case <-time.After(time.Second):
			t.Fatal("slow subscriber channel not closed")
		}
	}
}

// TestTurnEventBus_FinishFinish 后停止发布且 done 关闭
func TestTurnEventBus_Finish(t *testing.T) {
	bus := newTurnEventBus()
	bus.Finish()
	bus.Publish(newTestEvent()) // 终结后发布被忽略，不 panic
	select {
	case <-bus.Done():
	default:
		t.Fatal("done should be closed after Finish")
	}
}

// TestChatRuntime_MutexAndStop 会话级互斥 / Stop 取消 / Remove 清理
func TestChatRuntime_MutexAndStop(t *testing.T) {
	r := newChatRuntime()

	turn, err := r.Start(1, "t1", "u1", context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if _, err := r.Start(1, "t2", "u1", context.Background()); err != errTurnRunning {
		t.Fatalf("second start should return errTurnRunning, got %v", err)
	}
	if r.GetRunning(1) != turn {
		t.Fatal("GetRunning should return the running turn")
	}

	// Stop 取消 turn ctx
	if !r.Stop(1) {
		t.Fatal("Stop should report the running turn exists")
	}
	if turn.ctx.Err() != context.Canceled {
		t.Fatalf("turn ctx should be canceled, got %v", turn.ctx.Err())
	}

	// 收尾移除后可重新 Start
	r.Remove(1, "t1")
	if r.GetRunning(1) != nil {
		t.Fatal("turn should be removed from registry")
	}
	if _, err := r.Start(1, "t2", "u1", context.Background()); err != nil {
		t.Fatalf("start after remove: %v", err)
	}
}

// TestChatRuntime_StartAfterFinishFinish 中的 turn 允许被新 turn 覆盖
func TestChatRuntime_StartAfterFinish(t *testing.T) {
	r := newChatRuntime()
	turn, err := r.Start(1, "t1", "u1", context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	turn.bus.Finish()
	if _, err := r.Start(1, "t2", "u1", context.Background()); err != nil {
		t.Fatalf("start after finish should succeed, got %v", err)
	}
}

// TestPumpTurnSubscription_ReplayRealtimeEnd 真实 WS 链路验证泵行为：
// 回放快照 → turn_attached → 实时事件 → Finish 后 drain 残余并写 end
func TestPumpTurnSubscription_ReplayRealtimeEnd(t *testing.T) {
	turn, err := newChatRuntime().Start(1, "t1", "u1", context.Background())
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	// 预置缓冲：回放内容
	turn.Publish(protocol.NewTurnStartedEvent("t1", 1))
	turn.Publish(&protocol.EventMsg{Type: protocol.EventTypeItemUpdated, TurnId: "t1"})

	upgrader := websocket.Upgrader{}
	var pumpStarted = make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		ws := &wsWriter{conn: conn}
		close(pumpStarted)
		sub := turn.Subscribe(protocol.NewTurnAttachedEvent("t1", 1))
		pumpTurnSubscription(ws, sub)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()
	<-pumpStarted

	readTypes := func(t *testing.T, n int) []string {
		t.Helper()
		types := make([]string, 0, n)
		for len(types) < n {
			client.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, msg, err := client.ReadMessage()
			if err != nil {
				t.Fatalf("read %d/%d: %v", len(types), n, err)
			}
			evt, err := jsonx.ToByStr[protocol.EventMsg](string(msg))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			types = append(types, evt.Type)
		}
		return types
	}

	// 回放（2 条）→ turn_attached
	types := readTypes(t, 3)
	if types[0] != protocol.EventTypeTurnStarted || types[1] != protocol.EventTypeItemUpdated || types[2] != protocol.EventTypeTurnAttached {
		t.Fatalf("unexpected sequence: %v", types)
	}

	// 实时事件续上
	turn.Publish(&protocol.EventMsg{Type: protocol.EventTypeItemUpdated, TurnId: "t1", ItemId: "realtime"})
	if got := readTypes(t, 1); got[0] != protocol.EventTypeItemUpdated {
		t.Fatalf("realtime event: %v", got)
	}

	// turn 收尾：turn_completed 先发布，Finish 后泵 drain 残余并写 end
	turn.Publish(protocol.NewTurnCompletedEvent("t1", "success", nil))
	turn.bus.Finish()
	types = readTypes(t, 2)
	if types[0] != protocol.EventTypeTurnCompleted || types[1] != protocol.EventTypeEnd {
		t.Fatalf("final sequence: %v", types)
	}
}

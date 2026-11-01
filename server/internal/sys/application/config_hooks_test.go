package application

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestConfigChangeHookNotify 配置变更钩子的三条契约：
// 精确 key 匹配、同 key 多钩子按注册顺序执行、钩子上下文不继承请求取消
// （保存接口返回后请求 ctx 会被取消，钩子不能因此被掐断）
func TestConfigChangeHookNotify(t *testing.T) {
	key := "test:hot-reload:" + t.Name()

	var mu sync.Mutex
	var got []string
	done := make(chan struct{}, 2)

	RegisterConfigChangeHook(key, func(ctx context.Context, k string) {
		mu.Lock()
		got = append(got, "first")
		mu.Unlock()
		if k != key {
			t.Errorf("hook key = %q, want %q", k, key)
		}
		if ctx.Err() != nil {
			t.Errorf("hook ctx should not inherit request cancellation: %v", ctx.Err())
		}
		done <- struct{}{}
	})
	RegisterConfigChangeHook(key, func(ctx context.Context, _ string) {
		mu.Lock()
		got = append(got, "second")
		mu.Unlock()
		done <- struct{}{}
	})
	RegisterConfigChangeHook(key+"-other", func(context.Context, string) {
		t.Error("其它 key 的钩子不应被触发")
	})

	// 模拟「请求已返回、ctx 已取消」的真实时相
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	notifyConfigChanged(ctx, key)

	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatalf("hook timeout, executed: %v", got)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Fatalf("hooks executed = %v, want [first second]（注册顺序）", got)
	}
}

// TestConfigChangeHookPanicIsolated 单个钩子 panic 不得影响同 key 的其它钩子
// （否则一个模块的刷新缺陷会让其它模块的热生效静默失效）
func TestConfigChangeHookPanicIsolated(t *testing.T) {
	key := "test:hot-reload:" + t.Name()

	done := make(chan struct{}, 1)
	RegisterConfigChangeHook(key, func(context.Context, string) {
		panic("boom")
	})
	RegisterConfigChangeHook(key, func(context.Context, string) {
		done <- struct{}{}
	})

	notifyConfigChanged(context.Background(), key)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("second hook should still run after first hook panicked")
	}
}

// TestConfigChangeHookNoSubscriber 无订阅者的 key 不产生任何副作用（保存其它配置零开销）
func TestConfigChangeHookNoSubscriber(t *testing.T) {
	notifyConfigChanged(context.Background(), "test:hot-reload:no-subscriber:"+t.Name())
}

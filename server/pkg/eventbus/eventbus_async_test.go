package eventbus

import (
	"context"
	"testing"
	"time"

	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/model"
)

// TestAsyncSubscriberReceivesLiveContext 异步订阅者拿到的 ctx 必须仍可继续写库。
//
// 回归的真实故障：异步订阅者沿用发布方传进来的请求 ctx，而它跑在独立协程里、通常晚于响应返回，
// 那时 net/http 已 cancel 了这个 ctx —— 订阅者里的落库在驱动层直接失败（实测资源操作留痕表
// INSERT 得到 rows:0 + "context canceled"），错误只进日志、数据永久丢失。
// 异步处理者的返回值只被写进日志，所以这里自己把 ctx 状态取回来断言
func TestAsyncSubscriberReceivesLiveContext(t *testing.T) {
	bus := New[any]()
	topic := "test:async-ctx"

	type observed struct {
		err      error
		traceId  string
		operator string
	}
	got := make(chan observed, 1)

	if err := bus.SubscribeAsync(topic, "probe", func(ctx context.Context, _ *Event[any]) error {
		la := contextx.GetLoginAccount(ctx)
		operator := ""
		if la != nil {
			operator = la.Username
		}
		got <- observed{err: ctx.Err(), traceId: contextx.GetTraceId(ctx), operator: operator}
		return nil
	}, false); err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	ctx = contextx.WithTraceId(ctx)
	ctx = contextx.WithLoginAccount(ctx, &model.LoginAccount{Id: 1, Username: "admin"})
	cancel()

	bus.Publish(ctx, topic, "payload")
	bus.WaitAsync()

	select {
	case o := <-got:
		if o.err != nil {
			t.Fatalf("订阅者拿到了已失效的 ctx，它的落库会被驱动层静默丢弃: %v", o.err)
		}
		// 剥掉的只能是取消信号：traceId 丢了无法与请求日志串联，登录账号丢了留痕的归属人会变成 '-'
		if o.traceId == "" {
			t.Fatal("traceId 必须随 ctx 保留，否则这批写入无法与请求日志串联")
		}
		if o.operator != "admin" {
			t.Fatalf("登录账号必须随 ctx 保留，否则留痕的归属人无从确定，实际: %q", o.operator)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("异步订阅者没有被执行")
	}
}

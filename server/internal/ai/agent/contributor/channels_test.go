package contributor

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// fakeMiddlewareContributor 中间件贡献者 fake
type fakeMiddlewareContributor struct {
	id   string
	err  error
	mws  []adk.ChatModelAgentMiddleware
	noop bool
}

func (f *fakeMiddlewareContributor) Id() string { return f.id }

func (f *fakeMiddlewareContributor) Middlewares(ctx context.Context) ([]adk.ChatModelAgentMiddleware, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.mws, nil
}

type fakeMiddleware struct {
	// 内嵌接口以获得完整方法集（测试中仅计数，不会真实调用）
	adk.ChatModelAgentMiddleware
	tag string
}

func (f *fakeMiddleware) WrapInvokableToolCall(ctx context.Context, endpoint adk.InvokableToolCallEndpoint, tc *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	return endpoint, nil
}

// fakeUsageContributor 用量贡献者 fake
type fakeUsageContributor struct {
	id    string
	err   error
	calls int
	last  *TokenUsageInput
}

func (f *fakeUsageContributor) Id() string { return f.id }

func (f *fakeUsageContributor) OnTokenUsage(ctx context.Context, in *TokenUsageInput) error {
	f.calls++
	f.last = in
	return f.err
}

// TestCollectMiddlewares 按注册顺序聚合；单个贡献者失败 fail-open 跳过
func TestCollectMiddlewares(t *testing.T) {
	b := NewBuilder()
	b.RegisterMiddleware(&fakeMiddlewareContributor{id: "mw-a", mws: []adk.ChatModelAgentMiddleware{&fakeMiddleware{tag: "a"}}})
	b.RegisterMiddleware(&fakeMiddlewareContributor{id: "mw-broken", err: errors.New("boom")})
	b.RegisterMiddleware(&fakeMiddlewareContributor{id: "mw-c", mws: []adk.ChatModelAgentMiddleware{&fakeMiddleware{tag: "c"}}})

	mws := b.Build().CollectMiddlewares(context.Background())
	if len(mws) != 2 {
		t.Fatalf("expected 2 middlewares (broken skipped), got %d", len(mws))
	}
}

// TestNotifyTokenUsage 通知全部用量贡献者；单个失败 fail-open
func TestNotifyTokenUsage(t *testing.T) {
	ok := &fakeUsageContributor{id: "usage-ok"}
	broken := &fakeUsageContributor{id: "usage-broken", err: errors.New("boom")}

	b := NewBuilder()
	b.RegisterTokenUsage(ok)
	b.RegisterTokenUsage(broken)

	in := &TokenUsageInput{
		SessionKey: "conv:1",
		TurnId:     "turn-1",
		UserId:     "1",
		Usage:      &schema.TokenUsage{TotalTokens: 100},
		Cause:      TurnEndCompleted,
	}
	b.Build().NotifyTokenUsage(context.Background(), in)

	if ok.calls != 1 || ok.last != in {
		t.Errorf("ok contributor should be notified once with input, calls=%d", ok.calls)
	}
	if broken.calls != 1 {
		t.Errorf("broken contributor should still be invoked (fail-open), calls=%d", broken.calls)
	}
}

// TestNotifyTokenUsage_NilRegistry nil registry 安全
func TestNotifyTokenUsage_NilRegistry(t *testing.T) {
	var r *Registry
	r.NotifyTokenUsage(context.Background(), &TokenUsageInput{})
	r.CollectMiddlewares(context.Background())
}

// TestWithFilter_NewChannels 新增通道同样支持按 Id 裁剪
func TestWithFilter_NewChannels(t *testing.T) {
	b := NewBuilder()
	b.RegisterMiddleware(&fakeMiddlewareContributor{id: "mw-a", mws: []adk.ChatModelAgentMiddleware{&fakeMiddleware{tag: "a"}}})
	b.RegisterTokenUsage(&fakeUsageContributor{id: "usage-a"})

	r := b.Build().WithFilter(map[string]struct{}{"usage-a": {}})
	if mws := r.CollectMiddlewares(context.Background()); len(mws) != 1 {
		t.Errorf("mw-a should be kept, got %d middlewares", len(mws))
	}
	if r.tokenUsageCount() != 0 {
		t.Errorf("usage-a should be filtered out")
	}
}

// tokenUsageCount 测试辅助：统计 token_usage 通道贡献者数量
func (r *Registry) tokenUsageCount() int {
	return r.ChannelCount(ChannelTokenUsage)
}

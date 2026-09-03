package contributor

import (
	"context"
	"errors"
	"testing"
)

// fakeActivatable 装配期激活型贡献者 fake
type fakeActivatable struct {
	id        string
	err       error
	activates int
}

func (f *fakeActivatable) Id() string { return f.id }

func (f *fakeActivatable) Activate(_ context.Context) error {
	f.activates++
	return f.err
}

// fakePlain 非激活型贡献者 fake
type fakePlain struct{ id string }

func (f *fakePlain) Id() string { return f.id }

// TestRegistry_Activate 仅激活实现 Activatable 的贡献者，按注册顺序执行
func TestRegistry_Activate(t *testing.T) {
	plain := &fakePlain{id: "plain"}
	a1 := &fakeActivatable{id: "svc-a"}
	a2 := &fakeActivatable{id: "svc-b"}

	b := NewBuilder()
	b.RegisterContext(&fakeContext{id: "ctx-a"}) // 非 service 通道同样参与激活探测
	b.RegisterService(plain)
	b.RegisterService(a1)
	b.RegisterService(a2)

	if err := b.Build().Activate(context.Background()); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if a1.activates != 1 || a2.activates != 1 {
		t.Errorf("activatables should be activated once, a=%d b=%d", a1.activates, a2.activates)
	}
}

// TestRegistry_Activate_FailOpen 单个激活失败不阻断其余贡献者，且返回首个错误
func TestRegistry_Activate_FailOpen(t *testing.T) {
	broken := &fakeActivatable{id: "broken", err: errors.New("boom")}
	ok := &fakeActivatable{id: "ok"}

	b := NewBuilder()
	b.RegisterService(broken)
	b.RegisterService(ok)

	err := b.Build().Activate(context.Background())
	if err == nil {
		t.Fatal("expected first activation error to be returned")
	}
	if broken.activates != 1 || ok.activates != 1 {
		t.Errorf("fail-open violated: broken=%d ok=%d", broken.activates, ok.activates)
	}
}

// TestRegistry_Activate_WithFilter 被裁剪的扩展不会激活（裁剪先于激活）
func TestRegistry_Activate_WithFilter(t *testing.T) {
	kept := &fakeActivatable{id: "kept"}
	dropped := &fakeActivatable{id: "dropped"}

	b := NewBuilder()
	b.RegisterService(kept)
	b.RegisterService(dropped)

	r := b.Build().WithFilter(map[string]struct{}{"dropped": {}})
	if err := r.Activate(context.Background()); err != nil {
		t.Fatalf("activate: %v", err)
	}
	if dropped.activates != 0 {
		t.Errorf("filtered contributor should not be activated, activates=%d", dropped.activates)
	}
	if kept.activates != 1 {
		t.Errorf("kept contributor should be activated, activates=%d", kept.activates)
	}
}

// TestRegistry_Activate_NilRegistry nil registry 安全
func TestRegistry_Activate_NilRegistry(t *testing.T) {
	var r *Registry
	if err := r.Activate(context.Background()); err != nil {
		t.Errorf("nil registry activate should be noop, got %v", err)
	}
}

// TestRegistry_Summary 各通道装配摘要
func TestRegistry_Summary(t *testing.T) {
	b := NewBuilder()
	b.RegisterContext(&fakeContext{id: "ctx-a"})
	b.RegisterContext(&fakeContext{id: "ctx-b"})
	b.RegisterTool(&fakeToolProvider{id: "tool-a"})

	summary := b.Build().Summary()
	if summary != "context=2, tool=1" {
		t.Errorf("unexpected summary: %s", summary)
	}

	var r *Registry
	if r.Summary() != "<nil>" {
		t.Errorf("nil registry summary: %s", r.Summary())
	}
}

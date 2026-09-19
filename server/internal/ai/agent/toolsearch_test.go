package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// fakeTsTool 仅实现 Info（toolsearch 中间件构建期仅消费工具元信息）
type fakeTsTool struct {
	name string
}

func (f *fakeTsTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: f.name}, nil
}

func fakeTools(normal, deferred int) []tool.BaseTool {
	tools := make([]tool.BaseTool, 0, normal+deferred)
	for i := 0; i < normal; i++ {
		tools = append(tools, &fakeTsTool{name: fmt.Sprintf("builtin_tool_%d", i)})
	}
	for i := 0; i < deferred; i++ {
		tools = append(tools, &fakeTsTool{name: fmt.Sprintf("deferred_tool_%d", i)})
	}
	return tools
}

// fakeDeferredNames 构建与 fakeTools 中 deferred 部分对应的名称集合
func fakeDeferredNames(deferred int) map[string]struct{} {
	if deferred == 0 {
		return nil
	}
	names := make(map[string]struct{}, deferred)
	for i := 0; i < deferred; i++ {
		names[fmt.Sprintf("deferred_tool_%d", i)] = struct{}{}
	}
	return names
}

func TestBuildToolSearchMiddleware(t *testing.T) {
	ctx := context.Background()

	// 未配置阈值（0）：禁用，全量直注（原工具清单原样返回）
	tools := fakeTools(10, 5)
	mw, out := buildToolSearchMiddleware(ctx, tools, fakeDeferredNames(5), 0)
	if mw != nil {
		t.Fatal("threshold 0 should disable tool search")
	}
	if len(out) != len(tools) {
		t.Fatalf("disabled should return original tool list, got %d want %d", len(out), len(tools))
	}

	// 未超阈值：禁用
	if mw, _ = buildToolSearchMiddleware(ctx, fakeTools(3, 2), fakeDeferredNames(2), 5); mw != nil {
		t.Fatal("tools within threshold should disable tool search")
	}
	// 超阈值但无 deferred 工具（nil deferredNames）：禁用
	if mw, _ = buildToolSearchMiddleware(ctx, fakeTools(6, 0), nil, 5); mw != nil {
		t.Fatal("no deferred tools should disable tool search")
	}
	// 超阈值但 deferredNames 为空 map：禁用
	if mw, _ = buildToolSearchMiddleware(ctx, fakeTools(6, 0), map[string]struct{}{}, 5); mw != nil {
		t.Fatal("empty deferred names should disable tool search")
	}

	// 超阈值且存在 deferred 工具：启用，且返回静态清单（排除 deferred 工具）
	mw, out = buildToolSearchMiddleware(ctx, fakeTools(6, 3), fakeDeferredNames(3), 5)
	if mw == nil {
		t.Fatal("expected tool search middleware when tools exceed threshold with deferred tools")
	}
	if len(out) != 6 {
		t.Fatalf("static tool list should exclude deferred tools, got %d want 6", len(out))
	}
	for _, t2 := range out {
		info, err := t2.Info(ctx)
		if err != nil {
			t.Fatalf("unexpected info error: %v", err)
		}
		if _, ok := fakeDeferredNames(3)[info.Name]; ok {
			t.Fatalf("static tool list must not contain deferred tool %q (ToolsConfig 传全量会导致 runCtx.Tools 重名重复注册)", info.Name)
		}
	}
}

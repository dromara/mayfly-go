package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	mcpext "mayfly-go/internal/ai/agent/ext/mcp/mcpext"
)

// fakeTsTool 仅实现 Info（toolsearch 中间件构建期仅消费工具元信息）
type fakeTsTool struct {
	name string
}

func (f *fakeTsTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: f.name}, nil
}

func fakeTools(normal, mcp int) []tool.BaseTool {
	tools := make([]tool.BaseTool, 0, normal+mcp)
	for i := 0; i < normal; i++ {
		tools = append(tools, &fakeTsTool{name: fmt.Sprintf("builtin_tool_%d", i)})
	}
	for i := 0; i < mcp; i++ {
		tools = append(tools, &fakeTsTool{name: fmt.Sprintf("mcp_server1_tool_%d", i)})
	}
	return tools
}

func TestBuildToolSearchMiddleware(t *testing.T) {
	ctx := context.Background()

	// 未配置阈值（0）：禁用，全量直注（原工具清单原样返回）
	tools := fakeTools(10, 5)
	mw, out := buildToolSearchMiddleware(ctx, tools, 0)
	if mw != nil {
		t.Fatal("threshold 0 should disable tool search")
	}
	if len(out) != len(tools) {
		t.Fatalf("disabled should return original tool list, got %d want %d", len(out), len(tools))
	}

	// 未超阈值：禁用
	if mw, _ = buildToolSearchMiddleware(ctx, fakeTools(3, 2), 5); mw != nil {
		t.Fatal("tools within threshold should disable tool search")
	}
	// 超阈值但无 MCP 工具：禁用（无可延迟工具，避免无谓的 tool_search 元工具）
	if mw, _ = buildToolSearchMiddleware(ctx, fakeTools(6, 0), 5); mw != nil {
		t.Fatal("no mcp tools should disable tool search")
	}

	// 超阈值且存在 MCP 工具：启用，且返回静态清单（排除 mcp_ 前缀工具）
	mw, out = buildToolSearchMiddleware(ctx, fakeTools(6, 3), 5)
	if mw == nil {
		t.Fatal("expected tool search middleware when tools exceed threshold with mcp tools")
	}
	if len(out) != 6 {
		t.Fatalf("static tool list should exclude deferred mcp tools, got %d want 6", len(out))
	}
	for _, t2 := range out {
		info, err := t2.Info(ctx)
		if err != nil {
			t.Fatalf("unexpected info error: %v", err)
		}
		if strings.HasPrefix(info.Name, mcpext.ToolNamePrefix) {
			t.Fatalf("static tool list must not contain deferred tool %q (ToolsConfig 传全量会导致 runCtx.Tools 重名重复注册)", info.Name)
		}
	}
}

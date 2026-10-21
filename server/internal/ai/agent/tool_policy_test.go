package agent

import (
	"context"
	"errors"
	"testing"

	"mayfly-go/internal/ai/tools"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/require"
)

// fakeTool 最小 tool.BaseTool 实现（仅 Info，供能力过滤测试）
type fakeTool struct {
	name string
	err  error // 非 nil 时 Info 返回错误（模拟元信息异常工具）
}

var _ tool.BaseTool = (*fakeTool)(nil)

func (f *fakeTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &schema.ToolInfo{Name: f.name}, nil
}

func toolNames(ts []tool.BaseTool) []string {
	names := make([]string, 0, len(ts))
	for _, t := range ts {
		info, _ := t.Info(context.Background())
		names = append(names, info.Name)
	}
	return names
}

func TestFilterToolsByPolicy_UnrestrictedReturnsAll(t *testing.T) {
	src := []tool.BaseTool{&fakeTool{name: "DbQueryData"}, &fakeTool{name: "ExecSql"}}
	deferred := map[string]struct{}{"ExecSql": {}}

	// nil 策略：原样返回
	got, gotDeferred := filterToolsByPolicy(context.Background(), src, deferred, nil)
	require.Len(t, got, 2)
	require.Equal(t, deferred, gotDeferred)

	// 空允许集策略：同样不限制
	got2, _ := filterToolsByPolicy(context.Background(), src, deferred, tools.NewToolPolicy())
	require.Len(t, got2, 2)
}

func TestFilterToolsByPolicy_ReadOnlyDropsDangerous(t *testing.T) {
	src := []tool.BaseTool{
		&fakeTool{name: "DbQueryData"},   // read → 保留
		&fakeTool{name: "ListResources"}, // read → 保留
		&fakeTool{name: "ExecSql"},       // dangerous → 裁剪
	}
	deferred := map[string]struct{}{"ExecSql": {}, "DbQueryData": {}}

	got, gotDeferred := filterToolsByPolicy(context.Background(), src, deferred, tools.ReadOnlyPolicy())
	require.ElementsMatch(t, []string{"DbQueryData", "ListResources"}, toolNames(got))
	// 被裁剪工具同步从 deferred 集合移除
	_, stillDeferred := gotDeferred["ExecSql"]
	require.False(t, stillDeferred, "被裁剪的高危工具应从 deferred 集合移除")
	_, keptDeferred := gotDeferred["DbQueryData"]
	require.True(t, keptDeferred, "放行工具的 deferred 声明应保留")
}

func TestFilterToolsByPolicy_InfoErrorDroppedConservatively(t *testing.T) {
	src := []tool.BaseTool{
		&fakeTool{name: "DbQueryData"},
		&fakeTool{name: "broken", err: errors.New("info boom")}, // 元信息异常 → 保守排除
	}
	got, _ := filterToolsByPolicy(context.Background(), src, nil, tools.ReadOnlyPolicy())
	require.Equal(t, []string{"DbQueryData"}, toolNames(got))
}

// TestFilterToolsByPolicy_KeepsContentCapabilityTool 装配期只读档必须保留「内容级工具」，
// 否则模型工具清单里没有它、连只读命令都不会去尝试（Run 91 实测：诊断只会「建议你手动跑 top」）。
// 同时仍裁剪无只读子集的纯高危工具——不能因为开了内容级接缝就放开全部高危工具。
func TestFilterToolsByPolicy_KeepsContentCapabilityTool(t *testing.T) {
	const probe = "probe_filter_content_tool"
	tools.DeclareContentCapability(probe, func(string) tools.Capability { return tools.CapabilityRead })
	src := []tool.BaseTool{
		&fakeTool{name: "DbQueryData"}, // read → 保留
		&fakeTool{name: "ExecSql"},     // 纯高危无内容级 → 裁剪
		&fakeTool{name: probe},         // 内容级 → 保留（运行期按 args 精判）
	}
	got, _ := filterToolsByPolicy(context.Background(), src, nil, tools.ReadOnlyPolicy())
	require.ElementsMatch(t, []string{"DbQueryData", probe}, toolNames(got))
}

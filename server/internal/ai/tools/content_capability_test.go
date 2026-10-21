package tools

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestReadOnlyPolicyKeepsContentCapabilityTool 只读策略在装配期必须保留「内容级工具」，
// 否则只读诊断档的模型工具清单里根本没有它，连只读命令都不会去尝试（实测：诊断只会「建议你手动跑 top」）。
// 真正的拦/放行动作发生在运行期 capability_guard（按实际 args），装配期只负责「让模型看得到这个工具」。
func TestReadOnlyPolicyKeepsContentCapabilityTool(t *testing.T) {
	const probe = "probe_content_cap_tool"
	DeclareContentCapability(probe, func(string) Capability { return CapabilityRead })
	t.Cleanup(func() {
		contentCapabilityMu.Lock()
		delete(contentCapabilityJudges, probe)
		contentCapabilityMu.Unlock()
	})

	// 探针静态未声明（默认 Write），但注册了内容级判定器 → 只读策略须放行它进清单
	require.True(t, ReadOnlyPolicy().Allows(probe), "内容级工具装配期须保留，交由运行期按入参判定")

	// 对比：静态高危且无内容级判定器的工具，只读档仍须移除（不能因为开了内容级接缝就放开全部高危工具）
	require.False(t, ReadOnlyPolicy().Allows(ToolNameExecSql), "无只读子集的高危工具仍须被只读策略裁剪")
}

package tools

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCapabilityOf_BuiltinDeclarations(t *testing.T) {
	// 内置只读工具声明为 read
	require.Equal(t, CapabilityRead, CapabilityOf("DbQueryData"))
	require.Equal(t, CapabilityRead, CapabilityOf("ListResources"))
	require.Equal(t, CapabilityRead, CapabilityOf("memory_search"))
	// 内置高危工具声明为 dangerous
	require.Equal(t, CapabilityDangerous, CapabilityOf("ExecSql"))
	require.Equal(t, CapabilityDangerous, CapabilityOf("MachineCommandExec"))
}

func TestCapabilityOf_UndeclaredDefaultsToWrite(t *testing.T) {
	// 未声明工具保守按 write 处理（只读档不放行未知工具，防 MCP 等绕过）
	require.Equal(t, CapabilityWrite, CapabilityOf("some_unknown_mcp_tool"))
}

func TestDeclareCapability(t *testing.T) {
	const name = "__test_capability_tool__"
	// 能力表是**包级全局注册表**：注入的探针必须用完即删。否则同包后续任何「遍历能力表」的
	// 守卫会被这条脏数据顶掉——实测污染过一次：本用例把它登记成 Dangerous 后，
	// 「每个高危工具都必须声明入参形状」的守卫如实报它缺声明（看起来像守卫误报，其实是脏源）
	t.Cleanup(func() {
		capabilityMu.Lock()
		delete(capabilities, name)
		capabilityMu.Unlock()
	})

	// 初始未声明 → write
	require.Equal(t, CapabilityWrite, CapabilityOf(name))
	// 声明为 read 后生效
	DeclareCapability(name, CapabilityRead)
	require.Equal(t, CapabilityRead, CapabilityOf(name))
	// 覆盖声明
	DeclareCapability(name, CapabilityDangerous)
	require.Equal(t, CapabilityDangerous, CapabilityOf(name))
	// 空工具名忽略（不 panic）
	DeclareCapability("", CapabilityRead)
}

func TestCapability_String(t *testing.T) {
	require.Equal(t, "read", CapabilityRead.String())
	require.Equal(t, "write", CapabilityWrite.String())
	require.Equal(t, "dangerous", CapabilityDangerous.String())
	require.Equal(t, "unknown", Capability(99).String())
}

func TestToolPolicy_NilUnrestricted(t *testing.T) {
	var p *ToolPolicy
	require.False(t, p.IsRestricted())
	// nil 策略恒放行（含高危工具）
	require.True(t, p.Allows("ExecSql"))
	require.True(t, p.Allows("anything"))
}

func TestToolPolicy_EmptyUnrestricted(t *testing.T) {
	p := NewToolPolicy() // 空允许集
	require.False(t, p.IsRestricted())
	require.True(t, p.Allows("ExecSql"))
}

func TestToolPolicy_ReadOnly(t *testing.T) {
	p := ReadOnlyPolicy()
	require.True(t, p.IsRestricted())
	// 只读工具放行
	require.True(t, p.Allows("DbQueryData"))
	require.True(t, p.Allows("ListResources"))
	// 高危/写/未知工具拒绝
	require.False(t, p.Allows("ExecSql"))
	require.False(t, p.Allows("some_unknown_tool"))
	// MachineCommandExec 是「内容级工具」（machinetool init 注册了判定器，param_schema_guard_test 的
	// blank import 确保其在本测试二进制内生效）：只读档装配期须保留它进清单、运行期再按 args 精判，
	// 故 Allows=true。先显式断言判定器已注册，避免哪天 blank import 被删时 Allows 静默翻回 false 却测不出来
	require.True(t, HasContentCapability("MachineCommandExec"), "machinetool 应为 MachineCommandExec 注册内容级判定器")
	require.True(t, p.Allows("MachineCommandExec"), "内容级工具在只读档装配期须保留")
}

func TestToolPolicy_CustomAllowSet(t *testing.T) {
	p := NewToolPolicy(CapabilityRead, CapabilityDangerous)
	require.True(t, p.IsRestricted())
	require.True(t, p.Allows("DbQueryData"))   // read
	require.True(t, p.Allows("ExecSql"))       // dangerous
	require.False(t, p.Allows("unknown_tool")) // 默认 write，不在允许集
}

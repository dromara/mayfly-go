package tools

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestApprovedForBindsParams 「批准过这个工具」不等于「这个工具的任意参数都免审批」。
//
// 这是放行机制最容易被做坏的一处：若只按工具名放行，模型在受控执行那一轮里换一组参数
// （批准的是 command:"true"，跑成 rm -rf /），审批就等于被绕开——而这恰恰是受控执行要防的场景。
// 所以 ApprovedTools 的 value 存的是批准时的参数 JSON，放行前逐项比对。
func TestApprovedForBindsParams(t *testing.T) {
	grant := &RunGrant{
		AllowedToolNames: []string{"MachineCommandExec"},
		ApprovedTools:    map[string]string{"MachineCommandExec": `{"command":"true","authCertName":"tVUcRFKwOC"}`},
	}

	match := map[string]any{"authCertName": "tVUcRFKwOC", "command": "true"} // 键序与批准时不同
	tampered := map[string]any{"authCertName": "tVUcRFKwOC", "command": "rm -rf /"}
	extraKey := map[string]any{"authCertName": "tVUcRFKwOC", "command": "true", "remark": "顺手多带一个"}

	require.True(t, grant.ApprovedFor("MachineCommandExec", match), "同一组参数应放行（与 JSON 键序无关）")
	require.False(t, grant.ApprovedFor("MachineCommandExec", tampered), "参数取值被换必须不放行")
	require.False(t, grant.ApprovedFor("MachineCommandExec", extraKey), "多带参数也不放行（项数不等即视为不同调用）")
	require.False(t, grant.ApprovedFor("ExecSql", match), "未批准过的工具不放行")
	require.False(t, grant.ApprovedFor("MachineCommandExec", map[string]any{}), "实际参数为空不放行")
}

// TestApprovedForValueRepresentation 数字与字符串表示要视为同一取值（22 与 "22"）。
//
// 工具参数经 JSON 往返后数字可能变 float64、也可能仍是字符串：若严格区分类型，正常调用会被
// 误判成「参数被篡改」而永远执行不了；比对用的是展平后的 %v 文本，两边同规则故不会漏放真篡改
func TestApprovedForValueRepresentation(t *testing.T) {
	grant := &RunGrant{ApprovedTools: map[string]string{"MachineCommandExec": `{"port":22,"command":"true"}`}}

	require.True(t, grant.ApprovedFor("MachineCommandExec", map[string]any{"port": "22", "command": "true"}))
	require.True(t, grant.ApprovedFor("MachineCommandExec", map[string]any{"port": 22, "command": "true"}))
	require.False(t, grant.ApprovedFor("MachineCommandExec", map[string]any{"port": "2222", "command": "true"}))
}

// TestApprovedForFailClosed 三种「拿不到可靠凭据」的形态一律不放行。
func TestApprovedForFailClosed(t *testing.T) {
	match := map[string]any{"command": "true"}

	var nilGrant *RunGrant
	require.False(t, nilGrant.ApprovedFor("MachineCommandExec", match), "nil 许可（普通聊天/未审批运行）不放行")

	empty := &RunGrant{AllowedToolNames: []string{"MachineCommandExec"}}
	require.False(t, empty.ApprovedFor("MachineCommandExec", match), "只有工具清单、没有批准参数时不放行（否则等于按名字放行任意参数）")

	dirty := &RunGrant{ApprovedTools: map[string]string{"MachineCommandExec": "not-json"}}
	require.False(t, dirty.ApprovedFor("MachineCommandExec", match), "批准参数是脏数据时不放行，也不能 panic")

	blank := &RunGrant{ApprovedTools: map[string]string{"MachineCommandExec": ""}}
	require.False(t, blank.ApprovedFor("MachineCommandExec", match), "批准参数为空串不放行")
}

// 批准参数随运行自持（Run.Extra 往返）的契约在 agentops/domain/entity 包测试里（
// SetApprovedActions 属于那个聚合；tools 不依赖业务层，不能反向 import）

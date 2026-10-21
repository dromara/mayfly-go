package tools_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"mayfly-go/internal/ai/tools"
	// 内置工具的入参形状由各自包的 init 声明，这里只负责触发装载（与真实进程一致）
	_ "mayfly-go/internal/ai/tools/dbtool"
	_ "mayfly-go/internal/ai/tools/machinetool"
)

// TestDangerousToolsDeclareParamSchemas 每个高危工具都必须在包加载期声明入参形状且有必填属性。
//
// 为什么必须有这条机制而不是靠约定：ValidateToolParams 的「未声明形状则跳过」对 MCP/插件是
// 正确设计（不猜外部工具的形状），套在内置高危工具上就是静默失效通道——新增工具忘了写那行
// init，登记校验直接放行任意坏参数，编译、静态检查、既有单测全绿，只有真跑到批准执行那一步
// 才表现为「批了却什么都执行不了」（本轮已实测过同形态的时机缺陷）
//
// 必填属性非空也是同一类要求：required 项是审批人判断「这条变更到底动了什么」的唯一依据，
// 一个没有任何必填属性的高危工具，等于允许登记一组可以全为空的执行参数
func TestDangerousToolsDeclareParamSchemas(t *testing.T) {
	names := tools.DangerousToolNames()
	require.NotEmpty(t, names, "能力表应至少登记一个高危工具，否则本守卫退化为空跑")

	for _, name := range names {
		schema, ok := tools.ParamSchemaOf(name)
		require.Truef(t, ok, "高危工具 %s 必须在包初始化期 DeclareParamSchema（否则参数校验对它整体跳过）", name)
		require.NotEmptyf(t, schema.Props, "%s 的入参属性集不能为空", name)
		require.NotEmptyf(t, schema.Required,
			"%s 必须声明至少一个必填属性：那是审批人判断变更内容的依据", name)
	}
}

// TestDangerousToolNamesAreStableSnapshot 高危工具清单变化必须是有意识的决定。
//
// 上面的守卫只遍历表，若有人把新工具误登记成 Write（从而绕开形状要求）它抓不到；
// 这份快照让「新增/降级一个高危工具」必须显式改测试，而不是静默少一层防护
func TestDangerousToolNamesAreStableSnapshot(t *testing.T) {
	require.ElementsMatch(t, []string{"ExecSql", "MachineCommandExec"}, tools.DangerousToolNames())
}

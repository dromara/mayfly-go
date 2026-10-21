package application

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunEntityHasSingleConstructionPoint 运行实体只能有一个构造点，两条创建路径都必须走它。
//
// 缺陷类（真实发生过）：Enqueue 与 RunSync 各自拼一份 18 字段字面量，`RunRole` 只补了 Enqueue，
// 于是同步跑出来的运行——提案的受控执行/校验正好走这条路——永远没有形态标签，而编译、类型、
// 单测全绿。两处手抄的字段清单没有任何机制保证同步，补字段漏一处也不会报错。
//
// 判据是结构性的：文件里 `&entity.Run{` 只允许出现在 buildRunEntity 内；两个入口必须各有一处
// `buildRunEntity(req, ...)` 调用，且状态常量对齐各自语义（queued / running）
func TestRunEntityHasSingleConstructionPoint(t *testing.T) {
	source, err := os.ReadFile("run_service.go")
	require.NoError(t, err)
	text := stripLineCommentsForGuard(string(source))

	require.Equal(t, 1, strings.Count(text, "&entity.Run{"),
		"运行实体出现了第二个构造点：字段清单会再次分叉，新增字段只会落在一条路径上")

	require.Contains(t, text, "buildRunEntity(req, entity.RunStatusQueued)",
		"Enqueue 必须用共享构造器构造 queued 运行")
	require.Contains(t, text, "buildRunEntity(req, entity.RunStatusRunning)",
		"RunSync 必须用共享构造器构造 running 运行")
}

// stripLineCommentsForGuard 去掉行注释（守卫不应命中注释里引用的字面量）。
//
// 前提：不识别字符串字面量内的 `//`；被扫文件若出现此类字面量须改判据，而不是放宽断言
func stripLineCommentsForGuard(source string) string {
	var kept []string
	for _, line := range strings.Split(source, "\n") {
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

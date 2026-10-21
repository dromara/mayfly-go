package application

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPluginChangeResetsProfileAgents 插件/技能/MCP 变更后必须连档位 Agent 缓存一起重置。
//
// 缺陷类（真实发生过）：默认 Agent 与 profile Agent 是两份独立缓存，且工具清单在 NewAgent 时
// 一次性快照。`invalidateDefaultAgent` 曾只调 `agent.ResetDefaultAgent()`，而 `ResetAllProfiles`
// 全仓零生产调用者——于是新装/卸载的插件与 MCP 工具只在普通对话生效，headless 档
// （提案执行、评测）永远用旧清单，直到进程重启，且编译/日志/门禁全绿。
//
// 判据是结构性的：该函数体内必须出现 ResetAllProfiles()
func TestPluginChangeResetsProfileAgents(t *testing.T) {
	raw, err := os.ReadFile("plugin_instance.go")
	require.NoError(t, err)
	text := stripLineCommentsForInvalidateGuard(string(raw))

	body := regexp.MustCompile(`(?s)func invalidateDefaultAgent\(\) \{(.*?)\n\}`).FindStringSubmatch(text)
	require.Len(t, body, 2, "找不到 invalidateDefaultAgent 函数体：函数被改名/移走时本守卫须同步更新")

	require.Contains(t, body[1], "agent.ResetDefaultAgent()", "默认 Agent 缓存仍须重置")
	require.Contains(t, body[1], "ResetAllProfiles()",
		"必须同时重置档位 Agent 缓存，否则插件/MCP 工具变更在 headless 档永远不生效")
}

// stripLineCommentsForInvalidateGuard 去掉行注释（说明注释里会引用函数名与调用形态）
func stripLineCommentsForInvalidateGuard(source string) string {
	var kept []string
	for _, line := range strings.Split(source, "\n") {
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

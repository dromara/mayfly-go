package api

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestChatWSPerTurnAgentResolution 守卫：聊天 WebSocket 必须在每轮发送时重新解析 Agent。
//
// 缺陷类（真机实测）：Agent 在构造时绑定 ChatModel，而模型配置（地址 / Key / 模型名）
// 支持保存后热生效。若连接建立时取一次 Agent 并在整个连接生命周期复用，模型配置
// 改了之后「已打开页面的下一轮对话」仍会连旧端点 —— 表现为改完 Key 依旧 401/超时，
// 直到刷新页面重连，而此时后端日志里热生效链路一切正常（极难定位）。
func TestChatWSPerTurnAgentResolution(t *testing.T) {
	raw, err := os.ReadFile("chat_ws.go")
	if err != nil {
		t.Fatalf("read chat_ws.go: %v", err)
	}
	text := stripGoLineComments(string(raw))

	// runTurn 收到的必须是「按轮解析」的实例，而不是连接级快照
	if !regexp.MustCompile(`go a\.runTurn\(turn,\s*turnAgent\b`).MatchString(text) {
		t.Fatal("runTurn 必须以按轮解析的 turnAgent 调用，否则已打开的会话永远用旧模型")
	}
	if strings.Contains(text, "go a.runTurn(turn, ag,") {
		t.Fatal("不得再用连接级 ag 快照执行轮次（模型配置热生效会对已打开页面失效）")
	}
	// 校验位：发送分支内确实重新取过 Agent
	if strings.Count(text, "agent.GetDefaultAgent(") < 2 {
		t.Fatal("发送分支必须重新调用 agent.GetDefaultAgent 取当前实例")
	}
}

// stripGoLineComments 去掉行注释（说明注释里会引用这些调用形态）
func stripGoLineComments(source string) string {
	var kept []string
	for _, line := range strings.Split(source, "\n") {
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

package application

import (
	"os"
	"strings"
	"testing"

	"mayfly-go/internal/ai/agent"

	"github.com/stretchr/testify/require"
)

// TestReloadModelConfigInvalidatesProfileAgents 模型配置热生效必须同时失效档位（profile）缓存。
//
// 缺陷类：默认 Agent 与档位 Agent 是两份独立缓存，且 Agent 在构造时绑定 ChatModel。
// 只失效默认 Agent 会让「聊天已用新模型、提案执行/评测仍在连旧地址」，而编译、
// 日志、界面全部正常。档位缓存失效后，下一次取档位 Agent 会按当前模型重建。
func TestReloadModelConfigInvalidatesProfileAgents(t *testing.T) {
	seedProfiles(t, map[string]*agent.Agent{"diag": {}, "exec5": {}})

	ReloadModelConfig()

	require.Empty(t, snapshotProfiles(), "模型配置变更后档位 Agent 缓存必须清空")
}

// TestReloadAgentConfigInvalidatesProfileAgents Agent 配置（工具搜索阈值）同样按轮生效
func TestReloadAgentConfigInvalidatesProfileAgents(t *testing.T) {
	seedProfiles(t, map[string]*agent.Agent{"diag": {}})

	ReloadAgentConfig()

	require.Empty(t, snapshotProfiles(), "Agent 配置变更后档位 Agent 缓存必须清空")
}

// TestSessionStoreMetaCarriesUserId 会话元信息必须带归属人。
//
// 缺陷类（真机实测）：摘要完成事件里的 UserId 取自 SessionMeta，而 DB 版会话存储
// 从未设置它 → 事件 UserId 恒为空 → 订阅方 AgentMemoryExtractor 静默 return，
// 长期记忆提取在生产链路上永不触发（无日志、无报错、界面无感）。
func TestSessionStoreMetaCarriesUserId(t *testing.T) {
	raw, err := os.ReadFile("session_store.go")
	require.NoError(t, err)
	text := strings.Join(strings.Split(string(raw), "\n"), "\n")

	body := regexpFindFuncBody(t, text, "func (s *sessionStoreImpl) GetMeta(")
	require.Contains(t, body, "UserId:", "GetMeta 必须填充 SessionMeta.UserId，否则长期记忆提取永不触发")
	require.Contains(t, body, "conv.CreatorId", "归属人取会话创建人（会话表没有独立 user 列）")
}

// seedProfiles 覆盖档位 Agent 缓存，返回后自动还原
func seedProfiles(t *testing.T, items map[string]*agent.Agent) {
	t.Helper()
	profileMu.Lock()
	origin := profileAgents
	profileAgents = items
	profileMu.Unlock()
	t.Cleanup(func() {
		profileMu.Lock()
		profileAgents = origin
		profileMu.Unlock()
	})
}

func snapshotProfiles() map[string]*agent.Agent {
	profileMu.RLock()
	defer profileMu.RUnlock()
	out := make(map[string]*agent.Agent, len(profileAgents))
	for k, v := range profileAgents {
		out[k] = v
	}
	return out
}

// regexpFindFuncBody 截取目标函数体（到首个行首右花括号为止）
func regexpFindFuncBody(t *testing.T, source, signature string) string {
	t.Helper()
	start := strings.Index(source, signature)
	require.GreaterOrEqual(t, start, 0, "找不到函数声明：%s", signature)
	rest := source[start:]
	end := strings.Index(rest, "\n}")
	require.Greater(t, end, 0, "找不到函数体结束位置：%s", signature)
	return rest[:end]
}

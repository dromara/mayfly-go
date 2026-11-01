package agent

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"

	"mayfly-go/internal/ai/llm/llmtest"

	"github.com/stretchr/testify/require"
)

// stubModelRuntimeBuilder 替换模型运行时构造器（不依赖数据库与真实模型），
// 返回的构造结果每次自增，便于断言「重建了几次」
func stubModelRuntimeBuilder(builds *int32, err error) func(context.Context) (*modelRuntime, error) {
	return func(context.Context) (*modelRuntime, error) {
		if err != nil {
			return nil, err
		}
		n := atomic.AddInt32(builds, 1)
		return &modelRuntime{
			chatModel:     &llmtest.Model{Reply: "v" + strconv.Itoa(int(n))},
			contextWindow: int(n) * 1000,
		}, nil
	}
}

// TestModelRuntimeReload 模型运行时热替换的根契约：
// 配置未变时复用同一快照；失效后按新配置重建，且 Provider（摘要器 / 记忆提取器 /
// 上下文管理器唯一的取模型通道）立即返回新实例
func TestModelRuntimeReload(t *testing.T) {
	originBuild := buildModelRuntime
	t.Cleanup(func() {
		buildModelRuntime = originBuild
		ResetModelRuntime()
	})

	var builds int32
	buildModelRuntime = stubModelRuntimeBuilder(&builds, nil)
	ResetModelRuntime()

	provider := ModelProvider()
	first, err := provider.ChatModel(context.Background())
	require.NoError(t, err)
	second, err := provider.ChatModel(context.Background())
	require.NoError(t, err)
	require.Same(t, first, second, "配置未变时必须复用同一模型实例")
	require.Equal(t, 1000, provider.ContextWindow())
	require.Equal(t, int32(1), atomic.LoadInt32(&builds), "未失效时不应重复构造")

	// 模拟管理员改写模型配置：宿主调用失效入口
	ResetModelRuntime()

	third, err := provider.ChatModel(context.Background())
	require.NoError(t, err)
	require.NotSame(t, first, third, "失效后 Provider 必须返回新模型实例，否则摘要/记忆提取继续走旧地址")
	require.Equal(t, 2000, provider.ContextWindow(), "上下文窗口也必须跟随新配置")
	require.Equal(t, int32(2), atomic.LoadInt32(&builds))
}

// TestModelRuntimeBuildFailureNotCached 构造失败不缓存：配置写错再修正后，
// 下一次调用即可恢复（否则管理员必须重启服务）
func TestModelRuntimeBuildFailureNotCached(t *testing.T) {
	originBuild := buildModelRuntime
	t.Cleanup(func() {
		buildModelRuntime = originBuild
		ResetModelRuntime()
	})

	var builds int32
	buildModelRuntime = stubModelRuntimeBuilder(&builds, errModelNotConfigured)
	ResetModelRuntime()

	_, err := ModelProvider().ChatModel(context.Background())
	require.ErrorIs(t, err, errModelNotConfigured)
	require.Equal(t, 0, ModelProvider().ContextWindow(), "模型不可用时窗口读作 0（跳过窗口检查）")

	// 配置修正
	buildModelRuntime = stubModelRuntimeBuilder(&builds, nil)
	model, err := ModelProvider().ChatModel(context.Background())
	require.NoError(t, err, "失败结果不得被缓存，修正配置后应立刻可用")
	require.NotNil(t, model)
}

// TestResetModelRuntimeResetsDefaultAgent 失效入口必须同时重置默认 Agent：
// 只清模型运行时会让 Agent 继续用旧模型服务新对话（缺陷类：改完配置仍连旧地址）
func TestResetModelRuntimeResetsDefaultAgent(t *testing.T) {
	originBuild := buildModelRuntime
	originAgent := defaultAgent
	t.Cleanup(func() {
		buildModelRuntime = originBuild
		defaultAgent = originAgent
		ResetModelRuntime()
	})

	var builds int32
	buildModelRuntime = stubModelRuntimeBuilder(&builds, nil)

	defaultAgentMu.Lock()
	defaultAgent = &Agent{id: DefaultAgentId}
	defaultAgentMu.Unlock()

	ResetModelRuntime()

	require.Nil(t, defaultAgent, "模型配置变更后默认 Agent 单例必须一起失效")
}

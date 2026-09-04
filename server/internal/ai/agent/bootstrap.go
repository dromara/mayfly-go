package agent

import (
	"context"
	"fmt"
	"sync"

	"mayfly-go/internal/ai/agent/contributor"
	"mayfly-go/internal/ai/agent/ext"
	aiconfig "mayfly-go/internal/ai/config"
	"mayfly-go/internal/ai/memory"
	"mayfly-go/internal/ai/session"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/pkg/logx"

	"github.com/cloudwego/eino/components/model"
)

// DefaultRuntime 默认运行时装配结果（进程级唯一，Build 后只读、并发共享）
type DefaultRuntime struct {
	// ContextManager 默认上下文管理器（持有 Registry）
	ContextManager *ContextManager
	// Registry 默认贡献者注册中心（内置扩展 + 宿主扩展 + 配置裁剪后）
	Registry *contributor.Registry
	// ChatModel 默认聊天模型（装配期单次获取，Agent/摘要器等复用，避免重复 IOC 查询）
	ChatModel model.AgenticModel
}

var (
	defaultRuntimeMu sync.Mutex
	defaultRuntime   *DefaultRuntime
)

// AssembleDefault 装配 AI 模块默认运行时（进程级一次性，并发安全）
//
// 统一宿主装配点（对齐 tokhub 宿主经 DI 组装）：
//
//	存储 → 会话/记忆管理器 → 贡献者注册中心（内置 InstallAll + 宿主扩展
//	追加 + 配置 WithFilter 裁剪）→ ContextManager
//
// newAgent 与 GetDefaultContextManager 均依赖本函数，且工具聚合发生在
// 装配之后，消除此前「工具聚合先于装配 → 扩展以 nil 依赖装配并永久失效」
// 的初始化时序缺陷（memory / 历史扩展 / mid-turn 压缩在默认链路失效）。
//
// 装配失败不缓存结果，下次调用重试。
func AssembleDefault(ctx context.Context) (*DefaultRuntime, error) {
	defaultRuntimeMu.Lock()
	defer defaultRuntimeMu.Unlock()

	if defaultRuntime != nil {
		return defaultRuntime, nil
	}

	rt, err := assembleDefault(ctx)
	if err != nil {
		return nil, err
	}
	defaultRuntime = rt
	return rt, nil
}

func assembleDefault(ctx context.Context) (*DefaultRuntime, error) {
	agentConf := aiconfig.GetAgentConfig()

	// 会话存储（未显式注入时使用本地 JSONL 存储，目录可配置）
	var sessionStore session.Store
	var err error
	if session.DefaultSessionStore != nil {
		sessionStore = session.DefaultSessionStore
	} else {
		sessionStore, err = session.NewStoreJSONL(agentConf.SessionDir)
		if err != nil {
			return nil, fmt.Errorf("create session store: %w", err)
		}
		session.DefaultSessionStore = sessionStore
	}

	chatModel, err := GetChatModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("get chat model: %w", err)
	}

	// 会话管理器（摘要 + 上下文窗口）
	sessionManager := session.NewManager(sessionStore)
	summaryConfig := session.DefaultSummaryConfig()
	if chatModel != nil && summaryConfig.Summarizer != nil {
		if llmSummarizer, ok := summaryConfig.Summarizer.(*session.LLMSummarizer); ok {
			llmSummarizer.WithChatModel(chatModel)
			logx.Info("ChatModel injected into LLM Summarizer")
		}
	}
	sessionManager.WithSummaryConfig(summaryConfig)
	sessionManager.WithContextWindow(aiconfig.GetModel().ContextWindow)

	// 记忆存储：application 层装配时注入 DB 后端（t_ai_memory 表，多实例共享）；
	// 未装配时降级本地 JSONL 存储（目录可配置，仅适用于单实例部署/单测）
	var memoryStore memory.Store
	if memory.DefaultStore != nil {
		memoryStore = memory.DefaultStore
	} else {
		var err error
		memoryStore, err = memory.NewJSONLStore(agentConf.MemoryDir)
		if err != nil {
			return nil, fmt.Errorf("create memory store: %w", err)
		}
	}
	memoryManager := memory.NewManager(memoryStore)

	// 贡献者注册中心：内置扩展 → 宿主级扩展追加 → 配置裁剪（WithFilter 生成只读视图）
	b := contributor.NewBuilder()
	ext.InstallAll(b, ext.Deps{
		MemoryManager:  memoryManager,
		SessionManager: sessionManager,
		ChatModel:      chatModel,
	})
	ext.RunHostInstallers(b)
	registry := b.Build().WithFilter(agentConf.DisabledExtensionSet())

	// 装配期激活（中断扩展装载、记忆提取装配等）：发生在裁剪后，被裁剪的扩展不激活；
	// 激活后冻结中断扩展注册表 —— 运行期零变更（多实例部署下各实例清单一致）。
	// 激活失败 fail-open（记日志不阻断装配），实现方幂等，装配重试安全
	if err := registry.Activate(ctx); err != nil {
		logx.WarnfContext(ctx, "[agent] contributor activation errors: %v", err)
	}
	tools.FreezeInterruptExtensions()

	// 上下文管理器（持有注册中心，BuildMessages 经通道聚合）
	cm, err := NewContextManager(&ContextManagerConfig{
		SessionManager: sessionManager,
		ChatModel:      chatModel,
		ContextWindow:  aiconfig.GetModel().ContextWindow,
		Registry:       registry,
	})
	if err != nil {
		return nil, err
	}

	// 长期记忆提取由 memory_extraction 扩展在装配期激活（原副作用装配已移除）

	logx.Infof("[agent] default runtime assembled, summary=[%s], contributors=%v",
		registry.Summary(), registry.ContributorIds())
	return &DefaultRuntime{ContextManager: cm, Registry: registry, ChatModel: chatModel}, nil
}

// GetDefaultContextManager 获取默认上下文管理器（并发安全，装配失败不缓存）
func GetDefaultContextManager() (*ContextManager, error) {
	rt, err := AssembleDefault(context.Background())
	if err != nil {
		return nil, err
	}
	return rt.ContextManager, nil
}

// GetDefaultRegistry 获取默认贡献者注册中心（并发安全，装配失败不缓存）
func GetDefaultRegistry() (*contributor.Registry, error) {
	rt, err := AssembleDefault(context.Background())
	if err != nil {
		return nil, err
	}
	return rt.Registry, nil
}

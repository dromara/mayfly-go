package agent

import (
	"context"
	"errors"
	"sync"

	aiconfig "mayfly-go/internal/ai/config"
	"mayfly-go/internal/ai/llm"
	"mayfly-go/internal/ai/protocol"
	"mayfly-go/pkg/logx"

	"github.com/cloudwego/eino/components/model"
)

// errModelNotConfigured 系统配置中模型标识为空（AiModelConfig.model 未配置）
//
// 与「协议不支持」区分开：这是管理员尚未完成 AI 配置，原样上抛会让用户看到
// 「no supported AI model protocol found」这种与现场无关的提示
var errModelNotConfigured = errors.New("ai model is not configured")

// modelRuntime 当前生效的模型运行时快照
//
// 一次系统配置（AiModelConfig）对应一个快照：ChatModel 实例 + 装配 Agent 时需要的
// 模型参数。快照本身不可变，配置变更时整体替换（见 ResetModelRuntime），因此持有
// 快照引用的一方（如已构造成型的 Agent）不会读到半新半旧的配置。
type modelRuntime struct {
	modelName     string
	chatModel     model.AgenticModel
	contextWindow int
	retry         *aiconfig.ModelRetryConfig
	failover      *aiconfig.ModelFailoverConfig
}

var (
	modelRuntimeMu  sync.Mutex
	modelRuntimeCur *modelRuntime

	// buildModelRuntime 按系统配置构造模型运行时
	// （单测替换点：让它无需数据库与真实模型即可验证热替换语义）
	buildModelRuntime = newModelRuntimeFromConfig
)

// newModelRuntimeFromConfig 读取系统配置并构造模型运行时
func newModelRuntimeFromConfig(ctx context.Context) (*modelRuntime, error) {
	conf := aiconfig.GetModel()
	if conf.Model == "" {
		return nil, errModelNotConfigured
	}

	chatModel, err := protocol.GetChatModel(ctx, conf)
	if err != nil {
		return nil, err
	}
	return &modelRuntime{
		modelName:     conf.Model,
		chatModel:     chatModel,
		contextWindow: conf.ContextWindow,
		retry:         conf.Retry,
		failover:      conf.Failover,
	}, nil
}

// GetModelRuntime 获取当前模型运行时（首次调用或配置变更后按系统配置重建，并发安全）
//
// 构造失败不缓存：配置改错后管理员修正配置，下一次调用即可恢复，无需重启。
func GetModelRuntime(ctx context.Context) (*modelRuntime, error) {
	modelRuntimeMu.Lock()
	defer modelRuntimeMu.Unlock()

	if modelRuntimeCur != nil {
		return modelRuntimeCur, nil
	}
	rt, err := buildModelRuntime(ctx)
	if err != nil {
		return nil, err
	}
	modelRuntimeCur = rt

	failoverCount := 0
	if rt.failover != nil {
		failoverCount = len(rt.failover.Fallbacks)
	}
	logx.InfofContext(ctx, "[ai] model runtime ready: model=%s, contextWindow=%d, retry=%v, failover=%d",
		rt.modelName, rt.contextWindow, rt.retry != nil, failoverCount)
	return rt, nil
}

// ResetModelRuntime 使模型运行时与默认 Agent 失效（模型配置变更后由宿主调用）
//
// 两者必须一起失效：默认 Agent 是模型运行时的持有者，只清运行时会让 Agent 继续用
// 旧模型服务新对话（表现为改完配置仍连旧地址）。档位（profile）Agent 是另一份缓存，
// 由宿主入口 application.ReloadModelConfig 一并重置。
//
// 正在执行的轮次持有旧实例，不受影响；新对话按新配置重建。
func ResetModelRuntime() {
	modelRuntimeMu.Lock()
	hadRuntime := modelRuntimeCur != nil
	modelRuntimeCur = nil
	modelRuntimeMu.Unlock()

	// 锁序：先释放 modelRuntimeMu 再取 Agent 锁。NewAgent 是 Agent 锁 →
	// modelRuntimeMu 的顺序，嵌套持锁会形成 AB-BA 死锁
	ResetDefaultAgent()

	if hadRuntime {
		logx.Info("[ai] model runtime invalidated, rebuilt with current config on next use")
	}
}

// modelProvider llm.Provider 实现：每次调用读取当前生效的模型运行时
//
// 摘要器 / 记忆提取器 / 上下文管理器在装配期拿到的是本 Provider（而非 ChatModel
// 实例），配置变更后无需重建这些组件即可用上新模型
type modelProvider struct{}

var _ llm.Provider = modelProvider{}

// ModelProvider 返回当前生效模型的访问入口（装配期注入长生命周期组件）
func ModelProvider() llm.Provider { return modelProvider{} }

func (modelProvider) ChatModel(ctx context.Context) (model.AgenticModel, error) {
	rt, err := GetModelRuntime(ctx)
	if err != nil {
		return nil, err
	}
	return rt.chatModel, nil
}

// ContextWindow 窗口读取不带调用方 ctx：窗口是配置事实，不随请求生命周期变化
// （BuildMessages 等热路径每轮都会读到最新窗口）；配置不可用时返回 0（跳过窗口检查）
func (modelProvider) ContextWindow() int {
	rt, err := GetModelRuntime(context.Background())
	if err != nil {
		return 0
	}
	return rt.contextWindow
}

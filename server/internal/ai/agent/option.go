package agent

import (
	"context"
	"mayfly-go/internal/ai/agent/contributor"
	"mayfly-go/pkg/utils/stringx"

	"github.com/cloudwego/eino/adk"
)

// option 定义了Agent的配置选项
type option func(*Agent)

func WithId(id string) option {
	return func(agent *Agent) {
		agent.id = id
	}
}

func WithName(name string) option {
	return func(agent *Agent) {
		agent.name = name
	}
}

func WithDescription(description string) option {
	return func(agent *Agent) {
		agent.description = description
	}
}

// WithRegistry 指定贡献者注册中心（多 Agent 场景唯一扩展入口）
//
// 不同 Agent 可持有不同注册中心，实现工具集/扩展差异化（建议经
// contributor.NewBuilder 独立装配 + WithFilter 裁剪）；未指定时使用
// 默认装配的注册中心。注意：Agent 的工具/中间件/上下文均源自注册中心，
// 不存在绕过插件机制的装配旁路。
func WithRegistry(registry *contributor.Registry) option {
	return func(agent *Agent) {
		agent.registry = registry
	}
}

func WithMiddlewares(middlewares ...adk.ChatModelAgentMiddleware) option {
	return func(agent *Agent) {
		agent.middlewares = middlewares
	}
}

func WithContextManager(contextManager *ContextManager) option {
	return func(agent *Agent) {
		agent.contextManager = contextManager
	}
}

func WithMaxStep(maxStep int) option {
	return func(agent *Agent) {
		agent.maxStep = maxStep
	}
}

type runOptions struct {
	adkRunOptions []adk.AgentRunOption
	sessionKey    string
	turnId        string // 一次完整回复id
	resumeParams  []any  // 恢复参数（由各中断类型通过扩展注册表转换）

	// interrupted 本轮是否以中断挂起结束（事件流中检测到 Interrupted 动作时置位，
	// 用于轮次结束原因细分为 TurnEndInterrupted）
	interrupted bool

	// onChunk 流式内容块回调函数
	// 当 Agent 产生增量输出（如 LLM 生成的每一个 Token 或片段）时触发。
	// 适用于实现前端"打字机"效果，实时展示 AI 的思考或回复内容。
	// 参数 adk.Message 包含当前增量的 Content, ReasoningContent 或 ToolCalls 等信息。
	onChunk func(context.Context, adk.Message) error // 流式输出回调函数

	// onEvent 完整事件回调函数
	// 当 Agent 运行过程中产生完整的事件节点（如工具调用开始/结束、LLM 完整响应生成完毕等）时触发。
	// 参数 *adk.AgentEvent 事件信息；adk.Message 为该事件对应的完整消息对象。
	onEvent func(context.Context, *adk.AgentEvent, adk.Message) error
}

// CallOnChunk 增量消息回调
func (ro *runOptions) CallOnChunk(ctx context.Context, chunk adk.Message) error {
	if ro.onChunk != nil {
		return ro.onChunk(ctx, chunk)
	}
	return nil
}

// CallOnEvent 事件回调
func (ro *runOptions) CallOnEvent(ctx context.Context, event *adk.AgentEvent, msg adk.Message) error {
	if ro.onEvent != nil {
		return ro.onEvent(ctx, event, msg)
	}
	return nil
}

// newRunOptions 创建一个RunOptions
func newRunOptions(ctx context.Context, opts ...RunOption) *runOptions {
	options := &runOptions{}
	for _, opt := range opts {
		opt(options)
	}

	if options.turnId == "" {
		options.turnId = stringx.SortableUUID()
	}

	return options
}

// runOption 定义了Agent执行时的配置选项

type RunOption func(*runOptions)

func WithRunAdkOptions(options ...adk.AgentRunOption) RunOption {
	return func(opts *runOptions) {
		opts.adkRunOptions = options
	}
}

func WithRunSessionKey(sessionKey string) RunOption {
	return func(opts *runOptions) {
		opts.sessionKey = sessionKey
	}
}

func WithTurnId(turnId string) RunOption {
	return func(opts *runOptions) {
		opts.turnId = turnId
	}
}

// WithResumeParams 设置恢复参数（审批或参数补全）
func WithResumeParams(resumeParams ...any) RunOption {
	return func(opts *runOptions) {
		opts.resumeParams = resumeParams
	}
}

// WithOnChunk 设置流式增量回调
// 该回调会在 Agent 产生每一个微小的输出片段（如 LLM 生成的单个 Token）时触发。
// 典型应用场景：
// 1. 前端实现“打字机”效果，实时逐字显示 AI 回复。
// 2. 实时展示 AI 的思考过程（ReasoningContent）。
// 注意：由于触发频率极高，回调函数内部应避免执行耗时操作，以免阻塞 Agent 运行。
func WithOnChunk(onChunk func(context.Context, adk.Message) error) RunOption {
	return func(opts *runOptions) {
		opts.onChunk = onChunk
	}
}

// WithOnEvent 设置完整事件节点回调
// 该回调会在 Agent 完成一个完整的逻辑节点（如一次完整的工具调用、一轮完整的 LLM 推理结束）时触发。
// 典型应用场景：
// 1. 记录详细的执行日志或审计轨迹。
// 2. 前端展示复杂的交互状态（如：“正在搜索知识库...”、“已调用天气接口”）。
// 3. 监控 Agent 的运行步骤和决策路径。
// 参数说明：
//   - event: 事件信息
//   - msg: 该事件产生的完整消息对象（而非增量片段）。
func WithOnEvent(onEvent func(context.Context, *adk.AgentEvent, adk.Message) error) RunOption {
	return func(opts *runOptions) {
		opts.onEvent = onEvent
	}
}

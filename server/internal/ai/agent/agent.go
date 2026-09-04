package agent

import (
	"context"
	"errors"
	"fmt"
	"mayfly-go/internal/ai/agent/contributor"
	aiconfig "mayfly-go/internal/ai/config"
	"mayfly-go/internal/ai/imsg"
	"mayfly-go/internal/ai/session"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/jsonx"
	"slices"
	"sync"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// 默认 Agent 单例：避免每个 WS 连接/流程任务都重建 agent 与工具链
var (
	defaultAgent   *Agent
	defaultAgentMu sync.Mutex
)

// GetDefaultAgent 获取默认 agent（进程级单例，并发安全，首次调用时懒初始化；
// 初始化失败不缓存，下次调用会重试）
//
// 注意：opts 仅在首次初始化时生效，后续调用复用单例 —— 进程内仅存在一个默认
// Agent。多 Agent 差异化（不同注册中心/工具集）请经 NewAgent + WithRegistry
// 显式构造独立实例，不要依赖 GetDefaultAgent 的 opts 传递。
func GetDefaultAgent(ctx context.Context, opts ...option) (*Agent, error) {
	defaultAgentMu.Lock()
	defer defaultAgentMu.Unlock()

	if defaultAgent != nil {
		return defaultAgent, nil
	}

	ag, err := NewAgent(ctx, opts...)
	if err != nil {
		return nil, err
	}
	defaultAgent = ag
	return defaultAgent, nil
}

// ResetDefaultAgent 重置默认 Agent 单例（下次 GetDefaultAgent 时懒重建）
//
// 供插件管理变更（MCP 服务器增删改/启停）后刷新工具清单，运行期即时生效：
// 正在执行中的轮次持有旧实例引用不受影响，新对话使用重建后的 Agent
// （重建时 mcpext 连接缓存按配置指纹复用未变更服务器的长连接，无重连开销）
func ResetDefaultAgent() {
	defaultAgentMu.Lock()
	defer defaultAgentMu.Unlock()
	defaultAgent = nil
}

const (
	DefaultAgentId = "main"
)

// NewAgent 创建 Agent
//
// 内部经统一宿主装配点（AssembleDefault）获取默认 Registry 与
// ContextManager 后再聚合工具/中间件，保证扩展以真实依赖装配。
func NewAgent(ctx context.Context, opts ...option) (*Agent, error) {
	agent := &Agent{
		id:          DefaultAgentId,
		name:        "OpsExpert",
		description: "an agent for general task",
		maxStep:     20,
	}

	for _, opt := range opts {
		opt(agent)
	}

	// 统一宿主装配：注册中心/上下文管理器未显式指定时装配默认运行时，
	// 确保扩展以真实依赖装配（修复此前 InitDefault(nil,nil) 时序缺陷）；
	// 全量显式指定时跳过装配，避免无谓的装配开销与激活副作用
	// （中断扩展装载、记忆提取装配等由自定义注册中心的装配方自行负责）
	if agent.registry == nil || agent.contextManager == nil {
		rt, err := AssembleDefault(ctx)
		if err != nil {
			return nil, err
		}
		if agent.registry == nil {
			agent.registry = rt.Registry
		}
		if agent.contextManager == nil {
			agent.contextManager = rt.ContextManager
		}
		if agent.chatModel == nil && rt.ChatModel != nil {
			// 复用宿主装配期单次获取的 ChatModel（避免重复 IOC 查询）
			agent.chatModel = rt.ChatModel
		}
	}

	// 工具列表统一经贡献者注册中心聚合（内置 + 宿主扩展，后注册者覆盖先注册者），
	// 不存在绕过插件机制的装配旁路；AgentId 已注入贡献上下文，条件工具组可按 Agent 过滤
	agent.tools = agent.registry.BuildTools(ctx, &contributor.ToolContributionContext{
		AgentId: agent.id,
	})

	// 中间件经注册中心聚合（内置 safety 中间件 + 插件），选项指定者追加在后
	agent.middlewares = append(agent.registry.CollectMiddlewares(ctx), agent.middlewares...)

	// 工具搜索（eino v0.9 tool search middleware）：工具总量超过阈值时，
	// MCP 工具转为 deferred —— 模型经 tool_search 元工具按需发现加载，
	// 避免大工具清单挤占上下文（未配置阈值时全量直注，行为不变）。
	// 接线契约：ToolsConfig 须换用静态工具清单，DynamicTools 由中间件在
	// BeforeAgent 阶段追加为可执行工具（避免 runCtx.Tools 重名重复注册）
	tsMw, staticTools := buildToolSearchMiddleware(ctx, agent.tools, aiconfig.GetAgentConfig().ToolSearchThreshold)
	if tsMw != nil {
		agent.middlewares = append(agent.middlewares, tsMw)
		agent.tools = staticTools
	}

	// 静态系统提示词：渲染 system_prompt.md 注入 adk Instruction（每轮由 adk 前置为 system 消息）
	if agent.instruction == "" {
		agent.instruction = SystemInstruction()
	}

	if agent.chatModel == nil {
		chatModel, err := GetChatModel(ctx)
		if err != nil {
			return nil, err
		}
		agent.chatModel = chatModel
	}

	modelConf := aiconfig.GetModel()
	adkAgent, err := adk.NewTypedChatModelAgent[*schema.AgenticMessage](ctx, &adk.TypedChatModelAgentConfig[*schema.AgenticMessage]{
		Name:          agent.name,
		Description:   agent.description,
		Instruction:   agent.instruction,
		Model:         agent.chatModel,
		MaxIterations: agent.maxStep,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: agent.tools,
			},
		},
		Handlers: agent.middlewares,
		// 模型调用重试与故障转移（eino v0.9 Model Retry / Model Failover 能力）：
		// 系统配置 retry.maxRetries 启用重试、failover.fallbacks 启用转移，
		// 均未配置时保持零重试零转移行为；瞬时失败（网络/429/5xx）先重试、
		// 重试耗尽后按序转移到备用模型；主动取消/超时不重试不转移
		// （保持停止即中止语义）
		ModelRetryConfig:    buildModelRetryConfig(modelConf.Retry),
		ModelFailoverConfig: buildModelFailoverConfig(modelConf.Failover),
	})
	if err != nil {
		return nil, err
	}
	agent.agent = adkAgent
	return agent, nil
}

// RunResult 一次 agent 运行的结果
//
// Usage 为本轮各次模型调用的 token 用量累计（多步工具循环每步各带一份 usage），
// 模型未上报 usage 时为 nil。供 api 层下发 TurnCompleted 事件与会话统计累计。
type RunResult struct {
	// Output 最终输出文本（最后一条消息内容）
	Output string
	// Usage 本轮 token 用量累计（无 usage 上报时为 nil）
	Usage *schema.TokenUsage
}

type Agent struct {
	agent     agenticAgent
	chatModel model.AgenticModel // agent使用的chat model

	id          string
	name        string // agent名称
	description string // agent描述
	instruction string // 系统提示词（静态部分，注入 adk Instruction）
	maxStep     int    // agent最大执行步数，防止死循环

	tools          []tool.BaseTool               // 可调用的工具列表（经 Registry 聚合）
	middlewares    []contributor.AgentMiddleware // 中间件（经 Registry 聚合 + 选项追加）
	registry       *contributor.Registry         // 贡献者注册中心（生命周期/用量回调/中间件聚合来源）
	contextManager *ContextManager               // 上下文管理器
}

// Run 运行agent（轮次管道：装配输入 → 执行 → 事件处理 → 收尾兑底）
//
// 消息以 session.Message（全链路统一内存结构）流转，仅在 runner 执行边界
// 经 ToAgenticMessages 转换为 eino AgenticMessage。
func (a *Agent) Run(ctx context.Context, messages []*session.Message, runOpts ...RunOption) (*RunResult, error) {
	ctx = contextx.WithTraceId(ctx)

	runOptions := newRunOptions(ctx, runOpts...)
	if runOptions.sessionKey != "" {
		ctx = session.WithSessionKey(ctx, runOptions.sessionKey)
		ctx = session.WithTurn(ctx, runOptions.turnId)
	}

	turnUserId := ""
	if la := contextx.GetLoginAccount(ctx); la != nil {
		turnUserId = fmt.Sprintf("%d", la.Id)
	}

	// 轮次生命周期钩子：轮次开始（fail-open，不阻断对话）
	a.registry.NotifyTurnStart(ctx, &contributor.TurnStartInput{
		SessionKey: runOptions.sessionKey,
		TurnId:     runOptions.turnId,
		UserId:     turnUserId,
	})
	for _, inputMsg := range messages {
		SetTurnId(inputMsg, runOptions.turnId)
	}

	checkPointStore, err := GetDefaultCheckPointStore()
	if err != nil {
		return nil, err
	}
	runner := adk.NewTypedRunner[*schema.AgenticMessage](adk.TypedRunnerConfig[*schema.AgenticMessage]{
		EnableStreaming: true,
		Agent:           a.agent,
		CheckPointStore: checkPointStore,
	})

	adkRunOptions := append(runOptions.adkRunOptions,
		adk.WithCallbacks(logCallback),
		adk.WithCheckPointID(runOptions.turnId))

	var events *adk.AsyncIterator[*agentEvent]
	var outputMessages []*session.Message

	// 轮次收尾兜底：无论正常返回、错误返回还是 panic，都保证错误消息下发、
	// 消息持久化与生命周期/用量回调执行
	defer func() {
		a.finalizeTurn(ctx, runOptions, turnUserId, messages, &outputMessages, err)
	}()

	if runOptions.resumeParams != nil {
		resumeMsgs, resumeEvents, resumeErr := a.resumeRun(ctx, runner, checkPointStore, runOptions, adkRunOptions)
		if resumeErr != nil {
			return nil, resumeErr
		}
		events = resumeEvents
		// 中断恢复消息下发
		for _, resumeMsg := range resumeMsgs {
			runOptions.CallOnEvent(ctx, nil, resumeMsg)
		}
	} else {
		events, err = a.startRun(ctx, runner, messages, adkRunOptions)
		if err != nil {
			return nil, err
		}
	}

	eventOutputMessages, err := a.handleEvents(ctx, events, runOptions)
	// 先合并已收到的消息，即使后续出错也不丢失部分输出
	outputMessages = append(outputMessages, eventOutputMessages...)

	if err != nil {
		if toolErr, ok := errors.AsType[*tools.ToolError](err); ok {
			// 工具调用失败，并且没有重试，则记录对应错误消息
			toolErrMsg := &session.Message{
				Role:       schema.Tool,
				Content:    tools.GetToolErrorMsg(err),
				ToolName:   toolErr.ToolName,
				ToolCallId: toolErr.ToolCallId,
			}
			SetTurnId(toolErrMsg, runOptions.turnId)
			SetToolStatus(toolErrMsg, tools.ToolStatusError)

			runOptions.CallOnEvent(ctx, nil, toolErrMsg)
			outputMessages = append(outputMessages, toolErrMsg)
			// 工具错误的 AI 回复消息由 defer 统一处理
		}
		// 其他类型的错误交给 defer 统一处理
	}

	if len(outputMessages) > 0 {
		return &RunResult{
			Output: outputMessages[len(outputMessages)-1].Content,
			Usage:  accumulateUsage(outputMessages),
		}, err
	}

	return &RunResult{Output: "finished without output message", Usage: accumulateUsage(outputMessages)}, err
}

// startRun 启动新轮次：经上下文管理器组装历史 + preamble 后交给 runner
//
// fail-open：历史组装失败降级为空历史（仅本轮输入参与对话），不阻断对话。
func (a *Agent) startRun(ctx context.Context, runner *agenticRunner, messages []*session.Message, adkRunOptions []adk.AgentRunOption) (*adk.AsyncIterator[*agentEvent], error) {
	contextMessages, buildErr := a.contextManager.BuildMessages(ctx, messages...)
	if buildErr != nil {
		logx.ErrorContext(ctx, buildErr.Error())
		contextMessages = []*session.Message{}
	}
	// runner 执行边界：统一内存结构 → eino AgenticMessage（开闭边界）
	return runner.Run(ctx, session.ToAgenticMessages(slices.Concat(contextMessages, messages)), adkRunOptions...), nil
}

// resumeRun 恢复中断挂起的轮次：预检 checkpoint → 转换恢复参数 → ResumeWithParams
//
// 返回待下发的中断恢复内部消息；成功后删除已消费的 checkpoint。
func (a *Agent) resumeRun(ctx context.Context, runner *agenticRunner, checkPointStore CheckPointStore, runOptions *runOptions, adkRunOptions []adk.AgentRunOption) ([]*session.Message, *adk.AsyncIterator[*agentEvent], error) {
	// 预检查 checkpoint 是否存在：中断挂起超过 TTL 或服务重启后 checkpoint 会失效，
	// 提前返回友好提示，避免用户操作后收到晦涩的 "checkpoint not exist" 错误
	if _, ok, getErr := checkPointStore.Get(ctx, runOptions.turnId); getErr != nil || !ok {
		logx.WarnfContext(ctx, "[Agent.Run] resume checkpoint not found, turnId=%s, err=%v", runOptions.turnId, getErr)
		return nil, nil, errors.New(i18n.T(imsg.InterruptExpired))
	}

	targets := map[string]any{}
	var resumeMsgs []*session.Message
	for _, v := range runOptions.resumeParams {
		data, ok := v.(*tools.InterruptResume)
		if !ok {
			continue
		}
		interruptId := data.InterruptId
		logx.InfofContext(ctx, "[Agent.Run] resume: interruptId=%s, type=%s, action=%s, payload=%v",
			interruptId, data.InterruptType, data.Action, jsonx.ToStr(data.Payload))

		// key -> interruptId  value -> 具体类型的恢复参数
		targets[interruptId] = data.ToTarget()

		// 类型特有的预处理（如参数补全类型：按 toolCallId 缓存 + 注入 Go context），
		// 通过中断扩展注册表分发，新增中断类型无需修改此处
		ctx = tools.PrepareInterruptResumeCtx(ctx, data)

		// 中断恢复消息
		internalResumeMessage := &session.Message{
			Role: session.RoleInternal,
		}
		internalResumeMessage.Extra = NewInternalMessageExtra(InternalMessageTypeResume, data)
		SetTurnId(internalResumeMessage, runOptions.turnId)
		SetActionId(internalResumeMessage, interruptId)
		resumeMsgs = append(resumeMsgs, internalResumeMessage)
	}

	events, err := runner.ResumeWithParams(ctx, runOptions.turnId, &adk.ResumeParams{
		Targets: targets,
	}, adkRunOptions...)
	if err != nil {
		return nil, nil, err
	}

	checkPointStore.Delete(ctx, runOptions.turnId)
	return resumeMsgs, events, nil
}

// finalizeTurn 轮次收尾（defer 执行）：错误兜底消息 → 消息持久化 → 结束原因细分 → 生命周期/用量回调
func (a *Agent) finalizeTurn(ctx context.Context, runOptions *runOptions, turnUserId string, inputMessages []*session.Message, outputMessages *[]*session.Message, runErr error) {
	// 显式 stop 触发的 ctx 取消属预期行为（TurnEndAborted），
	// 不作为错误包装为回复消息推送/持久化，避免污染会话历史
	if runErr != nil && errors.Is(runErr, context.Canceled) {
		runErr = nil
	}

	// 如果有未处理的错误，包装为 AI 回复消息推送给前端并保存
	// （工具错误已经在 Run 主流程单独处理，这里只处理其他类型的错误）
	if runErr != nil {
		errMsg := &session.Message{
			Role:    schema.Assistant,
			Content: runErr.Error(),
		}
		SetTurnId(errMsg, runOptions.turnId)
		// 推送错误消息给前端
		if err := runOptions.CallOnChunk(ctx, errMsg); err != nil {
			logx.WarnfContext(ctx, "agent push error message failed: %v", err)
		}
		// 加入 outputMessages 以便保存到历史记录
		*outputMessages = append(*outputMessages, errMsg)
	}

	// 结束原因细分：主动中止（ctx 取消）→ 异常终止 → 中断挂起 → 正常完成
	// （需在切换到无取消 ctx 前判定，主动中止依赖 ctx.Err()）
	cause := turnEndCause(ctx, runOptions, runErr)

	// 收尾持久化与钩子切换到无取消 ctx：显式 stop 后 turn ctx 已取消，
	// 仍需保证消息落盘与生命周期/用量回调执行
	ctx = context.WithoutCancel(ctx)

	// 保证消息持久化：无论正常返回、错误返回还是 panic，都会尝试保存
	// 避免崩溃或提前 return 导致整轮消息丢失
	saveMsgs := slices.Concat(inputMessages, *outputMessages)
	if len(saveMsgs) > 0 {
		if err := a.contextManager.AppendMsgs(ctx, saveMsgs...); err != nil {
			logx.ErrorfContext(ctx, "agent append message error: %v", err)
		}
	}

	// 轮次生命周期钩子：轮次结束（按因施策）
	a.registry.NotifyTurnEnd(ctx, &contributor.TurnEndInput{
		SessionKey: runOptions.sessionKey,
		TurnId:     runOptions.turnId,
		UserId:     turnUserId,
		Cause:      cause,
	})

	// token 用量回调（唯一出口，计量/计费/统计类扩展经此接入，fail-open）
	a.registry.NotifyTokenUsage(ctx, &contributor.TokenUsageInput{
		SessionKey: runOptions.sessionKey,
		TurnId:     runOptions.turnId,
		UserId:     turnUserId,
		Usage:      accumulateUsage(*outputMessages),
		Cause:      cause,
	})
}

// turnEndCause 判定轮次结束原因
func turnEndCause(ctx context.Context, runOptions *runOptions, runErr error) contributor.TurnEndCause {
	if errors.Is(ctx.Err(), context.Canceled) {
		return contributor.TurnEndAborted
	}
	if runErr != nil {
		return contributor.TurnEndError
	}
	if runOptions.interrupted {
		return contributor.TurnEndInterrupted
	}
	return contributor.TurnEndCompleted
}

// accumulateUsage 累计各次模型调用的 token 用量（多步工具循环每步各带一份 usage）
func accumulateUsage(msgs []*session.Message) *schema.TokenUsage {
	var usage *schema.TokenUsage
	for _, msg := range msgs {
		if msg.ResponseMeta == nil || msg.ResponseMeta.Usage == nil {
			continue
		}
		u := msg.ResponseMeta.Usage
		if usage == nil {
			usage = &schema.TokenUsage{}
		}
		usage.PromptTokens += u.PromptTokens
		usage.CompletionTokens += u.CompletionTokens
		usage.TotalTokens += u.TotalTokens
	}
	if usage != nil && usage.TotalTokens == 0 {
		// 部分模型不回 total，按 prompt + completion 兼底
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	return usage
}

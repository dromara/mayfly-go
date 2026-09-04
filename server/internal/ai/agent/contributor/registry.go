package contributor

import (
	"context"
	"mayfly-go/internal/ai/session"
	"mayfly-go/pkg/logx"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/components/tool"
)

// Channel 贡献通道：一种 Agent 能力注入维度。
// 新增通道只需定义 Channel 常量 + Builder 注册方法 + dispatch 方法，
// WithFilter / ContributorIds / Activate / Summary 等结构性操作基于统一条目
// 存储天然覆盖，无需逐一修改（消除重复 for 块）。
type Channel int

const (
	// ChannelContext 统一 Prompt 注入通道（记忆/技能目录/环境等轮次级片段）
	ChannelContext Channel = iota
	// ChannelTool 工具执行器注册通道（内置/条件工具组统一走此通道聚合）
	ChannelTool
	// ChannelHistory 对话历史消息段通道（短期记忆进历史的唯一通道）
	ChannelHistory
	// ChannelPreambleFooter preamble 后置注入通道（窗口余量提醒等）
	ChannelPreambleFooter
	// ChannelLifecycle 轮次生命周期钩子通道（轮次开始/结束按因施策）
	ChannelLifecycle
	// ChannelMiddleware 工具执行中间件通道（安全拦截等）
	ChannelMiddleware
	// ChannelTokenUsage token 用量回调通道（计量/计费/统计类扩展唯一出口）
	ChannelTokenUsage
	// ChannelService 装配期激活通道：无运行期调度，贡献者须实现 Activatable，
	// 由宿主在裁剪后经 Registry.Activate 一次性激活（如中断类型扩展装载、记忆提取装配）
	ChannelService
)

// channelNames 通道名（诊断日志与装配摘要用）
var channelNames = map[Channel]string{
	ChannelContext:        "context",
	ChannelTool:           "tool",
	ChannelHistory:        "history",
	ChannelPreambleFooter: "preamble_footer",
	ChannelLifecycle:      "lifecycle",
	ChannelMiddleware:     "middleware",
	ChannelTokenUsage:     "token_usage",
	ChannelService:        "service",
}

func (ch Channel) String() string {
	if name, ok := channelNames[ch]; ok {
		return name
	}
	return "unknown"
}

// registryEntry 统一条目：全部通道的贡献者共用一个存储结构
type registryEntry struct {
	channel Channel
	id      string
	c       Contributor
}

// Registry 贡献者注册中心（对齐 tokhub ExtensionRegistry）
//
// 通过 Builder 显式注册，Build 后只读、并发共享。
// 注册顺序即覆盖优先级（后注册胜出）：内置注册在前、业务扩展注册在后。
// 宿主显式持有 Registry 实例并注入 Agent / ContextManager，
// 不存在进程级可变全局单例（装配时序与并发安全由宿主装配点保证）。
type Registry struct {
	entries []registryEntry
}

// view 返回指定通道的贡献者切片（按注册顺序，类型断言收敛）
func view[T Contributor](r *Registry, ch Channel) []T {
	if r == nil {
		return nil
	}
	var out []T
	for _, e := range r.entries {
		if e.channel != ch {
			continue
		}
		if c, ok := e.c.(T); ok {
			out = append(out, c)
		}
	}
	return out
}

// Builder 注册中心构建器，显式注册各贡献者，Build 产出不可变 Registry
type Builder struct {
	registry Registry
}

// NewBuilder 创建空构建器
func NewBuilder() *Builder {
	return &Builder{}
}

// RegisterContext 注册上下文贡献者（统一 Prompt 注入通道，链式调用）
func (b *Builder) RegisterContext(c ContextContributor) *Builder {
	return b.register(ChannelContext, c)
}

// RegisterTool 注册工具贡献者（链式调用）
func (b *Builder) RegisterTool(c ToolContributor) *Builder {
	return b.register(ChannelTool, c)
}

// RegisterHistory 注册历史贡献者（对话历史消息段通道，链式调用）
func (b *Builder) RegisterHistory(c HistoryContributor) *Builder {
	return b.register(ChannelHistory, c)
}

// RegisterPreambleFooter 注册 preamble 后置注入贡献者（链式调用）
func (b *Builder) RegisterPreambleFooter(c PreambleFooterContributor) *Builder {
	return b.register(ChannelPreambleFooter, c)
}

// RegisterLifecycle 注册轮次生命周期贡献者（链式调用）
func (b *Builder) RegisterLifecycle(c TurnLifecycleContributor) *Builder {
	return b.register(ChannelLifecycle, c)
}

// RegisterMiddleware 注册工具执行中间件贡献者（链式调用）
func (b *Builder) RegisterMiddleware(c ToolMiddlewareContributor) *Builder {
	return b.register(ChannelMiddleware, c)
}

// RegisterTokenUsage 注册 token 用量回调贡献者（链式调用）
func (b *Builder) RegisterTokenUsage(c TokenUsageContributor) *Builder {
	return b.register(ChannelTokenUsage, c)
}

// RegisterService 注册装配期激活型贡献者（无运行期调度通道）
//
// 贡献者须实现 Activatable，由宿主在 WithFilter 裁剪后经 Registry.Activate
// 一次性激活（如中断类型扩展装载、记忆提取装配）；被裁剪的扩展不会激活。
func (b *Builder) RegisterService(c Contributor) *Builder {
	return b.register(ChannelService, c)
}

// register 统一注册入口（链式调用）
func (b *Builder) register(ch Channel, c Contributor) *Builder {
	b.registry.entries = append(b.registry.entries, registryEntry{channel: ch, id: c.Id(), c: c})
	return b
}

// Build 构建不可变注册中心
func (b *Builder) Build() *Registry {
	r := b.registry
	return &r
}

// ── HistoryContributor dispatch（对话历史消息段统一通道） ────────

// HasHistoryContributors 是否已注册历史贡献者（供宿主降级判断）
func (r *Registry) HasHistoryContributors() bool {
	return r != nil && len(view[HistoryContributor](r, ChannelHistory)) > 0
}

// CollectHistory 构建对话历史（拼接所有 HistoryContributor 贡献段 + 统一 normalize）
//
// 合并语义：按注册顺序拼接各贡献者消息段，配对修复对**合并后的全量列表**
// 统一执行一次（这正是 dispatch 层职责，贡献者不应各自为政）。
// fail-open：单个贡献者失败记日志跳过，不阻断主流程。
func (r *Registry) CollectHistory(ctx context.Context, bc *HistoryBuildContext) []*session.Message {
	if r == nil {
		return nil
	}
	var history []*session.Message
	for _, c := range view[HistoryContributor](r, ChannelHistory) {
		msgs, err := c.ContributeMessages(ctx, bc)
		if err != nil {
			logx.WarnfContext(ctx, "[contributor] history contributor %s contribute error: %v", c.Id(), err)
			continue
		}
		history = append(history, msgs...)
	}
	// 统一配对修复：必须对合并后的全量列表执行（而非各贡献者分段修复）
	return NormalizeHistory(history)
}

// TryMidTurnCompaction 尝试 mid-turn 紧急压缩（按注册逆序，首个成功者胜出）
//
// token 口径由宿主估算后经 params 传入；贡献者只负责判断阈值与执行压缩。
// 逆序遍历使后注册的插件压缩策略可覆盖内置策略（覆盖协议）。
// 仅实现了 MidTurnCompactor 可选能力的贡献者参与竞争。
func (r *Registry) TryMidTurnCompaction(ctx context.Context, history []*session.Message, params *MidTurnCompactionParams) ([]*session.Message, *MidTurnCompactionInfo) {
	if r == nil {
		return history, nil
	}
	contribs := view[HistoryContributor](r, ChannelHistory)
	for i := len(contribs) - 1; i >= 0; i-- {
		if mc, ok := contribs[i].(MidTurnCompactor); ok {
			if compacted, info := mc.TryMidTurnCompaction(ctx, history, params); info != nil {
				return compacted, info
			}
		}
	}
	return history, nil
}

// ── PreambleFooterContributor dispatch（preamble 后置注入统一通道） ────────

// CollectPreambleFooters 收集全部 preamble 后置注入文本（按注册顺序聚合）
//
// 调用点：preamble 片段与合并后 history 均就绪后统一调度一次；返回文本
// 由调用方以空行分隔追加到 preamble 尾部。fail-open。
func (r *Registry) CollectPreambleFooters(ctx context.Context, fc *PreambleFooterContext) []string {
	if r == nil {
		return nil
	}
	var footers []string
	for _, c := range view[PreambleFooterContributor](r, ChannelPreambleFooter) {
		text := c.ContributePreambleFooter(ctx, fc)
		if text == "" {
			continue
		}
		footers = append(footers, text)
	}
	return footers
}

// CollectContext 收集全部上下文贡献者的片段（按注册顺序，fail-open）
func (r *Registry) CollectContext(ctx context.Context, in *TurnInput) []PromptFragment {
	if r == nil {
		return nil
	}
	var fragments []PromptFragment
	for _, c := range view[ContextContributor](r, ChannelContext) {
		frags, err := c.ContributeTurnContext(ctx, in)
		if err != nil {
			logx.WarnfContext(ctx, "[contributor] context contributor %s contribute error: %v", c.Id(), err)
			continue
		}
		fragments = append(fragments, fragmentWithDefaults(c, frags)...)
	}
	return fragments
}

// BuildTools 聚合全部工具贡献者的工具（按注册顺序，fail-open）
//
// 同名工具覆盖协议：后注册者覆盖先注册者（保留先注册位置），
// 内置工具贡献者注册在前、业务插件注册在后，插件可替换内置实现。
func (r *Registry) BuildTools(ctx context.Context, tc *ToolContributionContext) []tool.BaseTool {
	if r == nil {
		return nil
	}
	// ordered 保持先注册位置，覆盖时仅替换实现
	ordered := make([]tool.BaseTool, 0, 16)
	index := make(map[string]int)
	for _, c := range view[ToolContributor](r, ChannelTool) {
		toolList, err := c.Tools(ctx, tc)
		if err != nil {
			logx.WarnfContext(ctx, "[contributor] tool contributor %s build tools error: %v", c.Id(), err)
			continue
		}
		for _, t := range toolList {
			ti, err := t.Info(ctx)
			if err != nil {
				logx.WarnfContext(ctx, "[contributor] tool contributor %s get tool info error: %v", c.Id(), err)
				continue
			}
			if i, ok := index[ti.Name]; ok {
				// 同名覆盖（后注册胜出）：高价值诊断事件，插件替换内置实现的唯一痕迹
				logx.InfofContext(ctx, "[contributor] tool %s overridden by contributor %s (was index %d)", ti.Name, c.Id(), i)
				ordered[i] = t
				continue
			}
			index[ti.Name] = len(ordered)
			ordered = append(ordered, t)
		}
	}
	return ordered
}

// NotifyTurnStart 通知全部生命周期贡献者轮次开始（fail-open）
func (r *Registry) NotifyTurnStart(ctx context.Context, in *TurnStartInput) {
	if r == nil {
		return
	}
	for _, c := range view[TurnLifecycleContributor](r, ChannelLifecycle) {
		if err := c.OnTurnStart(ctx, in); err != nil {
			logx.WarnfContext(ctx, "[contributor] lifecycle contributor %s on turn start error: %v", c.Id(), err)
		}
	}
}

// NotifyTurnEnd 通知全部生命周期贡献者轮次结束（fail-open）
func (r *Registry) NotifyTurnEnd(ctx context.Context, in *TurnEndInput) {
	if r == nil {
		return
	}
	for _, c := range view[TurnLifecycleContributor](r, ChannelLifecycle) {
		if err := c.OnTurnEnd(ctx, in); err != nil {
			logx.WarnfContext(ctx, "[contributor] lifecycle contributor %s on turn end error: %v", c.Id(), err)
		}
	}
}

// ── ToolMiddlewareContributor dispatch（工具执行中间件统一通道） ────────

// CollectMiddlewares 聚合全部中间件贡献者的中间件（按注册顺序，fail-open）
func (r *Registry) CollectMiddlewares(ctx context.Context) []AgentMiddleware {
	if r == nil {
		return nil
	}
	var middlewares []AgentMiddleware
	for _, c := range view[ToolMiddlewareContributor](r, ChannelMiddleware) {
		mws, err := c.Middlewares(ctx)
		if err != nil {
			logx.WarnfContext(ctx, "[contributor] middleware contributor %s collect error: %v", c.Id(), err)
			continue
		}
		middlewares = append(middlewares, mws...)
	}
	return middlewares
}

// ── TokenUsageContributor dispatch（token 用量回调统一通道） ────────

// NotifyTokenUsage 通知全部 token 用量贡献者（轮次结束，fail-open）
func (r *Registry) NotifyTokenUsage(ctx context.Context, in *TokenUsageInput) {
	if r == nil {
		return
	}
	for _, c := range view[TokenUsageContributor](r, ChannelTokenUsage) {
		if err := c.OnTokenUsage(ctx, in); err != nil {
			logx.WarnfContext(ctx, "[contributor] token usage contributor %s error: %v", c.Id(), err)
		}
	}
}

// WithFilter 构建剔除指定 Id 贡献者的只读视图
//
// 用于宿主按配置裁剪扩展，无需改代码；被剔除的条目被移除，
// 各 dispatch 已有空列表短路，行为等价于「未安装该贡献者」。
func (r *Registry) WithFilter(disabled map[string]struct{}) *Registry {
	if r == nil || len(disabled) == 0 {
		return r
	}
	filtered := &Registry{}
	for _, e := range r.entries {
		if _, ok := disabled[e.id]; ok {
			continue
		}
		filtered.entries = append(filtered.entries, e)
	}
	return filtered
}

// EstimateTokens 粗略估算文本 token 数（1 token ≈ 4 字符，宿主口径）
func EstimateTokens(text string) int {
	return (utf8.RuneCountInString(text) + 3) / 4
}

// ContributorIds 已注册贡献者的稳定标识清单（去重，供启动日志诊断）
func (r *Registry) ContributorIds() []string {
	if r == nil {
		return nil
	}
	seen := make(map[string]struct{}, len(r.entries))
	ids := make([]string, 0, len(r.entries))
	for _, e := range r.entries {
		if _, ok := seen[e.id]; ok {
			continue
		}
		seen[e.id] = struct{}{}
		ids = append(ids, e.id)
	}
	return ids
}

// ChannelCount 指定通道的贡献者数量（诊断用）
func (r *Registry) ChannelCount(ch Channel) int {
	if r == nil {
		return 0
	}
	count := 0
	for _, e := range r.entries {
		if e.channel == ch {
			count++
		}
	}
	return count
}

// Summary 各通道装配摘要（如 "context=3, tool=4, ..."，启动日志诊断用）
func (r *Registry) Summary() string {
	if r == nil {
		return "<nil>"
	}
	counts := make(map[Channel]int, len(channelNames))
	for _, e := range r.entries {
		counts[e.channel]++
	}
	var sb strings.Builder
	for ch := ChannelContext; ch <= ChannelService; ch++ {
		if counts[ch] == 0 {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(ch.String())
		sb.WriteByte('=')
		sb.WriteString(strconv.Itoa(counts[ch]))
	}
	return sb.String()
}

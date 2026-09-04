package agent

import (
	"context"
	"errors"
	"fmt"
	"mayfly-go/internal/ai/agent/contributor"
	"mayfly-go/internal/ai/session"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/logx"
	"strings"
	"sync"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// ContextManagerConfig ContextManager 配置
type ContextManagerConfig struct {
	SessionManager *session.Manager   // 会话管理器（必需）
	ChatModel      model.AgenticModel // ChatModel 实例
	ContextWindow  int                // 模型上下文窗口（token，0 表示未配置）
	// Registry 贡献者注册中心（可选，nil 时历史降级直读会话管理器）
	Registry *contributor.Registry
}

// DefaultContextManagerConfig 返回默认配置
func DefaultContextManagerConfig() *ContextManagerConfig {
	return &ContextManagerConfig{}
}

type ContextManager struct {
	sessionManager *session.Manager      // 会话管理器
	chatModel      model.AgenticModel    // ChatModel 实例，用于 LLM 调用
	contextWindow  int                   // 模型上下文窗口（token），供贡献者按窗口调整注入策略
	registry       *contributor.Registry // 贡献者注册中心（nil 时历史降级直读会话管理器）
	mu             sync.RWMutex          // 读写锁，保护并发访问
}

// NewContextManager 创建并初始化 ContextManager 实例
func NewContextManager(config *ContextManagerConfig) (*ContextManager, error) {
	if config == nil {
		config = DefaultContextManagerConfig()
	}

	// 验证必需参数
	if config.SessionManager == nil {
		return nil, fmt.Errorf("SessionManager is required")
	}

	// ChatModel 必须由外部注入（可选）
	chatModel := config.ChatModel

	cm := &ContextManager{
		sessionManager: config.SessionManager,
		chatModel:      chatModel,
		contextWindow:  config.ContextWindow,
		registry:       config.Registry,
	}

	return cm, nil
}

// WithChatModel 手动覆盖 ChatModel 实例（可选）
func (c *ContextManager) WithChatModel(chatModel model.AgenticModel) *ContextManager {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.chatModel = chatModel
	return c
}

// GetSessionKey 获取会话Key
func (c *ContextManager) GetSessionKey(ctx context.Context) string {
	return session.GetSessionKey(ctx)
}

// BuildMessages 从上下文中构建消息列表，供Agent执行使用
//
// 对齐 tokhub start 路径的组装顺序：
//  1. 历史：经 HistoryContributor 注册表收集消息段（多贡献者拼接 + 统一
//     NormalizeHistory 配对修复）；未装配历史扩展时降级直读会话管理器
//  2. preamble：经 ContextContributor 注册表收集片段，预算裁剪后拼装
//     （每轮重建不进历史）
//  3. mid-turn 紧急压缩：preamble 与合并后 history 均就绪后，含 preamble
//     占用口径按注册逆序竞争执行（后注册的插件压缩策略覆盖内置）
func (c *ContextManager) BuildMessages(ctx context.Context, inputMsgs ...*session.Message) ([]*session.Message, error) {
	sessionKey := c.GetSessionKey(ctx)
	if sessionKey == "" {
		return nil, errors.New("session key is empty")
	}
	userId := c.getLoginUserId(ctx)

	// 1. 历史：经 HistoryContributor 注册表收集（短期记忆进历史的唯一通道；
	// registry 为 nil 时降级直读会话管理器）
	var history []*session.Message
	if c.registry.HasHistoryContributors() {
		history = c.registry.CollectHistory(ctx, &contributor.HistoryBuildContext{
			SessionKey: sessionKey,
		})
	} else {
		// 降级：未装配历史扩展时直读会话管理器（同样执行统一配对修复，
		// 保证与主路径行为一致，避免双路径漂移）
		var err error
		history, err = c.sessionManager.GetHistory(ctx, sessionKey)
		if err != nil {
			return nil, err
		}
		history = contributor.NormalizeHistory(history)
	}

	// 2. preamble：经 ContextContributor 注册表收集动态上下文片段并拼装
	in := &contributor.TurnInput{
		UserId:   userId,
		UserText: extractUserText(inputMsgs),
	}
	preamble := contributor.BuildPreamble(c.registry.CollectContext(ctx, in))
	preambleTokens := contributor.EstimateTokens(preamble)
	historyTokens := session.EstimateHistoryTokens(history)

	// 3. mid-turn 紧急压缩（含 preamble 占用口径；逆序单赢家，后注册覆盖内置）
	if c.contextWindow > 0 && preamble != "" {
		compacted, info := c.registry.TryMidTurnCompaction(ctx, history, &contributor.MidTurnCompactionParams{
			SessionKey:     sessionKey,
			ContextWindow:  c.contextWindow,
			PreambleTokens: preambleTokens,
			HistoryTokens:  historyTokens,
		})
		if info != nil {
			history = compacted
			historyTokens = info.CompressedTokens
			logx.InfofContext(ctx, "[ext] mid-turn compaction applied: %d -> %d tokens",
				info.OriginalTokens, info.CompressedTokens)
		}
	}

	// 4. preamble 后置注入：经 PreambleFooterContributor 注册表收集
	//（窗口余量提醒等，测量与呈现分离，与压缩决策同口径）
	footers := c.registry.CollectPreambleFooters(ctx, &contributor.PreambleFooterContext{
		Budget: &contributor.PreambleBudgetStatus{
			ContextWindow:        c.contextWindow,
			PreambleTokens:       preambleTokens,
			HistoryTokens:        historyTokens,
			ReservedOutputTokens: c.contextWindow / 5,
		},
	})
	for _, footer := range footers {
		preamble += "\n\n" + footer
	}

	if preamble != "" {
		history = append([]*session.Message{{Role: schema.System, Content: preamble}}, history...)
		logx.DebugfContext(ctx, "injected preamble (%d chars) into context", len(preamble))
	}

	return history, nil
}

// getLoginUserId 获取当前登录用户 ID（未登录返回空）
func (c *ContextManager) getLoginUserId(ctx context.Context) string {
	if la := contextx.GetLoginAccount(ctx); la != nil {
		return fmt.Sprintf("%d", la.Id)
	}
	return ""
}

// extractUserText 提取输入消息中的用户文本（用于 $skill-code 显式提及等）
func extractUserText(inputMsgs []*session.Message) string {
	var sb strings.Builder
	for _, m := range inputMsgs {
		if m.Role == schema.User && m.Content != "" {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(m.Content)
		}
	}
	return sb.String()
}

// AppendMsgs 追加消息到会话，并在达到阈值时触发自动摘要
func (c *ContextManager) AppendMsgs(ctx context.Context, msgs ...*session.Message) error {
	if len(msgs) == 0 {
		return nil
	}

	sessionKey := c.GetSessionKey(ctx)
	if sessionKey == "" {
		return errors.New("session key is empty")
	}

	// 先追加消息
	if err := c.sessionManager.AppendMsgs(ctx, sessionKey, msgs...); err != nil {
		return err
	}

	return nil
}

// ClearHistory 清空会话历史
func (c *ContextManager) ClearHistory(ctx context.Context) error {
	return c.sessionManager.ClearHistory(ctx, c.GetSessionKey(ctx))
}

// GetSessionMeta 获取会话元数据
func (c *ContextManager) GetSessionMeta(ctx context.Context) (*session.SessionMeta, error) {
	sessionKey := c.GetSessionKey(ctx)
	if sessionKey == "" {
		return nil, errors.New("session key is empty")
	}
	return c.sessionManager.GetMeta(ctx, sessionKey)
}

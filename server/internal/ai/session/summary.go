package session

import (
	"context"
	"fmt"
	"mayfly-go/internal/ai/prompt"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// 自动摘要和压缩的默认阈值配置
const (
	// DefaultMessageThreshold 消息数量阈值，超过此数量触发自动摘要。
	// 仅承担摘要节奏职责（控制单次摘要输入规模），窗口压力判断以 token 口径为准
	// （含工具调用的重对话每轮可产生多条消息，条数阈值不宜过小）
	DefaultMessageThreshold = 30
	// DefaultTokenThreshold Token 数量阈值，超过此数量触发自动压缩
	DefaultTokenThreshold = 100000
	// DefaultSummaryKeepCount 摘要后保留的最近消息数量
	DefaultSummaryKeepCount = 5
)

// Summarizer 摘要器接口
type Summarizer interface {
	// GenerateSummary 生成会话摘要
	GenerateSummary(ctx context.Context, messages []*Message) (string, error)
}

// SummaryConfig 摘要配置
type SummaryConfig struct {
	MessageThreshold int        // 消息数量阈值
	TokenThreshold   int        // Token 数量阈值
	KeepRecentCount  int        // 摘要后保留的最近消息数量
	Enabled          bool       // 是否启用自动摘要
	Summarizer       Summarizer // 摘要器（可选，默认使用 LLM 摘要器）
}

// DefaultSummaryConfig 返回默认的摘要配置
func DefaultSummaryConfig() *SummaryConfig {
	return &SummaryConfig{
		MessageThreshold: DefaultMessageThreshold,
		TokenThreshold:   DefaultTokenThreshold,
		KeepRecentCount:  DefaultSummaryKeepCount,
		Enabled:          true,
		Summarizer:       NewLLMSummarizer(), // 默认使用 LLM 摘要器
	}
}

// LLMSummarizer 基于 LLM 的摘要器
type LLMSummarizer struct {
	maxMessages int                // 用于摘要的最大消息数，避免超出上下文限制
	chatModel   model.AgenticModel // ChatModel 实例（可选，不设置则自动获取）
}

// NewLLMSummarizer 创建基于 LLM 的摘要器
func NewLLMSummarizer() *LLMSummarizer {
	return &LLMSummarizer{
		maxMessages: 50, // 默认最多使用 50 条消息进行摘要
	}
}

// WithMaxMessages 设置用于摘要的最大消息数
func (s *LLMSummarizer) WithMaxMessages(maxMessages int) *LLMSummarizer {
	s.maxMessages = maxMessages
	return s
}

// WithChatModel 设置 ChatModel 实例
func (s *LLMSummarizer) WithChatModel(chatModel model.AgenticModel) *LLMSummarizer {
	s.chatModel = chatModel
	return s
}

// GenerateSummary 使用 LLM 生成智能摘要
func (s *LLMSummarizer) GenerateSummary(ctx context.Context, messages []*Message) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	// 尝试使用 LLM 生成摘要，如果失败则降级为简单摘要
	summary, err := s.generateLLMSummary(ctx, messages)
	if err != nil {
		logx.WarnfContext(ctx, "LLM summary generation failed: %v, fallback to simple summary", err)
		return s.generateSimpleSummary(messages), nil
	}

	// 如果 LLM 返回空摘要，也降级
	if summary == "" {
		logx.WarnfContext(ctx, "LLM returned empty summary, fallback to simple summary")
		return s.generateSimpleSummary(messages), nil
	}

	logx.InfofContext(ctx, "LLM summary generated successfully, length: %d", len(summary))
	return summary, nil
}

// generateLLMSummary 尝试使用 LLM 生成摘要（可能 panic）
func (s *LLMSummarizer) generateLLMSummary(ctx context.Context, messages []*Message) (summary string, err error) {
	defer gox.Recover()

	// ChatModel 必须由外部注入
	if s.chatModel == nil {
		return "", fmt.Errorf("ChatModel not configured, cannot generate LLM summary")
	}

	// 准备用于摘要的消息（限制数量避免超出上下文）
	messagesForSummary := messages
	if len(messages) > s.maxMessages {
		messagesForSummary = messages[len(messages)-s.maxMessages:]
	}

	// 构建摘要提示
	prompt := s.buildSummaryPrompt(messagesForSummary)

	// 调用 LLM 生成摘要（System 设定角色，User 提供待处理内容；摘要为非交互场景，
	// 直接用 Generate 非流式获取完整结果）
	response, err := s.chatModel.Generate(ctx, []*schema.AgenticMessage{
		schema.SystemAgenticMessage("你是对话摘要专家，负责将对话历史压缩为结构化的摘要。保留关键信息，去除冗余细节。"),
		schema.UserAgenticMessage(prompt),
	})
	if err != nil {
		return "", fmt.Errorf("LLM generate: %w", err)
	}

	return ExtractText(response), nil
}

// buildSummaryPrompt 构建摘要提示
func (s *LLMSummarizer) buildSummaryPrompt(messages []*Message) string {
	var conversationText strings.Builder

	// 格式化对话内容，保留完整的上下文
	for _, msg := range messages {
		switch msg.Role {
		case schema.User:
			conversationText.WriteString("\n--- 用户消息 ---\n")
			conversationText.WriteString(msg.Content)
			conversationText.WriteString("\n")

		case schema.Assistant:
			conversationText.WriteString("\n--- 助手消息 ---\n")

			// 助手的文本回复
			if msg.Content != "" {
				conversationText.WriteString("回复内容:\n")
				conversationText.WriteString(msg.Content)
				conversationText.WriteString("\n")
			}

			// 工具调用详情（保留完整信息）
			if len(msg.ToolCalls) > 0 {
				conversationText.WriteString("工具调用:\n")
				for _, tc := range msg.ToolCalls {
					conversationText.WriteString(fmt.Sprintf("  - 工具ID: %s\n", tc.ID))
					conversationText.WriteString(fmt.Sprintf("  - 工具名称: %s\n", tc.Function.Name))
					if tc.Function.Arguments != "" {
						conversationText.WriteString(fmt.Sprintf("  - 调用参数: %s\n", tc.Function.Arguments))
					}
				}
			}

		case schema.Tool:
			conversationText.WriteString("\n--- 工具结果 ---\n")
			conversationText.WriteString(fmt.Sprintf("工具名称: %s\n", msg.ToolName))
			conversationText.WriteString(fmt.Sprintf("调用ID: %s\n", msg.ToolCallId))

			// 工具返回内容（限制长度但保持关键信息）
			// rune 安全截断（prefixLen=length 走前缀截断分支；len 按字节计会把中文截出半个字符）
			content := stringx.Truncate(msg.Content, 5000, 5000, "\n...[内容过长已截断]")
			conversationText.WriteString(fmt.Sprintf("返回结果:\n%s\n", content))

		case schema.System:
			// System 消息通常是旧摘要，完整保留并明确标识
			if strings.Contains(msg.Content, "之前的对话摘要") {
				// 直接写入原始 System 内容，避免重复包装标题
				conversationText.WriteString(msg.Content)
				conversationText.WriteString("\n")
			} else {
				conversationText.WriteString(fmt.Sprintf("\n[System]: %s\n", msg.Content))
			}

		default:
			conversationText.WriteString(fmt.Sprintf("\n[%s]: %s\n", msg.Role, msg.Content))
		}
	}

	val, err := prompt.GetPrompt("internal/session_summary.md", collx.Kvs("history", conversationText.String()))
	if err != nil {
		logx.Warnf("Failed to get summary prompt: %v, using fallback template", err)
		// 降级：使用简化的提示词
		return fmt.Sprintf("请对以下对话历史生成完整、准确的摘要：\n\n%s", conversationText.String())
	}
	return val
}

// generateSimpleSummary 生成简单的规则-based 摘要（降级方案）
func (s *LLMSummarizer) generateSimpleSummary(messages []*Message) string {
	var summaryParts []string
	userQuestions := 0
	toolCalls := 0

	for _, msg := range messages {
		switch msg.Role {
		case schema.User:
			userQuestions++
			// 提取用户问题的前 100 个字符作为关键点（rune 安全截断）
			content := stringx.Truncate(msg.Content, 100, 100, "...")
			summaryParts = append(summaryParts, fmt.Sprintf("用户提问: %s", content))

		case schema.Assistant:
			// 统计工具调用
			if len(msg.ToolCalls) > 0 {
				toolCalls += len(msg.ToolCalls)
			}

		case schema.Tool:
			// 记录工具执行结果摘要
			if msg.ToolName != "" {
				summaryParts = append(summaryParts, fmt.Sprintf("执行工具: %s", msg.ToolName))
			}
		}
	}

	// 构建摘要文本
	summary := fmt.Sprintf("对话概览: 共 %d 轮用户提问, %d 次工具调用\n", userQuestions, toolCalls)
	if len(summaryParts) > 0 {
		// 只保留最近的 5 个关键点
		if len(summaryParts) > 5 {
			summaryParts = summaryParts[len(summaryParts)-5:]
		}
		summary += "关键内容:\n"
		for i, part := range summaryParts {
			summary += fmt.Sprintf("%d. %s\n", i+1, part)
		}
	}

	return summary
}

package memory

import (
	"context"
	"fmt"
	"mayfly-go/internal/ai/pkg/utils"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Extractor 记忆提取器接口
type Extractor interface {
	// ExtractFromMessages 从消息历史中提取记忆
	ExtractFromMessages(ctx context.Context, userID string, messages []adk.Message) ([]*MemoryItem, error)
}

// LLMExtractorConfig LLM提取器配置
type LLMExtractorConfig struct {
	Enabled         bool                       // 是否启用
	Temperature     float64                    // 温度参数 (0-1)
	MaxTokens       int                        // 最大token数
	MinConfidence   float64                    // 最小置信度阈值
	MaxItemsPerCall int                        // 单次提取最大记忆数量
	ChatModel       model.ToolCallingChatModel // ChatModel 实例
}

// DefaultLLMExtractorConfig 返回默认配置
func DefaultLLMExtractorConfig() *LLMExtractorConfig {
	return &LLMExtractorConfig{
		Enabled:         true,
		Temperature:     0.3, // 较低温度，保证提取稳定性
		MaxTokens:       500,
		MinConfidence:   0.7,
		MaxItemsPerCall: 10,
		ChatModel:       nil, // 需要外部设置
	}
}

// LLMExtractor 基于LLM的记忆提取器
type LLMExtractor struct {
	config *LLMExtractorConfig
}

// MemoryExtractionResult LLM提取结果结构
type MemoryExtractionResult struct {
	Type       string   `json:"type"`             // 记忆类型: preference/fact/skill/experience
	Content    string   `json:"content"`          // 记忆内容（自然语言描述）
	Tags       []string `json:"tags"`             // 标签
	Confidence float64  `json:"confidence"`       // 置信度（仅用于过滤，不存储）
	Reason     string   `json:"reason,omitempty"` // 提取原因
}

// NewLLMExtractor 创建LLM提取器
func NewLLMExtractor() *LLMExtractor {
	return &LLMExtractor{
		config: DefaultLLMExtractorConfig(),
	}
}

// WithConfig 设置配置
func (e *LLMExtractor) WithConfig(config *LLMExtractorConfig) *LLMExtractor {
	if config != nil {
		e.config = config
	}
	return e
}

var _ Extractor = (*LLMExtractor)(nil)

// ExtractFromMessages 使用LLM从消息中提取记忆
func (e *LLMExtractor) ExtractFromMessages(ctx context.Context, userID string, messages []adk.Message) ([]*MemoryItem, error) {
	if !e.config.Enabled || len(messages) == 0 {
		return []*MemoryItem{}, nil
	}

	// 检查是否配置了 ChatModel
	if e.config.ChatModel == nil {
		logx.WarnfContext(ctx, "LLM extractor ChatModel not configured, skipping extraction")
		return []*MemoryItem{}, nil
	}

	// 只处理最近的消息（最多10条）
	recentMessages := messages
	if len(recentMessages) > 10 {
		recentMessages = recentMessages[len(recentMessages)-10:]
	}

	// 构建提示词
	prompt := e.buildExtractionPrompt(recentMessages)

	// 尝试使用 LLM 生成提取结果，如果失败则降级返回空列表
	memories, err := e.extractWithLLM(ctx, prompt, userID)
	if err != nil {
		logx.WarnfContext(ctx, "LLM memory extraction failed: %v, skipping", err)
		return []*MemoryItem{}, nil // 降级：返回空列表，不阻塞主流程
	}

	if len(memories) > 0 {
		logx.InfofContext(ctx, "extracted %d memories using LLM", len(memories))
		for _, m := range memories {
			logx.DebugfContext(ctx, "  - [%s] %s (tags: %v)", m.Type, m.Content, m.Tags)
		}
	}

	return memories, nil
}

// extractWithLLM 使用 LLM 提取记忆（带 panic 保护）
func (e *LLMExtractor) extractWithLLM(ctx context.Context, prompt string, userID string) (memories []*MemoryItem, err error) {
	defer gox.Recover()

	// 调用 LLM 生成提取结果
	response, err := e.config.ChatModel.Generate(ctx, []*schema.Message{
		{
			Role:    schema.System,
			Content: prompt,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("LLM generate: %w", err)
	}

	// 解析LLM返回结果
	memories, err = e.parseExtractionResult(response.Content, userID)
	if err != nil {
		return nil, fmt.Errorf("parse LLM response: %w", err)
	}

	// 过滤低置信度的记忆
	filteredMemories := e.filterByConfidence(memories)
	return filteredMemories, nil
}

// buildExtractionPrompt 构建提取提示词
func (e *LLMExtractor) buildExtractionPrompt(messages []adk.Message) string {
	var sb strings.Builder

	sb.WriteString("你是一个专业的用户信息提取助手。请从以下对话中提取用户的重要信息。\n\n")
	sb.WriteString("## 提取目标\n")
	sb.WriteString("识别并提取以下类型的信息：\n")
	sb.WriteString("1. **用户偏好**: 编辑器、编程语言、工具、工作习惯等\n")
	sb.WriteString("2. **事实信息**: 服务器IP、数据库配置、API端点、文件路径等\n")
	sb.WriteString("3. **工作环境**: 当前目录、项目结构、技术栈等\n")
	sb.WriteString("4. **任务状态**: 正在进行的操作、待办事项等\n\n")

	sb.WriteString("## 提取规则\n")
	sb.WriteString("- 只提取明确陈述的信息，不要推测\n")
	sb.WriteString("- 忽略寒暄、问候等无关内容\n")
	sb.WriteString("- 如果同一信息多次出现，选择最新或最详细的版本\n")
	sb.WriteString("- 置信度评分标准：\n")
	sb.WriteString("  * 0.9-1.0: 用户明确陈述的事实\n")
	sb.WriteString("  * 0.7-0.9: 用户暗示或间接提到的信息\n")
	sb.WriteString("  * <0.7: 不确定或模糊的信息（不应提取）\n\n")

	sb.WriteString("## 输出格式\n")
	sb.WriteString("必须以JSON数组格式返回，每个元素包含：\n")
	sb.WriteString("- `type`: 记忆类型（preference/fact/skill/experience）\n")
	sb.WriteString("- `content`: 记忆内容（使用自然语言完整描述，便于后续语义检索）\n")
	sb.WriteString("- `tags`: 标签数组（3-5个关键词，用于分类和快速过滤）\n")
	sb.WriteString("- `confidence`: 置信度（0-1之间的小数，仅用于内部过滤）\n")
	sb.WriteString("- `reason`: 提取原因（可选，简要说明为什么提取这条记忆）\n\n")

	sb.WriteString("## 示例\n")
	sb.WriteString("用户说：\"我喜欢用 vim 编辑配置文件，服务器是 192.168.1.100\"\n")
	sb.WriteString("返回：\n")
	sb.WriteString(`[
  {"type": "preference", "content": "用户偏好使用 vim 作为配置文件编辑器", "tags": ["editor", "vim", "preference"], "confidence": 0.95, "reason": "用户明确表达偏好"},
  {"type": "fact", "content": "用户的服务器IP地址为 192.168.1.100", "tags": ["server", "ip", "infrastructure"], "confidence": 0.9, "reason": "用户提供具体服务器地址"}
]` + "\n\n")

	sb.WriteString("## 注意事项\n")
	sb.WriteString("- 不要提取敏感信息（密码、密钥、token等）\n")
	sb.WriteString("- content 字段应使用完整的自然语言描述，而非简化的键值对\n")
	sb.WriteString("- tags 应使用英文小写，便于统一检索\n")
	sb.WriteString(fmt.Sprintf("- 最多提取 %d 条最重要的记忆\n", e.config.MaxItemsPerCall))
	sb.WriteString("- 如果没有值得提取的信息，返回空数组 []\n\n")

	sb.WriteString("## 对话内容\n")
	for i, msg := range messages {
		if i > 0 {
			sb.WriteString("\n")
		}

		// 根据消息角色格式化
		roleStr := strings.ToUpper(string(msg.Role))
		content := msg.Content

		// 如果有工具调用，也记录
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				content += fmt.Sprintf(" [调用工具: %s]", tc.Function.Name)
			}
		}

		sb.WriteString(fmt.Sprintf("%s: %s", roleStr, content))
	}

	return sb.String()
}

// parseExtractionResult 解析LLM返回的结果
func (e *LLMExtractor) parseExtractionResult(response string, userID string) ([]*MemoryItem, error) {
	results, err := utils.ParseLLMJSON[[]MemoryExtractionResult](response)
	if err != nil {
		return nil, fmt.Errorf("parse LLM JSON response: %w", err)
	}

	// 如果解析成功但结果为空，返回空切片而不是错误
	if results == nil || len(*results) == 0 {
		return []*MemoryItem{}, nil
	}

	// 转换为 MemoryItem
	var memories []*MemoryItem
	for _, result := range *results {
		// 验证必要字段
		if result.Type == "" || result.Content == "" {
			continue
		}

		// 验证置信度范围
		if result.Confidence < 0 || result.Confidence > 1 {
			logx.Warnf("invalid confidence value: %.2f, skipping", result.Confidence)
			continue
		}

		item := CreateMemory(userID, result.Type, result.Content, result.Tags)

		// 添加元数据
		if result.Reason != "" {
			item.Metadata["extraction_reason"] = result.Reason
		}
		item.Metadata["extracted_by"] = "llm"

		memories = append(memories, item)
	}

	return memories, nil
}

// filterByConfidence 根据置信度过滤记忆
func (e *LLMExtractor) filterByConfidence(memories []*MemoryItem) []*MemoryItem {
	// 注意：当前 MemoryItem 不再存储 Confidence，此方法保留用于未来扩展
	// 如果需要基于置信度过滤，应在提取阶段完成
	return memories
}

package memoryext

import (
	"context"

	"mayfly-go/internal/ai/agent/contributor"
	"mayfly-go/internal/ai/memory"
	"mayfly-go/internal/ai/session"
	"mayfly-go/pkg/eventbus"
	"mayfly-go/pkg/logx"

	"github.com/cloudwego/eino/components/model"
)

// MemoryExtractionExtension 长期记忆提取扩展（装配期激活型，ChannelService）
//
// 对齐 tokhub-ext-memory 的收尾提取：
//   - 注入 LLM Extractor：摘要完成后由事件异步触发长期记忆提取
//   - 订阅会话摘要完成事件：session.Manager 完成自动摘要后经事件总线
//     通知本扩展，异步提取并保存长期记忆（使用固定 subId 避免重复注册）
//
// 替代原副作用装配（SetupMemoryExtraction）：纳入统一装配链路后，可经配置
// 按 Id（memory_extraction）裁剪，激活发生在 WithFilter 裁剪之后；
// Activate 幂等（固定 subId 覆盖注册），装配重试安全。
type MemoryExtractionExtension struct {
	memoryManager  *memory.Manager
	sessionManager *session.Manager
	chatModel      model.ToolCallingChatModel
}

// NewExtractionExtension 创建记忆提取扩展（memoryManager 为 nil 时返回 nil，
// 未启用记忆能力，宿主跳过注册）
func NewExtractionExtension(memoryManager *memory.Manager, sessionManager *session.Manager, chatModel model.ToolCallingChatModel) *MemoryExtractionExtension {
	if memoryManager == nil {
		return nil
	}
	return &MemoryExtractionExtension{
		memoryManager:  memoryManager,
		sessionManager: sessionManager,
		chatModel:      chatModel,
	}
}

var _ contributor.Activatable = (*MemoryExtractionExtension)(nil)

func (e *MemoryExtractionExtension) Id() string { return "memory_extraction" }

// Activate 装配期激活：注入 LLM Extractor + 订阅会话摘要完成事件（幂等）
func (e *MemoryExtractionExtension) Activate(_ context.Context) error {
	// 注入 LLM Extractor（无 ChatModel 时提取能力降级为不可用）
	if e.chatModel != nil {
		extractor := memory.NewLLMExtractor()
		extractor.WithConfig(&memory.LLMExtractorConfig{
			Enabled:       true,
			MinConfidence: 0.7,
			ChatModel:     e.chatModel,
		})
		e.memoryManager.WithExtractor(extractor)
		logx.Info("LLM memory extractor auto-configured")
	} else {
		logx.Warn("ChatModel not provided, memory extraction will be disabled")
	}

	if e.sessionManager == nil {
		return nil
	}

	// 订阅会话摘要完成事件：事件总线解耦 session 摘要与记忆提取，
	// session 包不感知下游消费者
	session.EventBus.SubscribeAsync(session.EventTopicSummarized, "AgentMemoryExtractor", func(ctx context.Context, event *eventbus.Event[any]) error {
		evt, ok := event.Val.(*session.SummarizedEvent)
		if !ok || evt.UserId == "" {
			return nil
		}

		// 获取当前会话历史消息用于记忆提取
		history, err := e.sessionManager.GetHistory(ctx, evt.SessionKey)
		if err != nil {
			logx.WarnfContext(ctx, "[ext_memory] get history for memory extraction failed: %v", err)
			return nil // 不阻塞事件总线
		}

		if err := e.memoryManager.ExtractAndSave(ctx, &memory.ExtractMemoryReq{
			UserId: evt.UserId,
			Msgs:   history,
		}); err != nil {
			logx.ErrorfContext(ctx, "[ext_memory] auto extract memories error: %v", err)
		}
		return nil
	}, false)
	return nil
}

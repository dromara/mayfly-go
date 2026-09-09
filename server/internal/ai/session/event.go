package session

import "mayfly-go/pkg/eventbus"

// EventBus 会话模块的局部事件总线，用于解耦会话生命周期事件
// 外部模块（如记忆提取）可通过订阅此总线响应会话状态变化，无需直接依赖 session 包内部实现
var EventBus eventbus.Bus[any] = eventbus.New[any]()

const (
	// EventTopicSummarized 会话完成自动摘要后触发
	// 事件值类型为 *SummarizedEvent
	EventTopicSummarized = "session:summarized"

	// EventTopicInterruptResume 中断恢复时触发
	// 事件值类型为 *tools.InterruptResume
	EventTopicInterruptResume = "session:interrupt-resume"
)

// SummarizedEvent 会话摘要完成事件
// 由 session.Manager 在 summarizeSession 成功后发布
// 订阅方可据此执行长期记忆提取、压缩事件项落库、日志记录、告警等后续操作
type SummarizedEvent struct {
	UserId     string // 用户ID（取决于Meta是否设置了UserId，可能为空）
	SessionKey string // 会话标识
	Summary    string // 生成的摘要文本
	Skip       int    // 跳过的消息数
	Count      int    // 消息总数

	// 压缩统计（ContextCompaction，供订阅方落库压缩事件项）
	OriginalTokens         int // 压缩前被摘要内容的估算 token 数
	CompressedTokens       int // 压缩后摘要文本的估算 token 数
	CompressedMessageCount int // 被压缩的消息条数
}

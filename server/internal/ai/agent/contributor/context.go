package contributor

import (
	"context"
)

// PromptFragment 提示词片段（ContextContributor 产出）
//
// 片段按 Source 作为 section key 拼装到 preamble（system 消息），
// 多个贡献者必须使用不同的 Source 值以避免混淆。
type PromptFragment struct {
	// Source 片段唯一标识（section key，用于诊断与去重）
	Source string
	// Content 片段正文
	Content string
	// Retained 预算裁剪时是否保留（如权限/安全约束类片段应声明为 true）
	Retained bool
	// Priority 优先级，数值越大越优先保留
	Priority int
}

// PriorityContributor 可选接口：声明贡献片段的默认优先级（数值越大越优先保留）
type PriorityContributor interface {
	Priority() int
}

// RetainedContributor 可选接口：声明贡献片段预算裁剪时是否默认保留
type RetainedContributor interface {
	Retained() bool
}

// TurnInput 轮次输入上下文（对齐 tokhub TurnInputEnvironment + user_input）
//
// 仅携带 ContextContributor 实际消费的轮次运行时信息；
// 新增字段须以真实消费方为前提（避免契约膨胀）。
type TurnInput struct {
	// UserId 当前登录用户 ID（未登录为空；按用户加载数据，如记忆归属）
	UserId string
	// UserText 本轮用户输入文本（用于 $skill-code 显式提及等场景）
	UserText string
}

// ContextContributor 上下文贡献者（统一 Prompt 注入通道）
//
// 在每轮对话构建 prompt 时被调用，返回需要注入的片段列表。
// 实现应无状态或自行保证并发安全（每轮构建都会调用）。
//
// 对齐 tokhub ContextContributor::contribute_turn_context。
type ContextContributor interface {
	Contributor
	// ContributeTurnContext 贡献轮次级上下文片段，返回空切片表示本轮不注入
	ContributeTurnContext(ctx context.Context, in *TurnInput) ([]PromptFragment, error)
}

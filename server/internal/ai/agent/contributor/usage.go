package contributor

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

// TokenUsageInput 单轮 token 用量输入（turn 粒度，轮次结束时回调一次）
type TokenUsageInput struct {
	SessionKey string
	TurnId     string
	UserId     string
	// Usage 本轮各次模型调用的 token 用量累计（模型未上报时为 nil）
	Usage *schema.TokenUsage
	// Cause 轮次结束原因（用量统计与结束原因同源下发）
	Cause TurnEndCause
}

// TokenUsageContributor token 用量回调贡献者（唯一出口，对齐 tokhub TokenUsageContributor）
//
// Agent 在轮次结束时（含异常终止/中断挂起）回调一次；计量、计费、统计类
// 扩展经此通道接入，宿主无需感知具体扩展。
// fail-open：单个贡献者失败记日志跳过，不阻断主流程。
type TokenUsageContributor interface {
	Contributor
	// OnTokenUsage 轮次结束时回调（每轮一次）
	OnTokenUsage(ctx context.Context, in *TokenUsageInput) error
}

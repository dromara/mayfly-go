package agent

import (
	"context"
	"errors"

	aiconfig "mayfly-go/internal/ai/config"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// buildModelRetryConfig 将模型配置的重试策略转换为 adk 模型重试配置
// （v0.9 Model Retry 能力，AgenticMessage 路径 Typed 实例化）
//
// 仅对模型调用失败（err != nil）重试，成功输出直接接受；主动取消/超时
// （context.Canceled / DeadlineExceeded）不重试——保持「用户停止」的即时
// 中止语义，避免停止后仍发起后续请求。退避使用 adk 内置指数退避 + 抖动。
func buildModelRetryConfig(rc *aiconfig.ModelRetryConfig) *adk.TypedModelRetryConfig[*schema.AgenticMessage] {
	if rc == nil || rc.MaxRetries <= 0 {
		return nil
	}
	return &adk.TypedModelRetryConfig[*schema.AgenticMessage]{
		MaxRetries: rc.MaxRetries,
		ShouldRetry: func(ctx context.Context, retryCtx *adk.TypedRetryContext[*schema.AgenticMessage]) *adk.TypedRetryDecision[*schema.AgenticMessage] {
			// 模型成功返回输出（含空流）：接受，不重试
			if retryCtx.Err == nil {
				return nil
			}
			// 主动取消 / 超时：保持中止语义，不重试
			if errors.Is(retryCtx.Err, context.Canceled) || errors.Is(retryCtx.Err, context.DeadlineExceeded) {
				return nil
			}
			// 其余模型调用失败（网络抖动、429/5xx、流中断等）：重试
			return &adk.TypedRetryDecision[*schema.AgenticMessage]{Retry: true}
		},
	}
}

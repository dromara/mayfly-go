package agent

import (
	"context"
	"errors"
	"fmt"
	"mayfly-go/internal/ai/config"
	"mayfly-go/internal/ai/protocol"
	"mayfly-go/pkg/logx"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// buildModelFailoverConfig 将模型配置的故障转移策略转换为 adk 模型故障转移配置
// （v0.9 Model Failover 能力，AgenticMessage 路径 Typed 实例化）
//
// 与 Model Retry 的组合语义：主模型先经 Retry 重试，重试耗尽后（LastErr 为
// RetryExhaustedError）才触发转移，按序选用 fallbacks[attempt-1]；中断错误 /
// 流取消（ErrStreamCanceled）/ ctx 已取消由 eino 侧直接排除，此处仅需排除
// 主动取消与超时 —— 保持「用户停止即中止」语义，不向备用模型转移。
//
// 备用模型经 protocol.GetChatModel 创建（按完整配置缓存，进程内复用实例）；
// 单个备用模型创建失败视为该次转移失败，继续向后转移或以错误收尾。
func buildModelFailoverConfig(fc *config.ModelFailoverConfig) *adk.ModelFailoverConfig[*schema.AgenticMessage] {
	if fc == nil || len(fc.Fallbacks) == 0 {
		return nil
	}
	// maxFailovers 未配置（0）时默认全部 fallbacks 可用；超出清单数量时收口
	maxFailovers := fc.MaxFailovers
	if maxFailovers <= 0 || maxFailovers > len(fc.Fallbacks) {
		maxFailovers = len(fc.Fallbacks)
	}
	return &adk.ModelFailoverConfig[*schema.AgenticMessage]{
		MaxRetries: uint(maxFailovers),
		ShouldFailover: func(ctx context.Context, _ *schema.AgenticMessage, outputErr error) bool {
			if outputErr == nil {
				return false
			}
			// 主动取消 / 超时：保持中止语义，不转移
			if errors.Is(outputErr, context.Canceled) || errors.Is(outputErr, context.DeadlineExceeded) {
				return false
			}
			return true
		},
		GetFailoverModel: func(ctx context.Context, failoverCtx *adk.FailoverContext[*schema.AgenticMessage]) (model.BaseModel[*schema.AgenticMessage], []*schema.AgenticMessage, error) {
			idx := int(failoverCtx.FailoverAttempt) - 1
			if idx < 0 || idx >= len(fc.Fallbacks) {
				return nil, nil, fmt.Errorf("failover attempt %d exceeds configured fallbacks (%d)",
					failoverCtx.FailoverAttempt, len(fc.Fallbacks))
			}
			fallback := fc.Fallbacks[idx]
			failoverModel, err := protocol.GetChatModel(ctx, fallback)
			if err != nil {
				return nil, nil, fmt.Errorf("create failover model %s: %w", fallback.Model, err)
			}
			logx.WarnfContext(ctx, "[agent] failover to model %s (attempt %d, lastErr=%v)",
				fallback.Model, failoverCtx.FailoverAttempt, failoverCtx.LastErr)
			return failoverModel, nil, nil
		},
	}
}

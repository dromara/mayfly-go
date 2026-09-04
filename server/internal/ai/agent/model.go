package agent

import (
	"context"
	"mayfly-go/internal/ai/config"
	"mayfly-go/internal/ai/protocol"

	"github.com/cloudwego/eino/components/model"
)

// GetChatModel 获取Chat模型
func GetChatModel(ctx context.Context) (model.AgenticModel, error) {
	return protocol.GetChatModel(ctx, config.GetModel())
}

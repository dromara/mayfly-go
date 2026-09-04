package protocol

import (
	"context"
	"mayfly-go/internal/ai/config"

	"github.com/cloudwego/eino/components/model"
)

// 定义响应格式常量
const (
	ResponseFormatJSON = "json_object"
	ResponseFormatText = "text"
)

const ProtocolOpenai = "openai"

type Protocol interface {

	// Name 返回协议标识，如 openai、azure 等
	Name() string

	// NewChatModel 创建 chat 模型（AgenticMessage 路径，eino v0.9 Typed API：
	// 返回以 *schema.AgenticMessage 实例化的模型，供 TypedChatModelAgent 消费）
	NewChatModel(ctx context.Context, modelConfig *config.ModelConfig) (model.AgenticModel, error)
}

package model

import (
	"context"
	"mayfly-go/internal/ai/config"

	"github.com/cloudwego/eino/components/model"
)

func init() {
	RegisterAIModel(new(Openai))
}

var (
	aiModelMap = map[string]AIModel{}
)

type AIModel interface {

	// SupportModel 支持的模型
	SupportModel() string

	// GetChatModel 获取聊天模型
	GetChatModel(ctx context.Context, aiConfig *config.AIModelConfig) (model.ToolCallingChatModel, error)
}

// RegisterAIModel 注册AI模型
func RegisterAIModel(aiModel AIModel) {
	aiModelMap[aiModel.SupportModel()] = aiModel
}

// GetAIModel 获取AI模型
func GetAIModel(name string) AIModel {
	return aiModelMap[name]
}

// GetAIModelByConfig 根据配置获取AI模型
func GetAIModelByConfig(aiConfig *config.AIModelConfig) AIModel {
	return GetAIModel(aiConfig.ModelType)
}

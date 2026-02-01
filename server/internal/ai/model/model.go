package model

import (
	"context"
	"errors"
	"fmt"
	"mayfly-go/internal/ai/config"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"

	"github.com/cloudwego/eino/components/model"
)

// 定义响应格式常量
const (
	ResponseFormatJSON = "json_object"
	ResponseFormatText = "text"
)

func init() {
	RegisterAIModel(new(Openai))
}

var (
	aiModelMap = map[string]AIModel{}
	chatModels = collx.SM[string,model.ToolCallingChatModel]{}
)

type AIModel interface {

	// SupportModel 支持的模型
	SupportModel() string

	// NewChatModel 创建chat模型
	NewChatModel(ctx context.Context, aiConfig *config.AIModelConfig) (model.ToolCallingChatModel, error)
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

// GetChatModel 获取Chat模型
func GetChatModel(ctx context.Context, aiConfig *config.AIModelConfig) (model.ToolCallingChatModel, error) {
	aiModel := GetAIModelByConfig(aiConfig)
	if aiModel == nil {
		return nil, errors.New("no supported AI model found")
	}

	cacheKey := generateCacheKey(aiConfig)
	if chatModel, ok := chatModels.Load(cacheKey); ok {
		logx.Debugf("ai model [%s/%s] - get chat model from cache", aiConfig.ModelType, aiConfig.Model)
		return chatModel, nil
	}

	// 删除已存在的缓存
	chatModels.Clear()

	chatModel, err := aiModel.NewChatModel(ctx, aiConfig)
	if err != nil {
		return nil, err
	}
	logx.Debugf("ai model [%s/%s] - new chat model", aiConfig.ModelType,aiConfig.Model)
	chatModels.Store(cacheKey, chatModel)
	return chatModel, nil
}

// generateCacheKey 生成基于 aiConfig 关键字段的缓存键
func generateCacheKey(config *config.AIModelConfig) string {
	return fmt.Sprintf("%s_%s_%s_%s_%d_%f", 
		config.ModelType,
		config.Model,

		config.BaseUrl,
		config.ApiKey,
		config.MaxTokens,
		config.Temperature,
	)
}
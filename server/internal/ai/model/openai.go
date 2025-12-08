package model

import (
	"context"
	"mayfly-go/internal/ai/config"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

type Openai struct {
}

func (o *Openai) SupportModel() string {
	return "openai"
}

func (o *Openai) GetChatModel(ctx context.Context, aiConfig *config.AIModelConfig) (model.ToolCallingChatModel, error) {
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL:     aiConfig.BaseUrl,
		Model:       aiConfig.Model,
		APIKey:      aiConfig.ApiKey,
		Timeout:     time.Duration(aiConfig.TimeOut) * time.Second,
		MaxTokens:   &aiConfig.MaxTokens,
		Temperature: &aiConfig.Temperature,
	})
}

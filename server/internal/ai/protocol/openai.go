package protocol

import (
	"context"
	"mayfly-go/internal/ai/config"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

type Openai struct {
}

func (o *Openai) Name() string {
	return ProtocolOpenai
}

func (o *Openai) NewChatModel(ctx context.Context, modelConfig *config.ModelConfig) (model.ToolCallingChatModel, error) {
	modelCfg := &openai.ChatModelConfig{
		BaseURL:     modelConfig.BaseUrl,
		Model:       modelConfig.GetModelSpec().Model,
		APIKey:      modelConfig.ApiKey,
		Timeout:     time.Duration(modelConfig.TimeOut) * time.Second,
		Temperature: &modelConfig.Temperature,
	}
	// maxTokens <= 0 时不传该参数，由服务端使用模型默认输出上限（对齐 tokhub：
	// agent 请求默认不带 max_tokens）——thinking 模型的 reasoning 计入该预算，
	// 传小了会导致回复被 finish_reason=length 截断（tool_call 尚未生成即中断）
	if modelConfig.MaxTokens > 0 {
		modelCfg.MaxTokens = &modelConfig.MaxTokens
	}
	// enable_thinking 仅透传给支持该参数的网关（OpenAI 官方对未知参数会 400，
	// config 层已按模型名兜底：仅 qwen 系列或显式配置时非 nil）；qwen3 系列默认
	// 开启思考，若遇工具调用不稳定（输出「正在执行 XX」却不发 tool_calls）
	// 可在系统配置中将 enableThinking 配置为 false 关闭思考
	if modelConfig.EnableThinking != nil {
		modelCfg.ExtraFields = map[string]any{"enable_thinking": *modelConfig.EnableThinking}
	}
	return openai.NewChatModel(ctx, modelCfg)
}

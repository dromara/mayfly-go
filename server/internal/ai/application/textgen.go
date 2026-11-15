package application

// textgen 端口适配器：把「当前生效模型」的一次性文本生成能力交给业务模块。
//
// 装配契约（两条都必须成立，否则表现为「管理员改了模型配置但功能仍走旧端点」）：
//  1. 每次 Complete 都经 agent.ModelProvider() 取当前生效模型，不在装配期固化实例；
//  2. 本适配器在 Init 期注册到中立包 ai/pkg/textgen，业务侧只依赖该中立包。

import (
	"context"
	"errors"
	"fmt"

	"mayfly-go/internal/ai/agent"
	aiconfig "mayfly-go/internal/ai/config"
	"mayfly-go/internal/ai/pkg/textgen"
	"mayfly-go/internal/ai/protocol"
	"mayfly-go/internal/ai/session"
	"mayfly-go/pkg/logx"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// textgenPort 文本生成端口实现
type textgenPort struct{}

var _ textgen.Port = (*textgenPort)(nil)

func (textgenPort) Available(ctx context.Context) (bool, error) {
	if _, err := agent.ModelProvider().ChatModel(ctx); err != nil {
		return false, err
	}
	return true, nil
}

// chatModelFor 取本次调用使用的模型：PreferFast 时在同源配置上关闭思考模式。
//
// 为什么可以关：SQL 草稿/选表/行内补全的产物是短结构化文本，思考模式主要增加
// 首字节延迟（真机单次 10–50s 里相当一部分耗在这里），对这类任务几乎不提升正确性。
// 安全性：只在配置**显式开启**思考时才降级（qwen 系列默认开、或管理员手工打开）；
// 未开启（OpenAI 官方等不认识该参数的网关）时原样返回当前模型，绝不主动传该参数。
func chatModelFor(ctx context.Context, preferFast bool) (model.AgenticModel, error) {
	if preferFast {
		if fast, ok := fastChatModel(ctx); ok {
			return fast, nil
		}
	}
	return agent.ModelProvider().ChatModel(ctx)
}

// fastModelConfig 判断是否需要构造「关闭思考」的同源配置；第二个返回值为 false 表示沿用默认模型。
//
// 判定条件只有一条：当前配置**显式开启**了思考。未配置（OpenAI 官方等不认识该参数的网关）
// 时绝不主动补这个参数——未知参数会被网关 400，优化反而变成故障。
func fastModelConfig(conf *aiconfig.ModelConfig) (*aiconfig.ModelConfig, bool) {
	if conf == nil || conf.Model == "" || conf.EnableThinking == nil || !*conf.EnableThinking {
		return nil, false
	}
	clone := *conf
	disabled := false
	clone.EnableThinking = &disabled
	return &clone, true
}

// fastChatModel 同源配置 + 关闭思考的模型实例（protocol 层按配置指纹缓存，重复取零成本）
func fastChatModel(ctx context.Context) (model.AgenticModel, bool) {
	clone, ok := fastModelConfig(aiconfig.GetModel())
	if !ok {
		return nil, false
	}
	fast, err := protocol.GetChatModel(ctx, clone)
	if err != nil {
		// 构造失败就回落默认模型：快速通道是优化项，不能成为新的失败点
		logx.WarnfContext(ctx, "textgen: build fast(no-thinking) model failed, fallback to default: %v", err)
		return nil, false
	}
	return fast, true
}

func (textgenPort) Complete(ctx context.Context, req *textgen.Request) (string, *textgen.Usage, error) {
	if req == nil {
		return "", nil, errors.New("textgen: nil request")
	}
	chatModel, err := chatModelFor(ctx, req.PreferFast)
	if err != nil {
		return "", nil, fmt.Errorf("textgen: current model unavailable: %w", err)
	}

	msgs := make([]*schema.AgenticMessage, 0, 2)
	if req.System != "" {
		msgs = append(msgs, schema.SystemAgenticMessage(req.System))
	}
	msgs = append(msgs, schema.UserAgenticMessage(req.User))

	// 调用期选项：温度与输出上限均由消费方给定，适配器不自作主张
	options := make([]model.Option, 0, 2)
	if req.Temperature != nil {
		options = append(options, model.WithTemperature(*req.Temperature))
	}
	if req.MaxTokens > 0 {
		options = append(options, model.WithMaxTokens(req.MaxTokens))
	}

	callCtx := ctx
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	resp, err := chatModel.Generate(callCtx, msgs, options...)
	if err != nil {
		return "", nil, err
	}
	return session.ExtractText(resp), usageOfAgenticMessage(resp), nil
}

// usageOfAgenticMessage 归一化取用量：eino 的 AgenticMessage 把 usage 放在
// ResponseMeta.TokenUsage，模型/网关未上报时为 nil（调用方按 0 处理）
func usageOfAgenticMessage(msg *schema.AgenticMessage) *textgen.Usage {
	if msg == nil || msg.ResponseMeta == nil || msg.ResponseMeta.TokenUsage == nil {
		return nil
	}
	usage := msg.ResponseMeta.TokenUsage
	return &textgen.Usage{
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
	}
}

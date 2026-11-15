package application

import (
	"testing"

	aiconfig "mayfly-go/internal/ai/config"
)

// 快速通道的判定只有一条：配置显式开启思考时才降级。
//
// 这条用例有牙：如果把 nil（未配置）也当成"要关思考"，会给不认识 enable_thinking
// 的网关塞一个未知参数，请求直接 400——优化变成故障。
func TestFastModelConfig(t *testing.T) {
	enabled := true
	disabled := false

	if _, ok := fastModelConfig(nil); ok {
		t.Fatal("nil config must not enter fast channel")
	}
	if _, ok := fastModelConfig(&aiconfig.ModelConfig{}); ok {
		t.Fatal("model not configured must not enter fast channel")
	}
	if _, ok := fastModelConfig(&aiconfig.ModelConfig{Model: "openai/gpt", EnableThinking: nil}); ok {
		t.Fatal("thinking not configured (e.g. official OpenAI) must not be touched")
	}
	if _, ok := fastModelConfig(&aiconfig.ModelConfig{Model: "openai/gpt", EnableThinking: &disabled}); ok {
		t.Fatal("thinking already disabled must reuse the default model")
	}

	conf := &aiconfig.ModelConfig{Model: "openai/qwen3", BaseUrl: "https://x/v1", ApiKey: "k", TimeOut: 60, EnableThinking: &enabled}
	clone, ok := fastModelConfig(conf)
	if !ok || clone == nil {
		t.Fatal("thinking enabled must produce a fast clone")
	}
	if clone.EnableThinking == nil || *clone.EnableThinking {
		t.Fatal("fast clone must disable thinking")
	}
	if clone.Model != conf.Model || clone.BaseUrl != conf.BaseUrl || clone.ApiKey != conf.ApiKey || clone.TimeOut != conf.TimeOut {
		t.Fatalf("fast clone must keep every other field identical: %+v", clone)
	}
	if conf.EnableThinking == nil || !*conf.EnableThinking {
		t.Fatal("original config must not be mutated")
	}
}

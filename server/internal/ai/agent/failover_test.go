package agent

import (
	"context"
	"errors"
	"testing"

	aiconfig "mayfly-go/internal/ai/config"
)

func TestBuildModelFailoverConfigDisabled(t *testing.T) {
	if cfg := buildModelFailoverConfig(nil); cfg != nil {
		t.Fatal("nil failover config should produce nil adk config")
	}
	if cfg := buildModelFailoverConfig(&aiconfig.ModelFailoverConfig{}); cfg != nil {
		t.Fatal("empty fallbacks should produce nil adk config")
	}
}

func TestBuildModelFailoverConfigMaxRetriesClamp(t *testing.T) {
	fc := &aiconfig.ModelFailoverConfig{
		// maxFailovers 超过 fallbacks 数量：收口到 fallbacks 数量
		MaxFailovers: 99,
		Fallbacks: []*aiconfig.ModelConfig{
			{Model: "openai/model-a"},
			{Model: "openai/model-b"},
		},
	}
	cfg := buildModelFailoverConfig(fc)
	if cfg == nil {
		t.Fatal("expected non-nil adk failover config")
	}
	if cfg.MaxRetries != 2 {
		t.Fatalf("MaxRetries = %d, want 2 (clamped to len(fallbacks))", cfg.MaxRetries)
	}

	// maxFailovers 未配置（0）：默认全部 fallbacks 可用
	fc.MaxFailovers = 0
	cfg = buildModelFailoverConfig(fc)
	if cfg.MaxRetries != 2 {
		t.Fatalf("MaxRetries = %d, want 2 (default to len(fallbacks))", cfg.MaxRetries)
	}
}

func TestBuildModelFailoverConfigShouldFailover(t *testing.T) {
	cfg := buildModelFailoverConfig(&aiconfig.ModelFailoverConfig{
		Fallbacks: []*aiconfig.ModelConfig{{Model: "openai/model-a"}},
	})
	if cfg == nil {
		t.Fatal("expected non-nil adk failover config")
	}
	ctx := context.Background()

	if cfg.ShouldFailover(ctx, nil, nil) {
		t.Fatal("nil error should not failover")
	}
	if !cfg.ShouldFailover(ctx, nil, errors.New("connection refused")) {
		t.Fatal("model error should failover")
	}
	if cfg.ShouldFailover(ctx, nil, context.Canceled) {
		t.Fatal("context canceled should not failover (保持停止即中止语义)")
	}
	if cfg.ShouldFailover(ctx, nil, context.DeadlineExceeded) {
		t.Fatal("deadline exceeded should not failover (保持停止即中止语义)")
	}
}

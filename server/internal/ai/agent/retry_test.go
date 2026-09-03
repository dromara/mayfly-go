package agent

import (
	"context"
	"errors"
	"testing"

	aiconfig "mayfly-go/internal/ai/config"

	"github.com/cloudwego/eino/adk"
)

// TestBuildModelRetryConfig_Disabled 未配置 / 非正数次数时不启用重试
func TestBuildModelRetryConfig_Disabled(t *testing.T) {
	if got := buildModelRetryConfig(nil); got != nil {
		t.Errorf("nil config should disable retry, got %v", got)
	}
	if got := buildModelRetryConfig(&aiconfig.ModelRetryConfig{MaxRetries: 0}); got != nil {
		t.Errorf("zero retries should disable retry, got %v", got)
	}
}

// TestBuildModelRetryConfig_Decisions 决策矩阵：成功接受 / 取消不重试 / 失败重试
func TestBuildModelRetryConfig_Decisions(t *testing.T) {
	cfg := buildModelRetryConfig(&aiconfig.ModelRetryConfig{MaxRetries: 2})
	if cfg == nil {
		t.Fatal("retry config should be enabled")
	}
	if cfg.MaxRetries != 2 {
		t.Errorf("max retries = %d, want 2", cfg.MaxRetries)
	}

	// 成功输出：接受，不重试
	if d := cfg.ShouldRetry(context.Background(), &adk.RetryContext{}); d != nil {
		t.Errorf("success output should be accepted (nil decision), got %v", d)
	}

	// 主动取消 / 超时：保持中止语义，不重试
	for _, err := range []error{context.Canceled, context.DeadlineExceeded} {
		if d := cfg.ShouldRetry(context.Background(), &adk.RetryContext{Err: err}); d != nil {
			t.Errorf("err %v should not retry, got %v", err, d)
		}
	}

	// 其余失败（网络抖动 / 429 / 5xx 等）：重试
	for _, err := range []error{errors.New("connection reset"), errors.New("rate limit 429")} {
		d := cfg.ShouldRetry(context.Background(), &adk.RetryContext{Err: err})
		if d == nil || !d.Retry {
			t.Errorf("err %v should retry, got %v", err, d)
		}
	}
}

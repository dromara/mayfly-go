package agent

import (
	"context"
	"errors"
	"testing"
)

// TestTurnEndCause 结束原因细分：中止 → 异常 → 中断挂起 → 正常完成
func TestTurnEndCause(t *testing.T) {
	runOptions := &runOptions{turnId: "turn-1"}

	// 正常完成
	if c := turnEndCause(context.Background(), runOptions, nil); c != "completed" {
		t.Errorf("expected completed, got %s", c)
	}

	// 异常终止
	if c := turnEndCause(context.Background(), runOptions, errors.New("boom")); c != "error" {
		t.Errorf("expected error, got %s", c)
	}

	// 中断挂起（事件流检测到 Interrupted 且无错误）
	runOptions.interrupted = true
	if c := turnEndCause(context.Background(), runOptions, nil); c != "interrupted" {
		t.Errorf("expected interrupted, got %s", c)
	}

	// 主动中止（ctx 取消优先于异常与中断）
	runOptions.interrupted = true
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if c := turnEndCause(ctx, runOptions, errors.New("context canceled")); c != "aborted" {
		t.Errorf("expected aborted, got %s", c)
	}
}

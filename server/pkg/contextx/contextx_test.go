package contextx

import (
	"context"
	"errors"
	"testing"

	"mayfly-go/pkg/model"
)

// TestDetachKeepsValuesDropsCancellation contextx.Detach 的两半语义都必须成立：
// 剥离取消信号（否则请求一返回，异步写入与对目标实例的执行会被掐成一半），
// 同时保留链路信息与登录账号（否则审计归因和日志串联又丢了）
func TestDetachKeepsValuesDropsCancellation(t *testing.T) {
	account := &model.LoginAccount{Id: 7, Username: "ops"}
	base := WithLoginAccount(WithTraceId(context.Background()), account)

	detached := Detach(base)

	if got := GetLoginAccount(detached); got == nil || got.Id != 7 || got.Username != "ops" {
		t.Errorf("detached ctx lost the login account: %+v", got)
	}
	if GetTraceId(detached) == "" {
		t.Error("detached ctx lost the trace id")
	}
	if detached.Done() != nil {
		t.Error("detached ctx must not carry a done channel")
	}
	if _, ok := detached.Deadline(); ok {
		t.Error("detached ctx must not carry a deadline")
	}
}

// TestDetachDoesNotAffectParent 派生不能反向影响父 ctx：请求 ctx 仍须可被取消
func TestDetachDoesNotAffectParent(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancelled = WithLoginAccount(cancelled, &model.LoginAccount{Id: 1, Username: "a"})

	detached := Detach(cancelled)
	cancel()

	if detached.Err() != nil {
		t.Errorf("detached ctx was cancelled: %v", detached.Err())
	}
	if !errors.Is(cancelled.Err(), context.Canceled) {
		t.Errorf("parent ctx error = %v, want context.Canceled", cancelled.Err())
	}
}

func TestWithTraceIdIsIdempotentAndUnique(t *testing.T) {
	ctx := WithTraceId(context.Background())
	first := GetTraceId(ctx)
	if first == "" {
		t.Fatal("trace id was not generated")
	}
	if again := GetTraceId(WithTraceId(ctx)); again != first {
		t.Errorf("WithTraceId overwrote an existing trace id: %q != %q", again, first)
	}
	if other := GetTraceId(WithTraceId(context.Background())); other == first {
		t.Error("two independent chains produced the same trace id")
	}
}

// TestGetLoginAccountWithoutValue 未设置时返回 nil 而不是 panic：调用方普遍按 nil 分支处理
func TestGetLoginAccountWithoutValue(t *testing.T) {
	if GetLoginAccount(context.Background()) != nil {
		t.Error("expected nil account for a bare context")
	}
	if GetTraceId(context.Background()) != "" {
		t.Error("expected empty trace id for a bare context")
	}
}

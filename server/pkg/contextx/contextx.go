package contextx

import (
	"context"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/stringx"
)

type CtxKey string

const (
	LoginAccountKey CtxKey = "loginAccount"
	TraceIdKey      CtxKey = "traceId"
)

func NewLoginAccount(la *model.LoginAccount) context.Context {
	return WithLoginAccount(context.Background(), la)
}

func WithLoginAccount(ctx context.Context, la *model.LoginAccount) context.Context {
	return context.WithValue(ctx, LoginAccountKey, la)
}

// 从context中获取登录账号信息
func GetLoginAccount(ctx context.Context) *model.LoginAccount {
	return ctx.Value(LoginAccountKey).(*model.LoginAccount)
}

func NewTraceId() context.Context {
	return WithTraceId(context.Background())
}

func WithTraceId(ctx context.Context) context.Context {
	return context.WithValue(ctx, TraceIdKey, stringx.Rand(16))
}

// 从context中获取traceId
func GetTraceId(ctx context.Context) string {
	return ctx.Value(TraceIdKey).(string)
}

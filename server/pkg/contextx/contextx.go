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

// GetLoginAccount 从context中获取登录账号信息，不存在返回nil
func GetLoginAccount(ctx context.Context) *model.LoginAccount {
	if la, ok := ctx.Value(LoginAccountKey).(*model.LoginAccount); ok {
		return la
	}
	return nil
}

/**   traceId   **/

// Detach 派生一个「脱离取消信号但保留上下文信息」的 ctx，用于请求结束后仍需完成的后台工作。
//
// 数据库访问会随 ctx 取消而中止（避免客户端已断开却仍在跑的僵尸事务），因此把请求 ctx 直接
// 带进 goroutine 或定时任务，会让这批写入在请求返回的瞬间被掐断。
//
// 也不能图省事改用 context.Background()：那会丢掉 traceId 与登录账号，写入的创建人/修改人
// 退化成系统账号、日志也无法与请求串联。WithoutCancel 保留 Values 只剥离取消信号，是这两者
// 之间唯一正确的选择
func Detach(ctx context.Context) context.Context {
	return context.WithoutCancel(ctx)
}

// WithTraceId 生成traceId并放置于context中, 如果已存在则不覆盖
func WithTraceId(ctx context.Context) context.Context {
	if GetTraceId(ctx) != "" {
		return ctx
	}
	return context.WithValue(ctx, TraceIdKey, stringx.RandByChars(16, stringx.Nums+stringx.LowerChars))
}

// 从context中获取traceId
func GetTraceId(ctx context.Context) string {
	if val, ok := ctx.Value(TraceIdKey).(string); ok {
		return val
	}
	return ""
}

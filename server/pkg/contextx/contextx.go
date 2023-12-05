package contextx

import (
	"context"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"

	"gorm.io/gorm"
)

type CtxKey string

const (
	LoginAccountKey CtxKey = "loginAccount"
	TraceIdKey      CtxKey = "traceId"
	DbKey           CtxKey = "db"
)

func NewLoginAccount(la *model.LoginAccount) context.Context {
	return WithLoginAccount(context.Background(), la)
}

func WithLoginAccount(ctx context.Context, la *model.LoginAccount) context.Context {
	return context.WithValue(ctx, LoginAccountKey, la)
}

// 从context中获取登录账号信息，不存在返回nil
func GetLoginAccount(ctx context.Context) *model.LoginAccount {
	if la, ok := ctx.Value(LoginAccountKey).(*model.LoginAccount); ok {
		return la
	}
	return nil
}

func NewTraceId() context.Context {
	return WithTraceId(context.Background())
}

func WithTraceId(ctx context.Context) context.Context {
	return context.WithValue(ctx, TraceIdKey, stringx.RandByChars(16, stringx.Nums+stringx.LowerChars))
}

// 从context中获取traceId
func GetTraceId(ctx context.Context) string {
	if val, ok := ctx.Value(TraceIdKey).(string); ok {
		return val
	}
	return ""
}

// 将事务db放置context中，使用stack保存。以便多个方法调用实现方法内部各自的事务操作
func WithDb(ctx context.Context, db *gorm.DB) context.Context {
	if dbStack, ok := ctx.Value(DbKey).(*collx.Stack[*gorm.DB]); ok {
		dbStack.Push(db)
		return ctx
	}
	dbStack := new(collx.Stack[*gorm.DB])
	dbStack.Push(db)

	return context.WithValue(ctx, DbKey, dbStack)
}

// 获取当前操作的栈顶事务数据库实例
func GetDb(ctx context.Context) *gorm.DB {
	if dbStack, ok := ctx.Value(DbKey).(*collx.Stack[*gorm.DB]); ok {
		return dbStack.Top()
	}
	return nil
}

func RmDb(ctx context.Context) *gorm.DB {
	if dbStack, ok := ctx.Value(DbKey).(*collx.Stack[*gorm.DB]); ok {
		return dbStack.Pop()
	}
	return nil
}

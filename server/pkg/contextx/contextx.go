package contextx

import (
	"context"
	"mayfly-go/pkg/model"
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

// Tx 事务上下文信息
type Tx struct {
	Count int
	DB    *gorm.DB
}

// WithTxDb 将事务db放置context中
func WithTxDb(ctx context.Context, db *gorm.DB) (context.Context, *Tx) {
	if tx := GetTx(ctx); tx != nil {
		return ctx, tx
	}

	tx := &Tx{Count: 1, DB: db}
	return context.WithValue(ctx, DbKey, tx), tx
}

// GetDb 获取ctx中的事务db
func GetDb(ctx context.Context) *gorm.DB {
	if tx := GetTx(ctx); tx != nil {
		return tx.DB
	}
	return nil
}

// GetTx 获取当前ctx事务
func GetTx(ctx context.Context) *Tx {
	if tx, ok := ctx.Value(DbKey).(*Tx); ok {
		return tx
	}
	return nil
}

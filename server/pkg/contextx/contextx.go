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

// Tx 事务上下文信息
type Tx struct {
	Count int
	DB    *gorm.DB
}

func (t *Tx) Rollback() {
	if t.Count == 0 {
		t.DB.Rollback()
	} else {

	}
}

// WithTxDb 将事务db放置context中，使用stack保存。以便多个方法调用实现方法内部各自的事务操作
func WithTxDb(ctx context.Context, db *gorm.DB) (context.Context, *Tx) {
	tx := &Tx{Count: 1, DB: db}
	if dbStack, ok := ctx.Value(DbKey).(*collx.Stack[*Tx]); ok {
		dbStack.Push(tx)
		return ctx, tx
	}
	dbStack := new(collx.Stack[*Tx])
	dbStack.Push(tx)

	return context.WithValue(ctx, DbKey, dbStack), tx
}

// GetDb 获取当前操作的栈顶事务数据库实例
func GetDb(ctx context.Context) *gorm.DB {
	if dbStack, ok := ctx.Value(DbKey).(*collx.Stack[*Tx]); ok {
		if tx := dbStack.Top(); tx != nil {
			return tx.DB
		}
	}
	return nil
}

// GetTx 获取当前操作的栈顶事务信息
func GetTx(ctx context.Context) *Tx {
	if dbStack, ok := ctx.Value(DbKey).(*collx.Stack[*Tx]); ok {
		if tx := dbStack.Top(); tx != nil {
			return tx
		}
	}
	return nil
}

// RmDb 删除数据库事务db
func RmDb(ctx context.Context) *Tx {
	if dbStack, ok := ctx.Value(DbKey).(*collx.Stack[*Tx]); ok {
		return dbStack.Pop()
	}
	return nil
}

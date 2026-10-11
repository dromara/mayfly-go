package gormx

import (
	"context"
	"database/sql"
	"errors"
	"mayfly-go/pkg/global"

	"gorm.io/gorm"
)

// txCtxKey 事务连接的 context 键。
//
// 用非导出结构体而非字符串作键，避免与其他包同名的字符串键相互覆盖
type txCtxKey struct{}

// DB 返回当前 ctx 应该使用的连接：ctx 里绑了事务就用事务，否则用全局连接池。
//
// 始终带上 ctx，取消与超时信号才能传到驱动层，链路信息也才能被日志与钩子读到。
// 这是「用哪条连接」的唯一决策点：业务与仓储层都不需要再关心事务对象
func DB(ctx context.Context) *gorm.DB {
	gdb := global.Db
	if tx := TxFromCtx(ctx); tx != nil {
		gdb = tx
	}
	return gdb.WithContext(ctx)
}

// WithTx 将事务db绑定至 ctx；ctx 已绑定事务时原样返回，使嵌套调用自动加入外层事务。
//
// 事务的传播只通过 ctx 完成，因此不再需要任何 `XxxWithDb` 形式的方法变体
func WithTx(ctx context.Context, tx *gorm.DB) context.Context {
	if TxFromCtx(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, txCtxKey{}, tx)
}

// TxFromCtx 获取 ctx 中绑定的事务连接，未绑定返回 nil
func TxFromCtx(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txCtxKey{}).(*gorm.DB); ok {
		return tx
	}
	return nil
}

// InTx 当前 ctx 是否已处于事务中
func InTx(ctx context.Context) bool {
	return TxFromCtx(ctx) != nil
}

// WithoutTx 返回脱离当前事务绑定的上下文，之后的读写回到连接池上的独立连接。
//
// 用于「事务提交之后」才执行的动作：那条事务连接已经不可再用，继续沿用 ctx 里的绑定去写入，
// 会直接撞 "sql: transaction has already been committed or rolled back"
func WithoutTx(ctx context.Context) context.Context {
	return context.WithValue(ctx, txCtxKey{}, nil)
}

// ErrNestedBegin 在已处于事务中的 ctx 上直接开启新事务时返回。
//
// 直接入口把外层事务对象交还给调用方是危险契约：嵌套拿到的其实是同一个事务对象，
// 调用方一句常规的 defer tx.Rollback() 就会把外层事务杀掉。需要事务传播语义时使用 TxWith，
// 它在嵌套时自动加入外层事务
var ErrNestedBegin = errors.New("gormx: Begin called inside an existing transaction; use TxWith for nesting")

// Begin 开启事务，返回绑定了该事务的 ctx 与事务对象。
//
// 调用方必须负责 Commit/Rollback；ctx 已绑定事务时拒绝执行（ErrNestedBegin）而非把外层
// 事务对象交出去——嵌套加入外层事务是 TxWith 的职责
func Begin(ctx context.Context, opts ...*sql.TxOptions) (context.Context, *gorm.DB, error) {
	if TxFromCtx(ctx) != nil {
		return ctx, nil, ErrNestedBegin
	}

	var tx *gorm.DB
	if len(opts) > 0 && opts[0] != nil {
		tx = global.Db.WithContext(ctx).Begin(opts[0])
	} else {
		tx = global.Db.WithContext(ctx).Begin()
	}
	if tx.Error != nil {
		return ctx, nil, tx.Error
	}
	return WithTx(ctx, tx), tx, nil
}

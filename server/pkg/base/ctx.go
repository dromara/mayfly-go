package base

import (
	"context"

	"gorm.io/gorm"
)

type CtxKey string

const (
	DbKey CtxKey = "db"
)

// Tx 事务上下文信息
type Tx struct {
	Count int
	DB    *gorm.DB
}

// NewCtxWithTxDb 将事务db放置context中
func NewCtxWithTxDb(ctx context.Context, db *gorm.DB) (context.Context, *Tx) {
	if tx := GetTxFromCtx(ctx); tx != nil {
		return ctx, tx
	}

	tx := &Tx{Count: 1, DB: db}
	return context.WithValue(ctx, DbKey, tx), tx
}

// GetDbFromCtx 获取ctx中的事务db
func GetDbFromCtx(ctx context.Context) *gorm.DB {
	if tx := GetTxFromCtx(ctx); tx != nil {
		return tx.DB
	}
	return nil
}

// GetTxFromCtx 获取当前ctx事务
func GetTxFromCtx(ctx context.Context) *Tx {
	if tx, ok := ctx.Value(DbKey).(*Tx); ok {
		return tx
	}
	return nil
}

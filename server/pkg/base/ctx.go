package base

import (
	"context"

	"gorm.io/gorm"
)

type CtxKey string

const (
	DbKey CtxKey = "db"
)

// NewCtxWithDb 将事务db放置context中，若已存在，则直接返回ctx
func NewCtxWithDb(ctx context.Context, db *gorm.DB) context.Context {
	if tx := GetDbFromCtx(ctx); tx != nil {
		return ctx
	}

	return context.WithValue(ctx, DbKey, db)
}

// GetDbFromCtx 获取ctx中的事务db
func GetDbFromCtx(ctx context.Context) *gorm.DB {
	if txdb, ok := ctx.Value(DbKey).(*gorm.DB); ok {
		return txdb
	}
	return nil
}

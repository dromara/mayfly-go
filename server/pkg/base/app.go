package base

import (
	"context"
	"fmt"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"

	"gorm.io/gorm"
)

// 基础application接口
type App[T model.ModelI] interface {

	// Insert 新增一个实体
	Insert(ctx context.Context, e T) error

	// BatchInsert 批量新增实体
	BatchInsert(ctx context.Context, models []T) error

	// UpdateById 根据实体id更新实体信息
	UpdateById(ctx context.Context, e T) error

	// UpdateByCond 更新满足条件的数据
	// @param values 需为模型结构体指针或map(更新零值等)
	// @param cond 可为*model.QueryCond也可以为普通查询model
	UpdateByCond(ctx context.Context, values any, cond any) error

	// DeleteById 根据实体主键删除实体
	DeleteById(ctx context.Context, id ...uint64) error

	// DeleteByCond 根据条件进行删除
	// @param cond 可为*model.QueryCond也可以为普通查询model
	DeleteByCond(ctx context.Context, cond any) error

	// Save 保存实体，实体IsCreate返回true则新增，否则更新
	Save(ctx context.Context, e T) error

	// GetById 根据实体id查询
	GetById(id uint64, cols ...string) (T, error)

	// GetByIds 根据实体id数组查询
	GetByIds(ids []uint64, cols ...string) ([]T, error)

	// GetByCond 根据实体条件查询实体信息(获取单个实体)
	// @param cond 可为*model.QueryCond也可以为普通查询model
	GetByCond(cond any) error

	// ListByCondToAny 根据条件查询数据映射至res
	// @param cond 可为*model.QueryCond也可以为普通查询model
	ListByCondToAny(cond any, res any) error

	// ListByCond 根据条件查询
	// @param cond 可为*model.QueryCond也可以为普通查询model
	ListByCond(cond any, cols ...string) ([]T, error)

	// PageByCondToAny 分页查询并绑定至指定toModels
	// @param cond 可为*model.QueryCond也可以为普通查询model
	PageByCondToAny(cond any, pageParam *model.PageParam, toModels any) (*model.PageResult[any], error)

	// PageByCond 根据指定条件分页查询
	// @param cond 可为*model.QueryCond也可以为普通查询model
	PageByCond(cond any, pageParam *model.PageParam, cols ...string) (*model.PageResult[[]T], error)

	// CountByCond 根据指定条件统计model表的数量
	// @param cond 可为*model.QueryCond也可以为普通查询model
	CountByCond(cond any) int64

	// Tx 执行事务操作
	Tx(ctx context.Context, funcs ...func(context.Context) error) (err error)
}

// 基础application接口实现
type AppImpl[T model.ModelI, R Repo[T]] struct {
	Repo R // repo接口
}

// 获取repo
func (ai *AppImpl[T, R]) GetRepo() R {
	return ai.Repo
}

// 新增一个实体 (单纯新增，不做其他业务逻辑处理)
func (ai *AppImpl[T, R]) Insert(ctx context.Context, e T) error {
	return ai.GetRepo().Insert(ctx, e)
}

// 批量新增实体 (单纯新增，不做其他业务逻辑处理)
func (ai *AppImpl[T, R]) BatchInsert(ctx context.Context, es []T) error {
	return ai.GetRepo().BatchInsert(ctx, es)
}

// 根据实体id更新实体信息 (单纯更新，不做其他业务逻辑处理)
func (ai *AppImpl[T, R]) UpdateById(ctx context.Context, e T) error {
	return ai.GetRepo().UpdateById(ctx, e)
}

// UpdateByCond 更新满足条件的数据
// @param values 需为模型结构体指针或map(更新零值等)
// @param cond 可为*model.QueryCond也可以为普通查询model
func (ai *AppImpl[T, R]) UpdateByCond(ctx context.Context, values any, cond any) error {
	return ai.GetRepo().UpdateByCond(ctx, values, cond)
}

// 保存实体，实体IsCreate返回true则新增，否则更新
func (ai *AppImpl[T, R]) Save(ctx context.Context, e T) error {
	return ai.GetRepo().Save(ctx, e)
}

// 保存实体，实体IsCreate返回true则新增，否则更新。
// 使用指定gorm db执行，主要用于事务执行
func (ai *AppImpl[T, R]) SaveWithDb(ctx context.Context, db *gorm.DB, e T) error {
	return ai.GetRepo().SaveWithDb(ctx, db, e)
}

// 根据实体主键删除实体 (单纯删除实体，不做其他业务逻辑处理)
func (ai *AppImpl[T, R]) DeleteById(ctx context.Context, id ...uint64) error {
	return ai.GetRepo().DeleteById(ctx, id...)
}

// 根据指定条件删除实体 (单纯删除实体，不做其他业务逻辑处理)
func (ai *AppImpl[T, R]) DeleteByCond(ctx context.Context, cond any) error {
	return ai.GetRepo().DeleteByCond(ctx, cond)
}

// 根据实体id查询
func (ai *AppImpl[T, R]) GetById(id uint64, cols ...string) (T, error) {
	return ai.GetRepo().GetById(id, cols...)
}

func (ai *AppImpl[T, R]) GetByIds(ids []uint64, cols ...string) ([]T, error) {
	return ai.GetRepo().GetByIds(ids, cols...)
}

// 根据实体条件查询实体信息
func (ai *AppImpl[T, R]) GetByCond(cond any) error {
	return ai.GetRepo().GetByCond(cond)
}

func (ai *AppImpl[T, R]) ListByCondToAny(cond any, res any) error {
	return ai.GetRepo().SelectByCondToAny(cond, res)
}

func (ai *AppImpl[T, R]) ListByCond(cond any, cols ...string) ([]T, error) {
	return ai.GetRepo().SelectByCond(cond, cols...)
}

// PageByCondToAny 分页查询
func (ai *AppImpl[T, R]) PageByCondToAny(cond any, pageParam *model.PageParam, toModels any) (*model.PageResult[any], error) {
	return ai.GetRepo().PageByCondToAny(cond, pageParam, toModels)
}

func (ai *AppImpl[T, R]) PageByCond(cond any, pageParam *model.PageParam, cols ...string) (*model.PageResult[[]T], error) {
	return ai.GetRepo().PageByCond(cond, pageParam, cols...)
}

// 根据指定条件统计model表的数量, cond为条件可以为map等
func (ai *AppImpl[T, R]) CountByCond(cond any) int64 {
	return ai.GetRepo().CountByCond(cond)
}

// Tx 执行事务操作
func (ai *AppImpl[T, R]) Tx(ctx context.Context, funcs ...func(context.Context) error) (err error) {
	tx := GetTxFromCtx(ctx)
	dbCtx := ctx
	var txDb *gorm.DB

	if tx == nil {
		txDb = global.Db.Begin()
		dbCtx, tx = NewCtxWithTxDb(ctx, txDb)
	} else {
		txDb = tx.DB
		tx.Count++
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Count = 0
			txDb.Rollback()
			err = fmt.Errorf("%v", r)
			return
		}

		tx.Count--
	}()

	for _, f := range funcs {
		err = f(dbCtx)
		if err != nil && tx.Count > 0 {
			tx.Count = 0
			txDb.Rollback()
			return
		}
	}

	if tx.Count == 1 {
		err = txDb.Commit().Error
	}
	return
}

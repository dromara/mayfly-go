package base

import (
	"context"
	"fmt"
	"mayfly-go/pkg/contextx"
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
	DeleteByCond(ctx context.Context, cond any) error

	// Save 保存实体，实体IsCreate返回true则新增，否则更新
	Save(ctx context.Context, e T) error

	// GetById 根据实体id查询
	GetById(e T, id uint64, cols ...string) (T, error)

	// GetByIds 根据实体id数组查询
	GetByIds(list any, ids []uint64, orderBy ...string) error

	// GetByCond 根据实体条件查询实体信息(获取单个实体)
	GetByCond(cond any) error

	// ListByCond 根据条件查询数据映射至res
	ListByCond(cond any, res any) error

	// PageByCond 分页查询
	PageByCond(cond any, pageParam *model.PageParam, toModels any) (*model.PageResult[any], error)

	// CountByCond 根据指定条件统计model表的数量
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
func (ai *AppImpl[T, R]) GetById(e T, id uint64, cols ...string) (T, error) {
	if err := ai.GetRepo().GetById(e, id, cols...); err != nil {
		return e, err
	}
	return e, nil
}

func (ai *AppImpl[T, R]) GetByIds(list any, ids []uint64, orderBy ...string) error {
	return ai.GetRepo().GetByIds(list, ids, orderBy...)
}

// 根据实体条件查询实体信息
func (ai *AppImpl[T, R]) GetByCond(cond any) error {
	return ai.GetRepo().GetByCond(cond)
}

func (ai *AppImpl[T, R]) ListByCond(cond any, res any) error {
	return ai.GetRepo().SelectByCond(cond, res)
}

// PageByCond 分页查询
func (ai *AppImpl[T, R]) PageByCond(cond any, pageParam *model.PageParam, toModels any) (*model.PageResult[any], error) {
	return ai.GetRepo().PageByCond(cond, pageParam, toModels)
}

// 根据指定条件统计model表的数量, cond为条件可以为map等
func (ai *AppImpl[T, R]) CountByCond(cond any) int64 {
	return ai.GetRepo().CountByCond(cond)
}

// 执行事务操作
func (ai *AppImpl[T, R]) Tx(ctx context.Context, funcs ...func(context.Context) error) (err error) {
	tx := global.Db.Begin()
	dbCtx := contextx.WithDb(ctx, tx)

	defer func() {
		// 移除当前已执行完成的的数据库事务实例
		contextx.RmDb(ctx)
		if r := recover(); r != nil {
			tx.Rollback()
			err = fmt.Errorf("%v", err)
		}
	}()

	for _, f := range funcs {
		err = f(dbCtx)
		if err != nil {
			tx.Rollback()
			return
		}
	}
	err = tx.Commit().Error
	return
}

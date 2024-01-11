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

	// 新增一个实体
	Insert(ctx context.Context, e T) error

	// 使用指定gorm db执行，主要用于事务执行
	InsertWithDb(ctx context.Context, db *gorm.DB, e T) error

	// 批量新增实体
	BatchInsert(ctx context.Context, models []T) error

	// 使用指定gorm db执行，主要用于事务执行
	BatchInsertWithDb(ctx context.Context, db *gorm.DB, es []T) error

	// 根据实体id更新实体信息
	UpdateById(ctx context.Context, e T) error

	// 使用指定gorm db执行，主要用于事务执行
	UpdateByIdWithDb(ctx context.Context, db *gorm.DB, e T) error

	// 根据实体主键删除实体
	DeleteById(ctx context.Context, id uint64) error

	// 使用指定gorm db执行，主要用于事务执行
	DeleteByIdWithDb(ctx context.Context, db *gorm.DB, id uint64) error

	// 根据实体条件，更新参数udpateFields指定字段
	Updates(ctx context.Context, cond any, udpateFields map[string]any) error

	// 保存实体，实体IsCreate返回true则新增，否则更新
	Save(ctx context.Context, e T) error

	// 保存实体，实体IsCreate返回true则新增，否则更新。
	// 使用指定gorm db执行，主要用于事务执行
	SaveWithDb(ctx context.Context, db *gorm.DB, e T) error

	// 根据实体条件删除实体
	DeleteByCond(ctx context.Context, cond any) error

	// 使用指定gorm db执行，主要用于事务执行
	DeleteByCondWithDb(ctx context.Context, db *gorm.DB, cond any) error

	// 根据实体id查询
	GetById(e T, id uint64, cols ...string) (T, error)

	GetByIdIn(list any, ids []uint64, orderBy ...string) error

	// 根据实体条件查询实体信息
	GetBy(condModel T, cols ...string) error

	// 根据条件查询数据映射至listModels
	ListByCond(cond any, listModels any, cols ...string) error

	// 获取满足model中不为空的字段值条件的所有数据.
	//
	// @param list为数组类型 如 var users *[]User，可指定为非model结构体
	// @param cond  条件
	ListByCondOrder(cond any, list any, order ...string) error

	// 根据指定条件统计model表的数量, cond为条件可以为map等
	CountByCond(cond any) int64

	// 执行事务操作
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

// 使用指定gorm db执行，主要用于事务执行
func (ai *AppImpl[T, R]) InsertWithDb(ctx context.Context, db *gorm.DB, e T) error {
	return ai.GetRepo().InsertWithDb(ctx, db, e)
}

// 批量新增实体 (单纯新增，不做其他业务逻辑处理)
func (ai *AppImpl[T, R]) BatchInsert(ctx context.Context, es []T) error {
	return ai.GetRepo().BatchInsert(ctx, es)
}

// 使用指定gorm db执行，主要用于事务执行
func (ai *AppImpl[T, R]) BatchInsertWithDb(ctx context.Context, db *gorm.DB, models []T) error {
	return ai.GetRepo().BatchInsertWithDb(ctx, db, models)
}

// 根据实体id更新实体信息 (单纯更新，不做其他业务逻辑处理)
func (ai *AppImpl[T, R]) UpdateById(ctx context.Context, e T) error {
	return ai.GetRepo().UpdateById(ctx, e)
}

// 使用指定gorm db执行，主要用于事务执行
func (ai *AppImpl[T, R]) UpdateByIdWithDb(ctx context.Context, db *gorm.DB, e T) error {
	return ai.GetRepo().UpdateByIdWithDb(ctx, db, e)
}

// 根据实体条件，更新参数udpateFields指定字段 (单纯更新，不做其他业务逻辑处理)
func (ai *AppImpl[T, R]) Updates(ctx context.Context, cond any, udpateFields map[string]any) error {
	return ai.GetRepo().Updates(cond, udpateFields)
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
func (ai *AppImpl[T, R]) DeleteById(ctx context.Context, id uint64) error {
	return ai.GetRepo().DeleteById(ctx, id)
}

func (ai *AppImpl[T, R]) DeleteByIdWithDb(ctx context.Context, db *gorm.DB, id uint64) error {
	return ai.GetRepo().DeleteByCondWithDb(ctx, db, id)
}

// 根据指定条件删除实体 (单纯删除实体，不做其他业务逻辑处理)
func (ai *AppImpl[T, R]) DeleteByCond(ctx context.Context, cond any) error {
	return ai.GetRepo().DeleteByCond(ctx, cond)
}

// 使用指定gorm db执行，主要用于事务执行
func (ai *AppImpl[T, R]) DeleteByCondWithDb(ctx context.Context, db *gorm.DB, cond any) error {
	return ai.GetRepo().DeleteByCondWithDb(ctx, db, cond)
}

// 根据实体id查询
func (ai *AppImpl[T, R]) GetById(e T, id uint64, cols ...string) (T, error) {
	if err := ai.GetRepo().GetById(e, id, cols...); err != nil {
		return e, err
	}
	return e, nil
}

func (ai *AppImpl[T, R]) GetByIdIn(list any, ids []uint64, orderBy ...string) error {
	return ai.GetRepo().GetByIdIn(list, ids, orderBy...)
}

// 根据实体条件查询实体信息
func (ai *AppImpl[T, R]) GetBy(condModel T, cols ...string) error {
	return ai.GetRepo().GetBy(condModel, cols...)
}

// 根据条件查询数据映射至listModels
func (ai *AppImpl[T, R]) ListByCond(cond any, listModels any, cols ...string) error {
	return ai.GetRepo().ListByCond(cond, listModels, cols...)
}

// 获取满足model中不为空的字段值条件的所有数据.
//
// @param list为数组类型 如 var users *[]User，可指定为非model结构体
// @param cond  条件
func (ai *AppImpl[T, R]) ListByCondOrder(cond any, list any, order ...string) error {
	return ai.GetRepo().ListByCondOrder(cond, list, order...)
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

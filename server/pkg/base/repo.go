package base

import (
	"context"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"

	"gorm.io/gorm"
)

// 基础repo接口
type Repo[T model.ModelI] interface {

	// GetModel 获取表的模型实例
	GetModel() T

	// 新增一个实体
	Insert(ctx context.Context, e T) error

	// 使用指定gorm db执行，主要用于事务执行
	InsertWithDb(ctx context.Context, db *gorm.DB, e T) error

	// 批量新增实体
	BatchInsert(ctx context.Context, models []T) error

	// 使用指定gorm db执行，主要用于事务执行
	BatchInsertWithDb(ctx context.Context, db *gorm.DB, models []T) error

	// 根据实体id更新实体信息
	UpdateById(ctx context.Context, e T, columns ...string) error

	// 使用指定gorm db执行，主要用于事务执行
	UpdateByIdWithDb(ctx context.Context, db *gorm.DB, e T, columns ...string) error

	// UpdateByWheres 更新满足wheres条件的数据
	// @param wheres key => "age > ?" value => 10等
	UpdateByWheres(ctx context.Context, e T, wheres collx.M, columns ...string) error

	// UpdateByWheresWithDb 使用指定gorm.Db更新满足wheres条件的数据
	// @param wheres key => "age > ?" value => 10等
	UpdateByWheresWithDb(ctx context.Context, db *gorm.DB, e T, wheres collx.M, columns ...string) error

	// 保存实体，实体IsCreate返回true则新增，否则更新
	Save(ctx context.Context, e T) error

	// 保存实体，实体IsCreate返回true则新增，否则更新。
	// 使用指定gorm db执行，主要用于事务执行
	SaveWithDb(ctx context.Context, db *gorm.DB, e T) error

	// 根据实体主键删除实体
	DeleteById(ctx context.Context, id uint64) error

	// 使用指定gorm db执行，主要用于事务执行
	DeleteByIdWithDb(ctx context.Context, db *gorm.DB, id uint64) error

	// 根据实体条件，更新参数udpateFields指定字段
	Updates(cond any, udpateFields map[string]any) error

	// 根据实体条件删除实体
	DeleteByCond(ctx context.Context, cond any) error

	// 使用指定gorm db执行，主要用于事务执行
	DeleteByCondWithDb(ctx context.Context, db *gorm.DB, cond any) error

	// DeleteByWheres 根据wheres条件进行删除
	// @param wheres key -> "age > ?" value -> 10等
	DeleteByWheres(ctx context.Context, wheres collx.M) error

	// DeleteByWheresWithDb 使用指定gorm.Db根据wheres条件进行删除
	// @param wheres key -> "age > ?" value -> 10等
	DeleteByWheresWithDb(ctx context.Context, db *gorm.DB, wheres collx.M) error

	// 根据实体id查询
	GetById(e T, id uint64, cols ...string) error

	// 根据实体id数组查询对应实体列表，并将响应结果映射至list
	GetByIdIn(list any, ids []uint64, orderBy ...string) error

	// 根据实体条件查询实体信息
	GetBy(cond T, cols ...string) error

	// 根据实体条件查询数据映射至listModels
	ListByCond(cond any, listModels any, cols ...string) error

	// 根据wheres条件进行过滤
	// @param wheres key -> "age > ?" value -> 10等
	ListByWheres(wheres collx.M, listModels any, cols ...string) error

	// PageQuery 分页查询
	PageQuery(cond any, pageParam *model.PageParam, toModels any) (*model.PageResult[any], error)

	// 获取满足model中不为空的字段值条件的所有数据.
	//
	// @param list为数组类型 如 var users *[]User，可指定为非model结构体
	// @param cond  条件
	ListByCondOrder(cond any, list any, order ...string) error

	// 根据指定条件统计model表的数量, cond为条件可以为map等
	CountByCond(cond any) int64
}

// 基础repo接口
type RepoImpl[T model.ModelI] struct {
	M T // 模型实例
}

func (br *RepoImpl[T]) Insert(ctx context.Context, e T) error {
	if db := contextx.GetDb(ctx); db != nil {
		return br.InsertWithDb(ctx, db, e)
	}
	return gormx.Insert(br.fillBaseInfo(ctx, e))
}

func (br *RepoImpl[T]) InsertWithDb(ctx context.Context, db *gorm.DB, e T) error {
	return gormx.InsertWithDb(db, br.fillBaseInfo(ctx, e))
}

func (br *RepoImpl[T]) BatchInsert(ctx context.Context, es []T) error {
	if db := contextx.GetDb(ctx); db != nil {
		return br.BatchInsertWithDb(ctx, db, es)
	}
	for _, e := range es {
		br.fillBaseInfo(ctx, e)
	}
	return gormx.BatchInsert[T](es)
}

// 使用指定gorm db执行，主要用于事务执行
func (br *RepoImpl[T]) BatchInsertWithDb(ctx context.Context, db *gorm.DB, es []T) error {
	for _, e := range es {
		br.fillBaseInfo(ctx, e)
	}
	return gormx.BatchInsertWithDb[T](db, es)
}

func (br *RepoImpl[T]) UpdateById(ctx context.Context, e T, columns ...string) error {
	if db := contextx.GetDb(ctx); db != nil {
		return br.UpdateByIdWithDb(ctx, db, e, columns...)
	}

	return gormx.UpdateById(br.fillBaseInfo(ctx, e), columns...)
}

func (br *RepoImpl[T]) UpdateByIdWithDb(ctx context.Context, db *gorm.DB, e T, columns ...string) error {
	return gormx.UpdateByIdWithDb(db, br.fillBaseInfo(ctx, e), columns...)
}

func (br *RepoImpl[T]) UpdateByWheres(ctx context.Context, e T, wheres collx.M, columns ...string) error {
	if db := contextx.GetDb(ctx); db != nil {
		return br.UpdateByWheresWithDb(ctx, db, e, wheres, columns...)
	}

	e = br.fillBaseInfo(ctx, e)
	// model的主键值需为空，否则会带上主键条件
	e.SetId(0)
	return gormx.UpdateByWheres(e, wheres)
}

func (br *RepoImpl[T]) UpdateByWheresWithDb(ctx context.Context, db *gorm.DB, e T, wheres collx.M, columns ...string) error {
	e = br.fillBaseInfo(ctx, e)
	// model的主键值需为空，否则会带上主键条件
	e.SetId(0)
	return gormx.UpdateByWheresWithDb(db, br.fillBaseInfo(ctx, e), wheres, columns...)
}

func (br *RepoImpl[T]) Updates(cond any, udpateFields map[string]any) error {
	return gormx.Updates(br.GetModel(), cond, udpateFields)
}

func (br *RepoImpl[T]) Save(ctx context.Context, e T) error {
	if e.IsCreate() {
		return br.Insert(ctx, e)
	}
	return br.UpdateById(ctx, e)
}

func (br *RepoImpl[T]) SaveWithDb(ctx context.Context, db *gorm.DB, e T) error {
	if e.IsCreate() {
		return br.InsertWithDb(ctx, db, e)
	}
	return br.UpdateByIdWithDb(ctx, db, e)
}

func (br *RepoImpl[T]) DeleteById(ctx context.Context, id uint64) error {
	if db := contextx.GetDb(ctx); db != nil {
		return br.DeleteByIdWithDb(ctx, db, id)
	}
	return gormx.DeleteById(br.GetModel(), id)
}

func (br *RepoImpl[T]) DeleteByIdWithDb(ctx context.Context, db *gorm.DB, id uint64) error {
	return gormx.DeleteByCondWithDb(db, br.GetModel(), id)
}

func (br *RepoImpl[T]) DeleteByCond(ctx context.Context, cond any) error {
	if db := contextx.GetDb(ctx); db != nil {
		return br.DeleteByCondWithDb(ctx, db, cond)
	}
	return gormx.DeleteByCond(br.GetModel(), cond)
}

func (br *RepoImpl[T]) DeleteByCondWithDb(ctx context.Context, db *gorm.DB, cond any) error {
	return gormx.DeleteByCondWithDb(db, br.GetModel(), cond)
}

func (br *RepoImpl[T]) DeleteByWheres(ctx context.Context, wheres collx.M) error {
	if db := contextx.GetDb(ctx); db != nil {
		return br.DeleteByWheresWithDb(ctx, db, wheres)
	}
	// model的主键值需为空，否则会带上主键条件
	e := br.GetModel()
	e.SetId(0)
	return gormx.DeleteByWheres(e, wheres)
}

func (br *RepoImpl[T]) DeleteByWheresWithDb(ctx context.Context, db *gorm.DB, wheres collx.M) error {
	// model的主键值需为空，否则会带上主键条件
	e := br.GetModel()
	e.SetId(0)
	return gormx.DeleteByWheresWithDb(db, e, wheres)
}

func (br *RepoImpl[T]) GetById(e T, id uint64, cols ...string) error {
	if err := gormx.GetById(e, id, cols...); err != nil {
		return err
	}
	return nil
}

func (br *RepoImpl[T]) GetByIdIn(list any, ids []uint64, orderBy ...string) error {
	return gormx.GetByIdIn(br.GetModel(), list, ids, orderBy...)
}

func (br *RepoImpl[T]) GetBy(cond T, cols ...string) error {
	return gormx.GetBy(cond, cols...)
}

func (br *RepoImpl[T]) ListByCond(cond any, listModels any, cols ...string) error {
	return gormx.ListByCond(br.GetModel(), cond, listModels, cols...)
}

func (br *RepoImpl[T]) ListByWheres(wheres collx.M, listModels any, cols ...string) error {
	return gormx.ListByWheres(br.GetModel(), wheres, listModels, cols...)
}

func (br *RepoImpl[T]) PageQuery(cond any, pageParam *model.PageParam, toModels any) (*model.PageResult[any], error) {
	qd := gormx.NewQuery(br.GetModel()).WithCondModel(cond)
	return gormx.PageQuery(qd, pageParam, toModels)
}

func (br *RepoImpl[T]) ListByCondOrder(cond any, list any, order ...string) error {
	return gormx.ListByCondOrder(br.GetModel(), cond, list, order...)
}

func (br *RepoImpl[T]) CountByCond(cond any) int64 {
	return gormx.CountByCond(br.GetModel(), cond)
}

// GetModel 获取表的模型实例
func (br *RepoImpl[T]) GetModel() T {
	return br.M
}

// 从上下文获取登录账号信息，并赋值至实体
func (br *RepoImpl[T]) fillBaseInfo(ctx context.Context, e T) T {
	// 默认使用数据库id策略, 若要改变则实体结构体自行覆盖FillBaseInfo方法。可参考 sys/entity.Resource
	e.FillBaseInfo(model.IdGenTypeNone, contextx.GetLoginAccount(ctx))
	return e
}

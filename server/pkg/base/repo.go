package base

import (
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/gormx"

	"gorm.io/gorm"
)

// 基础repo接口
type Repo[T any] interface {

	// 新增一个实体
	Insert(e T) error

	// 使用指定gorm db执行，主要用于事务执行
	InsertWithDb(db *gorm.DB, e T) error

	// 批量新增实体
	BatchInsert(models []T) error

	// 使用指定gorm db执行，主要用于事务执行
	BatchInsertWithDb(db *gorm.DB, es []T) error

	// 根据实体id更新实体信息
	UpdateById(e T) error

	// 使用指定gorm db执行，主要用于事务执行
	UpdateByIdWithDb(db *gorm.DB, e T) error

	// 根据实体主键删除实体
	DeleteById(id uint64) error

	// 使用指定gorm db执行，主要用于事务执行
	DeleteByIdWithDb(db *gorm.DB, id uint64) error

	// 根据实体条件，更新参数udpateFields指定字段
	Updates(cond any, udpateFields map[string]any) error

	// 根据实体条件删除实体
	DeleteByCond(cond any) error

	// 使用指定gorm db执行，主要用于事务执行
	DeleteByCondWithDb(db *gorm.DB, cond any) error

	// 根据实体id查询
	GetById(e T, id uint64, cols ...string) error

	// 根据实体id数组查询对应实体列表，并将响应结果映射至list
	GetByIdIn(list any, ids []uint64, orderBy ...string) error

	// 根据实体条件查询实体信息
	GetBy(cond T, cols ...string) error

	// 根据实体条件查询数据映射至listModels
	ListByCond(cond any, listModels any, cols ...string) error

	// 获取满足model中不为空的字段值条件的所有数据.
	//
	// @param list为数组类型 如 var users *[]User，可指定为非model结构体
	// @param cond  条件
	ListByCondOrder(cond any, list any, order ...string) error

	// 根据指定条件统计model表的数量, cond为条件可以为map等
	CountByCond(cond any) int64
}

// 基础repo接口
type RepoImpl[T any] struct {
	M any // 模型实例
}

func (br *RepoImpl[T]) Insert(e T) error {
	return gormx.Insert(e)
}

func (br *RepoImpl[T]) InsertWithDb(db *gorm.DB, e T) error {
	return gormx.InsertWithDb(db, e)
}

func (br *RepoImpl[T]) BatchInsert(es []T) error {
	return gormx.BatchInsert(es)
}

// 使用指定gorm db执行，主要用于事务执行
func (br *RepoImpl[T]) BatchInsertWithDb(db *gorm.DB, es []T) error {
	return gormx.BatchInsertWithDb(db, es)
}

func (br *RepoImpl[T]) UpdateById(e T) error {
	return gormx.UpdateById(e)
}

func (br *RepoImpl[T]) UpdateByIdWithDb(db *gorm.DB, e T) error {
	return gormx.UpdateByIdWithDb(db, e)
}

func (br *RepoImpl[T]) Updates(cond any, udpateFields map[string]any) error {
	return gormx.Updates(cond, udpateFields)
}

func (br *RepoImpl[T]) DeleteById(id uint64) error {
	return gormx.DeleteById(br.getModel(), id)
}

func (br *RepoImpl[T]) DeleteByIdWithDb(db *gorm.DB, id uint64) error {
	return gormx.DeleteByCondWithDb(db, br.getModel(), id)
}

func (br *RepoImpl[T]) DeleteByCond(cond any) error {
	return gormx.DeleteByCond(br.getModel(), cond)
}

func (br *RepoImpl[T]) DeleteByCondWithDb(db *gorm.DB, cond any) error {
	return gormx.DeleteByCondWithDb(db, br.getModel(), cond)
}

func (br *RepoImpl[T]) GetById(e T, id uint64, cols ...string) error {
	if err := gormx.GetById(e, id, cols...); err != nil {
		return err
	}
	return nil
}

func (br *RepoImpl[T]) GetByIdIn(list any, ids []uint64, orderBy ...string) error {
	return gormx.GetByIdIn(br.getModel(), list, ids, orderBy...)
}

func (br *RepoImpl[T]) GetBy(cond T, cols ...string) error {
	return gormx.GetBy(cond, cols...)
}

func (br *RepoImpl[T]) ListByCond(cond any, listModels any, cols ...string) error {
	return gormx.ListByCond(br.getModel(), cond, listModels, cols...)
}

func (br *RepoImpl[T]) ListByCondOrder(cond any, list any, order ...string) error {
	return gormx.ListByCondOrder(br.getModel(), cond, list, order...)
}

func (br *RepoImpl[T]) CountByCond(cond any) int64 {
	return gormx.CountByCond(br.getModel(), cond)
}

// 获取表的模型实例
func (br *RepoImpl[T]) getModel() any {
	biz.IsTrue(br.M != nil, "base.RepoImpl的M字段不能为空")
	return br.M
}

package base

import (
	"gorm.io/gorm"
)

// 基础application接口
type App[T any] interface {

	// 新增一个实体
	Insert(e T) error

	// 批量新增实体
	BatchInsert(models []T) error

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
}

// 基础application接口实现
type AppImpl[T any, R Repo[T]] struct {
	Repo R // repo接口
}

// 获取repo
func (ai *AppImpl[T, R]) GetRepo() R {
	return ai.Repo
}

// 新增一个实体 (单纯新增，不做其他业务逻辑处理)
func (ai *AppImpl[T, R]) Insert(e T) error {
	return ai.GetRepo().Insert(e)
}

// 批量新增实体 (单纯新增，不做其他业务逻辑处理)
func (ai *AppImpl[T, R]) BatchInsert(es []T) error {
	return ai.GetRepo().BatchInsert(es)
}

// 根据实体id更新实体信息 (单纯更新，不做其他业务逻辑处理)
func (ai *AppImpl[T, R]) UpdateById(e T) error {
	return ai.GetRepo().UpdateById(e)
}

// 使用指定gorm db执行，主要用于事务执行
func (ai *AppImpl[T, R]) UpdateByIdWithDb(db *gorm.DB, e T) error {
	return ai.GetRepo().UpdateByIdWithDb(db, e)
}

// 根据实体条件，更新参数udpateFields指定字段 (单纯更新，不做其他业务逻辑处理)
func (ai *AppImpl[T, R]) Updates(cond any, udpateFields map[string]any) error {
	return ai.GetRepo().Updates(cond, udpateFields)
}

// 根据实体主键删除实体 (单纯删除实体，不做其他业务逻辑处理)
func (ai *AppImpl[T, R]) DeleteById(id uint64) error {
	return ai.GetRepo().DeleteById(id)
}

func (ai *AppImpl[T, R]) DeleteByIdWithDb(db *gorm.DB, id uint64) error {
	return ai.GetRepo().DeleteByCondWithDb(db, id)
}

// 根据指定条件删除实体 (单纯删除实体，不做其他业务逻辑处理)
func (ai *AppImpl[T, R]) DeleteByCond(cond any) error {
	return ai.GetRepo().DeleteByCond(cond)
}

// 使用指定gorm db执行，主要用于事务执行
func (ai *AppImpl[T, R]) DeleteByCondWithDb(db *gorm.DB, cond any) error {
	return ai.GetRepo().DeleteByCondWithDb(db, cond)
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

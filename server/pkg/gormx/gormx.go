package gormx

import (
	"fmt"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
	"strings"
	"time"

	"gorm.io/gorm"
)

// 伪删除之未删除过滤条件
func UndeleteScope(db *gorm.DB) *gorm.DB {
	return db.Where(model.DeletedColumn, model.ModelUndeleted)
}

// 根据id获取实体对象。model需为指针类型（需要将查询出来的值赋值给model）
//
// 若error不为nil则为不存在该记录
// @param model  数据库映射实体模型
func GetById(model any, id uint64, cols ...string) error {
	return NewQuery(model).Eq("id", id).Undeleted().GenGdb().First(model).Error
}

// 根据id列表查询实体信息
// @param model  数据库映射实体模型
func GetByIdIn(model any, list any, ids []uint64, orderBy ...string) error {
	return NewQuery(model).In("id", ids).Undeleted().GenGdb().Find(list).Error
}

// 获取满足model中不为空的字段值条件的单个对象。model需为指针类型（需要将查询出来的值赋值给model）
//
// 若 error不为nil，则为不存在该记录
// @param model  数据库映射实体模型
func GetBy(model any, cols ...string) error {
	return global.Db.Select(cols).Where(model).Scopes(UndeleteScope).First(model).Error
}

// 根据model指定条件统计数量
func CountBy(model any) int64 {
	return CountByCond(model, model)
}

// 根据条件cond获取指定model表统计数量
func CountByCond(model any, cond any) int64 {
	var count int64
	NewQuery(model).WithCondModel(cond).Undeleted().GenGdb().Count(&count)
	return count
}

// 根据查询条件分页查询数据
// 若未指定查询列，则查询列以toModels字段为准
func PageQuery[T any](q *QueryCond, pageParam *model.PageParam, toModels T) (*model.PageResult[T], error) {
	q.Undeleted()
	gdb := q.GenGdb()
	var count int64
	err := gdb.Count(&count).Error
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return model.EmptyPageResult[T](), nil
	}

	page := pageParam.PageNum
	pageSize := pageParam.PageSize
	err = gdb.Limit(pageSize).Offset((page - 1) * pageSize).Find(toModels).Error
	if err != nil {
		return nil, err
	}
	return &model.PageResult[T]{Total: count, List: toModels}, nil
}

// 根据查询条件查询列表信息
func ListByQueryCond(q *QueryCond, list any) error {
	q.Undeleted()
	gdb := q.GenGdb()
	return gdb.Find(list).Error
}

// 获取满足model中不为空的字段值条件的所有数据.
//
// @param list为数组类型 如 var users *[]User，可指定为非model结构体，即只包含需要返回的字段结构体
func ListBy(model any, list any, cols ...string) error {
	return ListByCond(model, model, list, cols...)
}

// 获取满足cond中不为空的字段值条件的所有model表数据.
//
// @param list为数组类型 如 var users *[]User，可指定为非model结构体，即只包含需要返回的字段结构体
func ListByCond(model any, cond any, list any, cols ...string) error {
	return global.Db.Model(model).Select(cols).Where(cond).Scopes(UndeleteScope).Order("id desc").Find(list).Error
}

// 获取满足model中不为空的字段值条件的所有数据.
//
// @param list为数组类型 如 var users *[]User，可指定为非model结构体
// @param model  数据库映射实体模型
func ListByOrder(model any, list any, order ...string) error {
	return ListByCondOrder(model, model, list, order...)
}

// 获取满足cond中不为空的字段值条件的所有model表数据.
//
// @param list为数组类型 如 var users *[]User，可指定为非model结构体
// @param model  数据库映射实体模型
func ListByCondOrder(model any, cond any, list any, order ...string) error {
	var orderByStr string
	if order == nil {
		orderByStr = "id desc"
	} else {
		orderByStr = strings.Join(order, ",")
	}
	return global.Db.Model(model).Where(cond).Scopes(UndeleteScope).Order(orderByStr).Find(list).Error
}

func GetListBySql2Model(sql string, toEntity any, params ...any) error {
	return global.Db.Raw(sql, params...).Find(toEntity).Error
}

func ExecSql(sql string, params ...any) error {
	return global.Db.Exec(sql, params...).Error
}

// 插入model
// @param model  数据库映射实体模型
func Insert(model any) error {
	return InsertWithDb(global.Db, model)
}

// 使用指定gormDb插入model
func InsertWithDb(db *gorm.DB, model any) error {
	return db.Create(model).Error
}

// 批量插入
func BatchInsert[T any](models []T) error {
	return BatchInsertWithDb[T](global.Db, models)
}

// 批量插入
func BatchInsertWithDb[T any](db *gorm.DB, models []T) error {
	return db.CreateInBatches(models, len(models)).Error
}

// 根据id更新model，更新字段为model中不为空的值，即int类型不为0，ptr类型不为nil这类字段值
// @param model  数据库映射实体模型
func UpdateById(model any, columns ...string) error {
	return UpdateByIdWithDb(global.Db, model, columns...)
}

func UpdateByIdWithDb(db *gorm.DB, model any, columns ...string) error {
	return db.Model(model).Select(columns).Updates(model).Error
}

// 根据实体条件，更新参数udpateFields指定字段
func Updates(model any, condition any, updateFields map[string]any) error {
	return global.Db.Model(model).Where(condition).Updates(updateFields).Error
}

// 根据id删除model
// @param model  数据库映射实体模型
func DeleteById(model_ any, id uint64) error {
	return DeleteByIdWithDb(global.Db, model_, id)
}

// 根据id使用指定gromDb删除
func DeleteByIdWithDb(db *gorm.DB, model_ any, id uint64) error {
	return db.Model(model_).Where("id = ?", id).Updates(getDeleteColumnValue()).Error
}

// 根据model条件删除
// @param model  数据库映射实体模型
func DeleteBy(model_ any) error {
	return DeleteByCond(model_, model_)
}

// 根据cond条件删除指定model表数据
//
// @param model_  数据库映射实体模型
// @param cond  条件
func DeleteByCond(model_ any, cond any) error {
	return DeleteByCondWithDb(global.Db, model_, cond)
}

// 根据model条件删除
// @param model  数据库映射实体模型
func DeleteByWithDb(db *gorm.DB, model_ any) error {
	return DeleteByCondWithDb(db, model_, model_)
}

// 根据cond条件删除指定model表数据
//
// @param model  数据库映射实体模型
// @param cond 条件
func DeleteByCondWithDb(db *gorm.DB, model_ any, cond any) error {
	return db.Model(model_).Where(cond).Updates(getDeleteColumnValue()).Error
}

func getDeleteColumnValue() map[string]any {
	columnValue := make(map[string]any)
	columnValue[model.DeletedColumn] = model.ModelDeleted
	columnValue[model.DeleteTimeColumn] = time.Now()
	return columnValue
}

func Tx(funcs ...func(db *gorm.DB) error) (err error) {
	tx := global.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			err = fmt.Errorf("%v", r)
		}
	}()
	for _, f := range funcs {
		err = f(tx)
		if err != nil {
			tx.Rollback()
			return
		}
	}
	err = tx.Commit().Error
	return
}

package gormx

import (
	"fmt"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
	"time"

	"gorm.io/gorm"
)

// 伪删除之伪删除过滤条件
func UndeleteScope(db *gorm.DB) *gorm.DB {
	return db.Where(model.DeletedColumn, model.ModelUndeleted)
}

// 根据id获取实体对象。model需为指针类型（需要将查询出来的值赋值给model）
//
// 若error不为nil则为不存在该记录
// @param dbModel  数据库映射实体模型
func GetById(dbModel model.ModelI, id uint64, cols ...string) error {
	return NewQuery(dbModel, model.NewCond().Columns(cols...).Eq(model.IdColumn, id)).GenGdb().First(dbModel).Error
}

// 根据model获取单个实体对象
//
// 若 error不为nil，则为不存在该记录
// @param cond  查询条件
func GetByCond(dbModel model.ModelI, cond *model.QueryCond) error {
	return NewQuery(dbModel, cond).GenGdb().First(cond.GetDest()).Error
}

// 根据条件cond获取指定model表统计数量
func CountByCond(dbModel model.ModelI, cond *model.QueryCond) int64 {
	var count int64
	NewQuery(dbModel, cond).GenGdb().Count(&count)
	return count
}

// PageQuery 根据查询条件分页查询数据，若需要伪删除过滤，则自行过滤-调用q.Undeleted()
// 若未指定查询列，则查询列以toModels字段为准
func PageQuery[T any](q *Query, pageParam model.PageParam, toModels []T) (*model.PageResult[T], error) {
	gdb := q.GenGdb()
	var count int64
	err := gdb.Count(&count).Error
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return model.NewEmptyPageResult[T](), nil
	}

	page := pageParam.PageNum
	pageSize := pageParam.PageSize
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 100
	}
	if pageSize > 2000 {
		pageSize = 2000
	}
	err = gdb.Limit(pageSize).Offset((page - 1) * pageSize).Find(&toModels).Error
	if err != nil {
		return nil, err
	}
	return &model.PageResult[T]{Total: count, List: toModels}, nil
}

// PageByCond 根据指定查询条件分页查询数据
func PageByCond[T any](dbModel model.ModelI, cond *model.QueryCond, pageParam model.PageParam, toModels []T) (*model.PageResult[T], error) {
	return PageQuery(NewQuery(dbModel, cond), pageParam, toModels)
}

// SelectByCond 根据条件查询结果集
func SelectByCond(dbModel model.ModelI, cond *model.QueryCond) error {
	return NewQuery(dbModel, cond).GenGdb().Find(cond.GetDest()).Error
}

// SelectBySql 根据sql查询数据
func SelectBySql(sql string, toEntity any, params ...any) error {
	return global.Db.Raw(sql, params...).Scan(toEntity).Error
}

func ExecSql(sql string, params ...any) error {
	return global.Db.Exec(sql, params...).Error
}

// 插入model
// @param model  数据库映射实体模型
func Insert(model model.ModelI) error {
	return InsertWithDb(global.Db, model)
}

// 使用指定gormDb插入model
func InsertWithDb(db *gorm.DB, model model.ModelI) error {
	return db.Create(model).Error
}

// 批量插入
func BatchInsert[T model.ModelI](models []T) error {
	return BatchInsertWithDb[T](global.Db, models)
}

// 批量插入
func BatchInsertWithDb[T model.ModelI](db *gorm.DB, models []T) error {
	return db.CreateInBatches(models, len(models)).Error
}

// 根据id更新model，更新字段为model中不为空的值，即int类型不为0，ptr类型不为nil这类字段值
// @param model  数据库映射实体模型
func UpdateById(model model.ModelI, columns ...string) error {
	return UpdateByIdWithDb(global.Db, model, columns...)
}

func UpdateByIdWithDb(db *gorm.DB, model model.ModelI, columns ...string) error {
	return db.Model(model).Select(columns).Updates(model).Error
}

// UpdateByCond 使用默认global.Dd更新满足条件的数据(model的主键值需为空，否则会带上主键条件)
func UpdateByCond(dbModel model.ModelI, values any, cond *model.QueryCond) error {
	return UpdateByCondWithDb(global.Db, dbModel, values, cond)
}

// UpdateByCondWithDb 使用指定gorm.DB更新满足条件的数据(model的主键值需为空，否则会带上主键条件)
// @values values must be a struct or map.
func UpdateByCondWithDb(db *gorm.DB, dbModel model.ModelI, values any, cond *model.QueryCond) error {
	gormDb := db.Model(dbModel).Select(cond.GetSelectColumns())
	setGdbWhere(gormDb, cond)
	return gormDb.Updates(values).Error
}

// 根据id删除model
// @param model  数据库映射实体模型
func DeleteById(model_ model.ModelI, id ...uint64) error {
	return DeleteByIdWithDb(global.Db, model_, id...)
}

// 根据id使用指定gromDb删除
func DeleteByIdWithDb(db *gorm.DB, model_ model.ModelI, id ...uint64) error {
	return DeleteByCondWithDb(db, model_, model.NewCond().In(model.IdColumn, id))
}

// 根据cond条件删除指定model表数据
//
// @param dbModel  数据库映射实体模型
// @param cond  条件
func DeleteByCond(dbModel model.ModelI, cond *model.QueryCond) error {
	return DeleteByCondWithDb(global.Db, dbModel, cond)
}

// 根据cond条件删除指定model表数据
//
// @param dbModel  数据库映射实体模型
// @param cond 条件
func DeleteByCondWithDb(db *gorm.DB, dbModel model.ModelI, cond *model.QueryCond) error {
	gdb := setGdbWhere(db.Model(dbModel), cond)
	if !dbModel.LogicDelete() {
		return gdb.Delete(dbModel).Error
	}
	return gdb.Updates(getDeleteColumnValue()).Error
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

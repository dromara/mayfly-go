package base

import (
	"cmp"
	"context"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"reflect"
	"time"

	"gorm.io/gorm"
)

// 基础repo接口
type Repo[T model.ModelI] interface {

	// Insert 新增一个实体
	Insert(ctx context.Context, e T) error

	// InsertWithDb 使用指定gorm db执行，主要用于事务执行
	InsertWithDb(ctx context.Context, db *gorm.DB, e T) error

	// BatchInsert 批量新增实体
	BatchInsert(ctx context.Context, models []T) error

	// BatchInsertWithDb 使用指定gorm db执行，主要用于事务执行
	BatchInsertWithDb(ctx context.Context, db *gorm.DB, models []T) error

	// 根据实体id更新实体信息
	UpdateById(ctx context.Context, e T, columns ...string) error

	// 使用指定gorm db执行，主要用于事务执行
	UpdateByIdWithDb(ctx context.Context, db *gorm.DB, e T, columns ...string) error

	// UpdateByCond 更新满足条件的数据
	// @param values 需要模型结构体或map
	// @param cond 条件
	UpdateByCond(ctx context.Context, values any, cond any) error

	// UpdateByCondWithDb 更新满足条件的数据
	// @param values 需要模型结构体或map
	UpdateByCondWithDb(ctx context.Context, db *gorm.DB, values any, cond any) error

	// Save 保存实体，实体IsCreate返回true则新增，否则更新
	Save(ctx context.Context, e T) error

	// SaveWithDb 保存实体，实体IsCreate返回true则新增，否则更新。
	// 使用指定gorm db执行，主要用于事务执行
	SaveWithDb(ctx context.Context, db *gorm.DB, e T) error

	// DeleteById 根据实体主键删除实体
	DeleteById(ctx context.Context, id ...uint64) error

	// DeleteByIdWithDb 使用指定gorm db执行，主要用于事务执行
	DeleteByIdWithDb(ctx context.Context, db *gorm.DB, id ...uint64) error

	// DeleteByCond 根据实体条件删除实体
	DeleteByCond(ctx context.Context, cond any) error

	// DeleteByCondWithDb 使用指定gorm db执行，主要用于事务执行
	DeleteByCondWithDb(ctx context.Context, db *gorm.DB, cond any) error

	// ExecBySql 执行原生sql
	ExecBySql(sql string, params ...any) error

	// GetById 根据实体id查询
	GetById(id uint64, cols ...string) (T, error)

	// GetByIds 根据实体ids查询
	GetByIds(ids []uint64, cols ...string) ([]T, error)

	// GetByCond 根据实体条件查询实体信息（单个结果集）
	// @param cond 支持普通结构体或*model.QueryCond。如果cond为model.QueryCond，则需要使用Dest方法绑定值的指针
	GetByCond(cond any) error

	// SelectByCondToAny 根据条件查询数据映射至res
	SelectByCondToAny(cond any, res any) error

	// SelectByCond 根据条件查询模型实例数组
	SelectByCond(cond any, cols ...string) ([]T, error)

	// PageByCond 根据查询条件分页查询
	PageByCond(cond any, pageParam model.PageParam, cols ...string) (*model.PageResult[T], error)

	// SelectBySql 根据sql语句查询数据
	SelectBySql(sql string, res any, params ...any) error

	// CountByCond 根据指定条件统计model表的数量
	CountByCond(cond any) int64

	// SelectByCondWithOffset 根据条件查询数据并支持 offset + limit 分页
	SelectByCondWithOffset(cond any, limit int, offset int) ([]T, error)
}

var _ (Repo[*model.Model]) = (*RepoImpl[*model.Model])(nil)

// 基础repo接口
type RepoImpl[T model.ModelI] struct {
	model any // 模型实例

	modelType reflect.Type // 模型类型
}

func (br *RepoImpl[T]) Insert(ctx context.Context, e T) error {
	if db := GetDbFromCtx(ctx); db != nil {
		return br.InsertWithDb(ctx, db, e)
	}
	return gormx.Insert(br.fillBaseInfo(ctx, e))
}

func (br *RepoImpl[T]) InsertWithDb(ctx context.Context, db *gorm.DB, e T) error {
	return gormx.InsertWithDb(db, br.fillBaseInfo(ctx, e))
}

func (br *RepoImpl[T]) BatchInsert(ctx context.Context, es []T) error {
	if db := GetDbFromCtx(ctx); db != nil {
		return br.BatchInsertWithDb(ctx, db, es)
	}
	for _, e := range es {
		br.fillBaseInfo(ctx, e)
	}
	return gormx.BatchInsert[T](es)
}

func (br *RepoImpl[T]) BatchInsertWithDb(ctx context.Context, db *gorm.DB, es []T) error {
	for _, e := range es {
		br.fillBaseInfo(ctx, e)
	}
	return gormx.BatchInsertWithDb[T](db, es)
}

func (br *RepoImpl[T]) UpdateById(ctx context.Context, e T, columns ...string) error {
	return br.UpdateByIdWithDb(ctx, GetDbFromCtx(ctx), e, columns...)
}

func (br *RepoImpl[T]) UpdateByIdWithDb(ctx context.Context, db *gorm.DB, e T, columns ...string) error {
	if db == nil {
		return gormx.UpdateById(br.fillBaseInfo(ctx, e), columns...)
	}
	return gormx.UpdateByIdWithDb(db, br.fillBaseInfo(ctx, e), columns...)
}

func (br *RepoImpl[T]) UpdateByCond(ctx context.Context, values any, cond any) error {
	return br.UpdateByCondWithDb(ctx, GetDbFromCtx(ctx), values, cond)
}

func (br *RepoImpl[T]) UpdateByCondWithDb(ctx context.Context, db *gorm.DB, values any, cond any) error {
	if e, ok := values.(T); ok {
		// 先随机设置一个id，让fillBaseInfo不填充create信息
		e.SetId(1)
		e = br.fillBaseInfo(ctx, e)
		// model的主键值需为空，否则会带上主键条件
		e.SetId(0)
		values = e
	} else {
		var mapValues map[string]any
		// 非model实体，则为map
		if m, ok := values.(map[string]any); ok {
			mapValues = m
		} else if collxm, ok := values.(collx.M); ok {
			mapValues = map[string]any(collxm)
		}
		if len(mapValues) > 0 {
			mapValues[model.UpdateTimeColumn] = time.Now()
			if la := contextx.GetLoginAccount(ctx); la != nil {
				mapValues[model.ModifierColumn] = la.Username
				mapValues[model.ModifierIdColumn] = la.Id
			}
			values = mapValues
		}
	}

	if db == nil {
		return gormx.UpdateByCond(br.GetModel(), values, toQueryCond(cond))
	}
	return gormx.UpdateByCondWithDb(db, br.GetModel(), values, toQueryCond(cond))
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

func (br *RepoImpl[T]) DeleteById(ctx context.Context, id ...uint64) error {
	if db := GetDbFromCtx(ctx); db != nil {
		return br.DeleteByIdWithDb(ctx, db, id...)
	}
	return gormx.DeleteById(br.GetModel(), id...)
}

func (br *RepoImpl[T]) DeleteByIdWithDb(ctx context.Context, db *gorm.DB, id ...uint64) error {
	return gormx.DeleteByIdWithDb(db, br.GetModel(), id...)
}

func (br *RepoImpl[T]) DeleteByCond(ctx context.Context, cond any) error {
	if db := GetDbFromCtx(ctx); db != nil {
		return br.DeleteByCondWithDb(ctx, db, cond)
	}
	return gormx.DeleteByCond(br.GetModel(), toQueryCond(cond))
}

func (br *RepoImpl[T]) DeleteByCondWithDb(ctx context.Context, db *gorm.DB, cond any) error {
	return gormx.DeleteByCondWithDb(db, br.GetModel(), toQueryCond(cond))
}

func (br *RepoImpl[T]) ExecBySql(sql string, params ...any) error {
	return gormx.ExecSql(sql, params...)
}

func (br *RepoImpl[T]) GetById(id uint64, cols ...string) (T, error) {
	e := br.NewModel()
	return e, gormx.GetById(e, id, cols...)
}

func (br *RepoImpl[T]) GetByIds(ids []uint64, cols ...string) ([]T, error) {
	var models []T
	return models, br.SelectByCondToAny(model.NewCond().In(model.IdColumn, ids).Columns(cols...), &models)
}

func (br *RepoImpl[T]) GetByCond(cond any) error {
	return gormx.GetByCond(br.GetModel(), toQueryCond(cond))
}

func (br *RepoImpl[T]) SelectByCondToAny(cond any, res any) error {
	return gormx.SelectByCond(br.GetModel(), toQueryCond(cond).Dest(res))
}

func (br *RepoImpl[T]) SelectByCond(cond any, cols ...string) ([]T, error) {
	var models []T
	return models, br.SelectByCondToAny(toQueryCond(cond).Dest(models).Columns(cols...), &models)
}

func (br *RepoImpl[T]) PageByCond(cond any, pageParam model.PageParam, cols ...string) (*model.PageResult[T], error) {
	var models []T
	return gormx.PageByCond(br.GetModel(), toQueryCond(cond).Columns(cols...), pageParam, models)
}

func (br *RepoImpl[T]) SelectBySql(sql string, res any, params ...any) error {
	return gormx.SelectBySql(sql, res, params...)
}

func (br *RepoImpl[T]) CountByCond(cond any) int64 {
	return gormx.CountByCond(br.GetModel(), toQueryCond(cond))
}

func (br *RepoImpl[T]) SelectByCondWithOffset(cond any, limit int, offset int) ([]T, error) {
	var models []T
	err := gormx.NewQuery(br.GetModel(), toQueryCond(cond)).GenGdb().Limit(limit).Offset(offset).Find(&models).Error
	if err != nil {
		return nil, err
	}
	return models, nil
}

// NewModel 新建模型实例
func (br *RepoImpl[T]) NewModel() T {
	newModel := reflect.New(br.getModelType()).Interface()
	return newModel.(T)
}

// func (br *RepoImpl[T]) NewModes() *[]T {
// 	// 创建一个空的切片
// 	slice := reflect.MakeSlice(reflect.SliceOf(reflect.PointerTo(br.getModelType())), 0, 0)
// 	// 创建指向切片的指针
// 	ptrToSlice := reflect.New(slice.Type())
// 	// 设置指向切片的指针为创建的空切片
// 	ptrToSlice.Elem().Set(slice)
// 	// 转换指向切片的指针
// 	return ptrToSlice.Interface().(*[]T)
// }

// getModel 获取表的模型实例
func (br *RepoImpl[T]) GetModel() T {
	if br.model != nil {
		return br.model.(T)
	}

	br.model = br.NewModel()
	return br.model.(T)
}

// getModelType 获取模型类型(非指针模型)
func (br *RepoImpl[T]) getModelType() reflect.Type {
	if br.modelType != nil {
		return br.modelType
	}

	var model T
	modelType := reflect.TypeOf(model)
	// 检查 model 是否为指针类型
	if modelType.Kind() == reflect.Ptr {
		// 获取指针指向的类型
		modelType = modelType.Elem()
	}
	br.modelType = modelType
	return modelType
}

// 从上下文获取登录账号信息，并赋值至实体
func (br *RepoImpl[T]) fillBaseInfo(ctx context.Context, e T) T {
	// 默认使用数据库id策略, 若要改变则实体结构体自行覆盖FillBaseInfo方法。可参考 sys/entity.Resource
	e.FillBaseInfo(model.IdGenTypeNone, cmp.Or(contextx.GetLoginAccount(ctx), model.SysAccount))
	return e
}

// toQueryCond 统一转为*model.QueryCond
func toQueryCond(cond any) *model.QueryCond {
	if qc, ok := cond.(*model.QueryCond); ok {
		return qc
	}
	return model.NewModelCond(cond)
}

package gormx

import (
	"errors"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
	"strings"

	"gorm.io/gorm"
)

// ErrCondDestMissing 查询条件既没有绑定结果集也没有条件模型，无法确定输出去向
var ErrCondDestMissing = errors.New("query condition has neither dest nor cond model")

// ErrTableNameWithoutOrder 使用表名构建查询（关联查询等）时必须显式指定排序，
// 否则分页会在不确定的顺序上翻页，出现漏行/重复
var ErrTableNameWithoutOrder = errors.New("table based query requires explicit order by")

type Query struct {
	dbModel model.ModelI // 数据库模型
	table   string
	joins   string // join 类似 left join emails on emails.user_id = users.id

	cond *model.QueryCond // 条件
}

// NewQuery 构建查询条件。
//
// 软删过滤不在此决定：生成语句时（genGdbOn）按当时的条件统一判定，构造后再换条件也不会
// 残留构造期的结论
func NewQuery(dbModel model.ModelI, cond *model.QueryCond) *Query {
	if cond == nil {
		cond = model.NewCond()
	}
	return &Query{dbModel: dbModel, cond: cond}
}

// needDeletedFilter 判断是否需要自动附加逻辑删除过滤。
//
// 三种情况下不附加：调用方声明了要含已删除数据；调用方自己已经就该列表过态——否则自动过滤
// 会静默覆盖调用方的意图，「查回收站/已删除数据」永远返回空集
func needDeletedFilter(dbModel model.ModelI, cond *model.QueryCond) bool {
	if dbModel == nil || !dbModel.LogicDelete() {
		return false
	}
	return !cond.IsDeletedIgnored() && !cond.HasCondOnColumn(model.DeletedColumn)
}

// applyDeletedFilter 把逻辑删除过滤作为独立子句附加，不占用调用方的条件集合
func applyDeletedFilter(gdb *gorm.DB, dbModel model.ModelI, cond *model.QueryCond) {
	if needDeletedFilter(dbModel, cond) {
		gdb.Where(model.DeletedColumn, model.ModelUndeleted)
	}
}

func NewQueryWithTableName(tableName string) *Query {
	return &Query{table: tableName}
}

// WithCond 替换查询条件。
//
// 软删过滤在生成语句时按替换后的条件统一判定（见 genGdbOn），不会残留构造期的结论
func (q *Query) WithCond(cond *model.QueryCond) *Query {
	q.cond = cond
	return q
}

func (q *Query) Joins(joins string) *Query {
	q.joins = joins
	return q
}

func (q *Query) GenGdb() *gorm.DB {
	return q.genGdbOn(global.Db)
}

func (q *Query) genGdbOn(db *gorm.DB) *gorm.DB {
	var gdb *gorm.DB
	if q.table != "" {
		gdb = db.Table(q.table)
	} else {
		gdb = db.Model(q.dbModel)
	}

	cond := q.cond
	if cond == nil {
		cond = model.NewCond()
	}
	// 构造期记录的非法输入（如非法排序片段）在此转为错误，绝不带着它去访问数据库：
	// GORM 的 query/update/delete 回调开头均会因 db.Error 非空而直接返回
	if condErr := cond.GetErr(); condErr != nil {
		gdb.AddError(condErr)
		return gdb
	}
	if len(cond.GetSelectColumns()) > 0 {
		gdb.Select(cond.GetSelectColumns())
	}
	if q.joins != "" {
		gdb.Joins(q.joins)
	}

	setGdbWhere(gdb, cond)
	// 软删过滤作为独立子句附加，不占用调用方的条件集合：写进 cond 会就地污染调用方对象，
	// 且它与调用方自己就软删列表达的条件共用同一个列名，混在一起必然互相覆盖。
	// 表名查询（dbModel 为 nil）时 needDeletedFilter 天然判假，不会把单表列名带进关联查询
	applyDeletedFilter(gdb, q.dbModel, cond)
	if orderBy := cond.GetOrderBy(); len(orderBy) > 0 {
		gdb.Order(strings.Join(orderBy, ","))
	} else if q.table == "" {
		// 实体查询默认按主键倒序，保证分页与游标遍历的顺序稳定
		gdb.Order(model.IdColumn + " desc")
	} else {
		// 表名查询无从得知是否存在 id 列，静默加排序会生成非法列引用；交由调用方显式声明
		gdb.AddError(ErrTableNameWithoutOrder)
	}

	return gdb
}

// setGdbWhere 按追加顺序把条件片段绑定到 db 上。
//
// 顺序稳定是刻意要求：条件片段若来自 map 遍历，同一条逻辑查询每次执行都会换一种 SQL 文本，
// 慢 SQL 归因、语句缓存与回归比对因此全部失去可比性
func setGdbWhere(gdb *gorm.DB, cond *model.QueryCond) *gorm.DB {
	if cond.GetCondModel() != nil {
		gdb.Where(cond.GetCondModel())
	}

	for _, w := range cond.GetWheres() {
		gdb.Where(w.Expr, w.Args...)
	}

	return gdb
}

package gormx

import (
	"fmt"
	"mayfly-go/pkg/consts"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/anyx"
	"strings"

	"gorm.io/gorm"
)

type QueryCond struct {
	dbModel       any // 数据库模型
	table         string
	selectColumns string   // 查询的列信息
	joins         string   // join 类似 left join emails on emails.user_id = users.id
	condModel     any      // 条件模型
	columns       []string // 查询的所有列（与values一一对应）
	values        []any    // 查询列对应的值
	orderBy       []string
}

// NewQuery 构建查询条件
func NewQuery(dbModel any) *QueryCond {
	return &QueryCond{dbModel: dbModel}
}

func NewQueryWithTableName(tableName string) *QueryCond {
	return &QueryCond{table: tableName}
}

func (q *QueryCond) WithCondModel(condModel any) *QueryCond {
	q.condModel = condModel
	return q
}

func (q *QueryCond) WithOrderBy(orderBy ...string) *QueryCond {
	q.orderBy = append(q.orderBy, orderBy...)
	return q
}

func (q *QueryCond) Select(columns ...string) *QueryCond {
	q.selectColumns = strings.Join(columns, ",")
	return q
}

func (q *QueryCond) Joins(joins string) *QueryCond {
	q.joins = joins
	return q
}

func (q *QueryCond) OrderByDesc(column string) *QueryCond {
	q.orderBy = append(q.orderBy, fmt.Sprintf("%s DESC", column))
	return q
}

func (q *QueryCond) OrderByAsc(column string) *QueryCond {
	q.orderBy = append(q.orderBy, fmt.Sprintf("%s ASC", column))
	return q
}

// 添加未删除数据过滤条件（适用于单表用实体查询）
func (q *QueryCond) Undeleted() *QueryCond {
	// 存在表名，则可能为关联查询等，需要自行设置未删除条件过滤
	if q.table != "" {
		return q
	}
	return q.Eq0(model.DeletedColumn, model.ModelUndeleted)
}

func (q *QueryCond) GenGdb() *gorm.DB {
	return q.GenGdbWithDb(global.Db)
}

func (q *QueryCond) GenGdbWithDb(db *gorm.DB) *gorm.DB {
	var gdb *gorm.DB
	if q.table != "" {
		gdb = db.Table(q.table)
	} else {
		gdb = db.Model(q.dbModel)
	}

	if q.selectColumns != "" {
		gdb.Select(q.selectColumns)
	}
	if q.joins != "" {
		gdb.Joins(q.joins)
	}
	if q.condModel != nil {
		gdb.Where(q.condModel)
	}
	for i, v := range q.columns {
		gdb.Where(v, q.values[i])
	}
	if len(q.orderBy) > 0 {
		gdb.Order(strings.Join(q.orderBy, ","))
	} else {
		gdb.Order("id desc")
	}

	return gdb
}

// Eq 等于 =
func (q *QueryCond) Eq(column string, val any) *QueryCond {
	return q.Cond(consts.Eq, column, val, true)
}

// Eq 等于 = (零值也不忽略该添加)
func (q *QueryCond) Eq0(column string, val any) *QueryCond {
	return q.Cond(consts.Eq, column, val, false)
}

func (q *QueryCond) Like(column string, val string) *QueryCond {
	if val == "" {
		return q
	}
	return q.Cond(consts.Like, column, "%"+val+"%", false)
}

func (q *QueryCond) RLike(column string, val string) *QueryCond {
	if val == "" {
		return q
	}
	return q.Cond(consts.Like, column, val+"%", false)
}

func (q *QueryCond) In(column string, val any) *QueryCond {
	return q.Cond(consts.In, column, val, true)
}

func (q *QueryCond) NotIn(column string, val any) *QueryCond {
	return q.Cond(consts.NotIn, column, val, true)
}

func (q *QueryCond) In0(column string, val any) *QueryCond {
	return q.Cond(consts.In, column, val, true)
}

// // Ne 不等于 !=
func (q *QueryCond) Ne(column string, val any) *QueryCond {
	q.Cond(consts.Ne, column, val, true)
	return q
}

// Gt 大于 >
func (q *QueryCond) Gt(column string, val any) *QueryCond {
	q.Cond(consts.Gt, column, val, true)
	return q
}

// Ge 大于等于 >=
func (q *QueryCond) Ge(column string, val any) *QueryCond {
	q.Cond(consts.Ge, column, val, true)
	return q
}

// Lt 小于 <
func (q *QueryCond) Lt(column string, val any) *QueryCond {
	q.Cond(consts.Lt, column, val, true)
	return q
}

// Le 小于等于 <=
func (q *QueryCond) Le(column string, val any) *QueryCond {
	q.Cond(consts.Le, column, val, true)
	return q
}

// And条件
func (q *QueryCond) And(column string, val any) *QueryCond {
	q.columns = append(q.columns, column)
	q.values = append(q.values, val)
	return q
}

func (q *QueryCond) Cond(cond, column string, val any, skipBlank bool) *QueryCond {
	// 零值跳过
	if skipBlank && anyx.IsBlank(val) {
		return q
	}
	q.columns = append(q.columns, fmt.Sprintf("%s %s ?", column, cond))
	q.values = append(q.values, val)
	return q
}

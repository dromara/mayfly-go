package gormx

import (
	"fmt"
	"mayfly-go/pkg/consts"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/utils"
	"strings"

	"gorm.io/gorm"
)

type QueryCond struct {
	selectColumns string   // 查询的列信息
	joins         string   // join 类似 left join emails on emails.user_id = users.id
	dbModel       any      // 数据库模型
	condModel     any      // 条件模型
	columns       []string // 查询的所有列（与values一一对应）
	values        []any    // 查询列对应的值
	orderBy       []string
}

// NewQuery 构建查询条件
func NewQuery(dbModel any) *QueryCond {
	return &QueryCond{dbModel: dbModel}
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

// Eq 等于 =
func (q *QueryCond) Eq(column string, val any) *QueryCond {
	return q.Cond(consts.Eq, column, val)
}

func (q *QueryCond) Like(column string, val string) *QueryCond {
	if val == "" {
		return q
	}
	return q.Cond(consts.Like, column, "%"+val+"%")
}

func (q *QueryCond) RLike(column string, val string) *QueryCond {
	if val == "" {
		return q
	}
	return q.Cond(consts.Like, column, val+"%")
}

func (q *QueryCond) In(column string, val any) *QueryCond {
	return q.Cond(consts.In, column, val)
}

func (q *QueryCond) OrderByDesc(column string) *QueryCond {
	q.orderBy = append(q.orderBy, fmt.Sprintf("%s DESC", column))
	return q
}

func (q *QueryCond) OrderByAsc(column string) *QueryCond {
	q.orderBy = append(q.orderBy, fmt.Sprintf("%s ASC", column))
	return q
}

func (q *QueryCond) GenGdb() *gorm.DB {
	gdb := global.Db.Model(q.dbModel)

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

// // Ne 不等于 !=
func (q *QueryCond) Ne(column string, val any) *QueryCond {
	q.Cond(consts.Ne, column, val)
	return q
}

// Gt 大于 >
func (q *QueryCond) Gt(column string, val any) *QueryCond {
	q.Cond(consts.Gt, column, val)
	return q
}

// Ge 大于等于 >=
func (q *QueryCond) Ge(column string, val any) *QueryCond {
	q.Cond(consts.Ge, column, val)
	return q
}

// Lt 小于 <
func (q *QueryCond) Lt(column string, val any) *QueryCond {
	q.Cond(consts.Lt, column, val)
	return q
}

// Le 小于等于 <=
func (q *QueryCond) Le(column string, val any) *QueryCond {
	q.Cond(consts.Le, column, val)
	return q
}

func (q *QueryCond) Cond(cond, column string, val any) *QueryCond {
	// 零值跳过
	if utils.IsBlank(val) {
		return q
	}
	q.columns = append(q.columns, fmt.Sprintf("%s %s ?", column, cond))
	q.values = append(q.values, val)
	return q
}

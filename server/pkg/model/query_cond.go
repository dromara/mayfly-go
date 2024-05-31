package model

import (
	"fmt"
	"mayfly-go/pkg/consts"
	"mayfly-go/pkg/utils/anyx"
)

type QueryCond struct {
	selectColumns []string // 查询的列信息
	condModel     any      // 条件模型

	wheres  map[string][]any
	orderBy []string

	dest any // 结果集指针
}

// NewCond 构建查询条件
func NewCond() *QueryCond {
	return &QueryCond{}
}

// NewModelCond  新建模型条件（使用模型中非零值作为条件）
func NewModelCond(condModel any) *QueryCond {
	return &QueryCond{condModel: condModel}
}

// 设置结果集绑定的指针
func (q *QueryCond) Dest(dest any) *QueryCond {
	q.dest = dest
	return q
}

// Columns 设置查询的列
func (q *QueryCond) Columns(columns ...string) *QueryCond {
	q.selectColumns = columns
	return q
}

func (q *QueryCond) OrderByDesc(column string) *QueryCond {
	return q.OrderBy(fmt.Sprintf("%s DESC", column))
}

func (q *QueryCond) OrderByAsc(column string) *QueryCond {
	return q.OrderBy(fmt.Sprintf("%s ASC", column))
}

func (q *QueryCond) OrderBy(orderBy ...string) *QueryCond {
	q.orderBy = append(q.orderBy, orderBy...)
	return q
}

// Eq 等于 =
func (q *QueryCond) Eq(column string, val any) *QueryCond {
	return q.Cond(consts.Eq, column, val, true)
}

// Eq 等于 = (零值也不忽略该添加)
func (q *QueryCond) Eq0(column string, val any) *QueryCond {
	return q.Cond(consts.Eq, column, val, false)
}

// Like LIKE %xx%
func (q *QueryCond) Like(column string, val string) *QueryCond {
	if val == "" {
		return q
	}
	return q.Cond(consts.Like, column, "%"+val+"%", false)
}

// RLike LIKE xx%
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
func (q *QueryCond) And(column string, val ...any) *QueryCond {
	if q.wheres == nil {
		q.wheres = make(map[string][]any)
	}
	q.wheres[column] = val
	return q
}

func (q *QueryCond) Cond(cond, column string, val any, skipBlank bool) *QueryCond {
	// 零值跳过
	if skipBlank && anyx.IsBlank(val) {
		return q
	}
	return q.And(fmt.Sprintf("%s %s ?", column, cond), val)
}

func (q *QueryCond) GetWheres() map[string][]any {
	return q.wheres
}

func (q *QueryCond) GetCondModel() any {
	return q.condModel
}

func (q *QueryCond) GetSelectColumns() []string {
	return q.selectColumns
}

func (q *QueryCond) GetOrderBy() []string {
	return q.orderBy
}

// 获取输出目标结果集指针，若dest为空，则使用condModel
func (q *QueryCond) GetDest() any {
	if !anyx.IsBlank(q.dest) {
		return q.dest
	}
	if !anyx.IsBlank(q.condModel) {
		return q.condModel
	}

	panic("*model.QueryCond dest and condModel is nil")
}

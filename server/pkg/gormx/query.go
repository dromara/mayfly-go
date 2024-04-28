package gormx

import (
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
	"strings"

	"gorm.io/gorm"
)

type Query struct {
	dbModel any // 数据库模型
	table   string
	joins   string // join 类似 left join emails on emails.user_id = users.id

	cond *model.QueryCond // 条件
}

// NewQuery 构建查询条件
func NewQuery(dbModel any, cond *model.QueryCond) *Query {
	return &Query{dbModel: dbModel, cond: cond}
}

func NewQueryWithTableName(tableName string) *Query {
	return &Query{table: tableName}
}

func (q *Query) WithCond(cond *model.QueryCond) *Query {
	q.cond = cond
	return q
}

func (q *Query) Joins(joins string) *Query {
	q.joins = joins
	return q
}

// 添加未删除数据过滤条件（适用于单表用实体查询）
func (q *Query) Undeleted() *Query {
	// 存在表名，则可能为关联查询等，需要自行设置未删除条件过滤
	if q.table != "" {
		return q
	}
	q.cond.Eq0(model.DeletedColumn, model.ModelUndeleted)
	return q
}

func (q *Query) GenGdb() *gorm.DB {
	return q.GenGdbWithDb(global.Db)
}

func (q *Query) GenGdbWithDb(db *gorm.DB) *gorm.DB {
	var gdb *gorm.DB
	if q.table != "" {
		gdb = db.Table(q.table)
	} else {
		gdb = db.Model(q.dbModel)
	}

	cond := q.cond
	if len(cond.GetSelectColumns()) > 0 {
		gdb.Select(cond.GetSelectColumns())
	}
	if q.joins != "" {
		gdb.Joins(q.joins)
	}

	setGdbWhere(gdb, cond)

	if len(cond.GetOrderBy()) > 0 {
		gdb.Order(strings.Join(cond.GetOrderBy(), ","))
	} else {
		gdb.Order("id desc")
	}

	return gdb
}

func setGdbWhere(gdb *gorm.DB, cond *model.QueryCond) *gorm.DB {
	if cond.GetCondModel() != nil {
		gdb.Where(cond.GetCondModel())
	}

	for i, v := range cond.GetWheres() {
		gdb.Where(i, v)
	}
	return gdb
}

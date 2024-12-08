package sqlite

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"strings"
)

type SQLGenerator struct {
	dialect dbi.Dialect
}

func (ssg *SQLGenerator) GenTableDDL(table dbi.Table, columns []dbi.Column, dropBeforeCreate bool) []string {
	quoter := ssg.dialect.Quoter()

	sqlArr := make([]string, 0)
	tbName := ssg.dialect.Quoter().Quote(table.TableName)
	if dropBeforeCreate {
		sqlArr = append(sqlArr, fmt.Sprintf("DROP TABLE IF EXISTS %s", tbName))
	}
	// 组装建表语句
	createSql := fmt.Sprintf("CREATE TABLE %s (\n", tbName)
	fields := make([]string, 0)

	// 把通用类型转换为达梦类型
	for _, column := range columns {
		fields = append(fields, ssg.genColumnBasicSql(quoter, column))
	}
	createSql += strings.Join(fields, ",\n")
	createSql += "\n)"

	sqlArr = append(sqlArr, createSql)

	return sqlArr
}

func (ssg *SQLGenerator) GenIndexDDL(table dbi.Table, indexs []dbi.Index) []string {
	quoter := ssg.dialect.Quoter()
	quote := quoter.Quote

	sqls := make([]string, 0)
	for _, index := range indexs {
		unique := ""
		if index.IsUnique {
			unique = "unique"
		}
		// 取出列名，添加引号
		cols := strings.Split(index.ColumnName, ",")
		colNames := make([]string, len(cols))
		for i, name := range cols {
			colNames[i] = quote(name)
		}
		// 创建前尝试删除
		sqls = append(sqls, fmt.Sprintf("DROP INDEX IF EXISTS \"%s\"", index.IndexName))

		sqlTmp := "CREATE %s INDEX %s ON %s (%s) "
		sqls = append(sqls, fmt.Sprintf(sqlTmp, unique, quote(index.IndexName), quote(table.TableName), strings.Join(colNames, ",")))
	}

	return sqls
}

func (ssg *SQLGenerator) GenInsert(tableName string, columns []dbi.Column, values [][]any, duplicateStrategy int) []string {
	if duplicateStrategy == dbi.DuplicateStrategyNone {
		return collx.AsArray(dbi.GenCommonInsert(ssg.dialect, DbTypeSqlite, tableName, columns, values))
	}

	sqls := make([]string, 0)
	sqls = append(sqls, "PRAGMA foreign_keys = false")

	prefix := "insert or ignore into"
	if duplicateStrategy == dbi.DuplicateStrategyUpdate {
		prefix = "insert or replace into"
	}

	columnStr, valuesStrs := dbi.GenInsertSqlColumnAndValues(ssg.dialect, DbTypeSqlite, columns, values)

	sqls = append(sqls, "PRAGMA foreign_keys = true")
	sqls = append(sqls, fmt.Sprintf("%s %s %s VALUES \n%s", prefix, ssg.dialect.Quoter().Quote(tableName), columnStr, strings.Join(valuesStrs, ",\n")))
	return sqls
}

func (ssg *SQLGenerator) genColumnBasicSql(quoter dbi.Quoter, column dbi.Column) string {
	incr := ""
	if column.IsIdentity {
		incr = " AUTOINCREMENT"
	}

	nullAble := ""
	if !column.Nullable {
		nullAble = " NOT NULL"
	}

	quoteColumnName := quoter.Quote(column.ColumnName)

	// 如果是主键，则直接返回，不判断默认值
	if column.IsPrimaryKey {
		return fmt.Sprintf(" %s integer PRIMARY KEY%s%s", quoteColumnName, incr, nullAble)
	}

	defVal := "" // 默认值需要判断引号，如函数是不需要引号的 // 为了防止跨源函数不支持 当默认值是函数时，不需要设置默认值
	if column.ColumnDefault != "" && !strings.Contains(column.ColumnDefault, "(") {
		// 哪些字段类型默认值需要加引号
		mark := false
		if collx.ArrayAnyMatches([]string{"char", "text", "date", "time", "lob"}, strings.ToLower(string(column.DataType))) {
			// 当数据类型是日期时间，默认值是日期时间函数时，默认值不需要引号
			if collx.ArrayAnyMatches([]string{"date", "time"}, strings.ToLower(string(column.DataType))) &&
				collx.ArrayAnyMatches([]string{"DATE", "TIME"}, strings.ToUpper(column.ColumnDefault)) {
				mark = false
			} else {
				mark = true
			}
		}
		if mark {
			defVal = fmt.Sprintf(" DEFAULT '%s'", column.ColumnDefault)
		} else {
			defVal = fmt.Sprintf(" DEFAULT %s", column.ColumnDefault)
		}
	}

	return fmt.Sprintf(" %s %s%s%s", quoteColumnName, column.GetColumnType(), nullAble, defVal)
}

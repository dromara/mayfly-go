package mysql

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"strings"

	"github.com/may-fly/cast"
)

type SQLGenerator struct {
	Dialect dbi.Dialect
}

func (msg *SQLGenerator) GenTableDDL(table dbi.Table, columns []dbi.Column, dropBeforeCreate bool) []string {
	sqlArr := make([]string, 0)
	quoter := msg.Dialect.Quoter()

	if dropBeforeCreate {
		sqlArr = append(sqlArr, fmt.Sprintf("DROP TABLE IF EXISTS %s", quoter.Quote(table.TableName)))
	}

	// 组装建表语句
	createSql := fmt.Sprintf("CREATE TABLE %s (\n", quoter.Quote(table.TableName))
	fields := make([]string, 0)
	pks := make([]string, 0)

	for _, column := range columns {
		if column.IsPrimaryKey {
			pks = append(pks, column.ColumnName)
		}
		fields = append(fields, msg.genColumnBasicSql(quoter, column))
	}

	// 建表ddl
	createSql += strings.Join(fields, ",\n")
	if len(pks) > 0 {
		createSql += fmt.Sprintf(", \nPRIMARY KEY (%s)", strings.Join(pks, ","))
	}
	createSql += "\n)"

	// 表注释
	if table.TableComment != "" {
		createSql += fmt.Sprintf(" COMMENT '%s'", dbi.QuoteEscape(table.TableComment))
	}

	sqlArr = append(sqlArr, createSql)

	return sqlArr
}

func (msg *SQLGenerator) GenIndexDDL(table dbi.Table, indexs []dbi.Index) []string {
	sqlArr := make([]string, 0)
	quoter := msg.Dialect.Quoter()

	for _, index := range indexs {
		unique := ""
		if index.IsUnique {
			unique = "unique"
		}
		// 取出列名，添加引号
		colNames := quoter.Quotes(strings.Split(index.ColumnName, ","))

		// 暂时先处理单个索引的情况，多个涉及获取索引时的合并等，以及前端调整等，后续完善
		if subPart := cast.ToInt(index.Extra[IndexSubPartKey]); subPart > 0 && len(colNames) == 1 {
			colNames[0] = fmt.Sprintf("%s(%d)", colNames[0], subPart)
		}

		sqlTmp := "ALTER TABLE %s ADD %s INDEX %s(%s) USING BTREE"
		sqlStr := fmt.Sprintf(sqlTmp, quoter.Quote(table.TableName), unique, quoter.Quote(index.IndexName), strings.Join(colNames, ","))
		comment := dbi.QuoteEscape(index.IndexComment)
		if comment != "" {
			sqlStr += fmt.Sprintf(" COMMENT '%s'", comment)
		}
		sqlArr = append(sqlArr, sqlStr)
	}

	return sqlArr
}

func (msg *SQLGenerator) GenInsert(tableName string, columns []dbi.Column, values [][]any, duplicateStrategy int) []string {
	if duplicateStrategy == dbi.DuplicateStrategyNone {
		return collx.AsArray(dbi.GenCommonInsert(msg.Dialect, DbTypeMysql, tableName, columns, values))
	}

	prefix := "insert ignore into"
	if duplicateStrategy == dbi.DuplicateStrategyUpdate {
		prefix = "replace into"
	}

	quote := msg.Dialect.Quoter().Quote
	columnStr, valuesStrs := dbi.GenInsertSqlColumnAndValues(msg.Dialect, DbTypeMysql, columns, values)

	return collx.AsArray[string](fmt.Sprintf("%s %s %s VALUES \n%s", prefix, quote(tableName), columnStr, strings.Join(valuesStrs, ",\n")))
}

func (msg *SQLGenerator) genColumnBasicSql(quoter dbi.Quoter, column dbi.Column) string {
	dataType := column.DataType

	incr := ""
	if column.AutoIncrement {
		incr = " AUTO_INCREMENT"
	}

	nullAble := ""
	if !column.Nullable {
		nullAble = " NOT NULL"
	}
	columnType := column.GetColumnType()
	if nullAble == "" && strings.Contains(columnType, "timestamp") {
		nullAble = " NULL"
	}

	defVal := "" // 默认值需要判断引号，如函数是不需要引号的
	if column.ColumnDefault != "" &&
		// 当默认值是字符串'NULL'时，不需要设置默认值
		column.ColumnDefault != "NULL" &&
		// 为了防止跨源函数不支持 当默认值是函数时，不需要设置默认值
		!strings.Contains(column.ColumnDefault, "(") {
		// 哪些字段类型默认值需要加引号
		mark := false
		if collx.ArrayAnyMatches([]string{"char", "text", "date", "time", "lob"}, strings.ToLower(dataType)) {
			// 当数据类型是日期时间，默认值是日期时间函数时，默认值不需要引号
			if collx.ArrayAnyMatches([]string{"date", "time"}, strings.ToLower(dataType)) &&
				collx.ArrayAnyMatches([]string{"DATE", "TIME"}, strings.ToUpper(column.ColumnDefault)) {
				mark = false
			} else {
				mark = true
			}
		}
		if mark {
			// 去掉单引号
			column.ColumnDefault = strings.Trim(column.ColumnDefault, "'")
			defVal = fmt.Sprintf(" DEFAULT '%s'", column.ColumnDefault)
		} else {
			defVal = fmt.Sprintf(" DEFAULT %s", column.ColumnDefault)
		}
	}
	comment := ""
	if column.ColumnComment != "" {
		// 防止注释内含有特殊字符串导致sql出错
		commentStr := dbi.QuoteEscape(column.ColumnComment)
		comment = fmt.Sprintf(" COMMENT '%s'", commentStr)
	}

	columnSql := fmt.Sprintf(" %s %s%s%s%s%s", quoter.Quote(column.ColumnName), columnType, nullAble, incr, defVal, comment)
	return columnSql
}

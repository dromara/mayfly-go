package dm

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"regexp"
	"strings"
)

type SQLGenerator struct {
	Dialect  dbi.Dialect
	Metadata dbi.Metadata
}

func (sg *SQLGenerator) GenTableDDL(table dbi.Table, columns []dbi.Column, dropBeforeCreate bool) []string {
	quoter := sg.Dialect.Quoter()
	quote := quoter.Quote
	tbName := quote(table.TableName)
	sqlArr := make([]string, 0)

	if dropBeforeCreate {
		sqlArr = append(sqlArr, fmt.Sprintf("drop table if exists %s", tbName))
	}
	// 组装建表语句
	createSql := fmt.Sprintf("create table %s (", tbName)
	fields := make([]string, 0)
	pks := make([]string, 0)
	columnComments := make([]string, 0)

	for _, column := range columns {
		if column.IsPrimaryKey {
			pks = append(pks, quote(column.ColumnName))
		}
		fields = append(fields, sg.genColumnBasicSql(quoter, column))
		if column.ColumnComment != "" {
			comment := dbi.QuoteEscape(column.ColumnComment)
			columnComments = append(columnComments, fmt.Sprintf("comment on column %s.%s is '%s'", tbName, quote(column.ColumnName), comment))
		}
	}
	createSql += strings.Join(fields, ",\n")
	if len(pks) > 0 {
		createSql += fmt.Sprintf(",\n PRIMARY KEY (%s)", strings.Join(pks, ","))
	}
	createSql += "\n)"

	tableCommentSql := ""
	if table.TableComment != "" {
		comment := dbi.QuoteEscape(table.TableComment)
		tableCommentSql = fmt.Sprintf("comment on table %s is '%s'", tbName, comment)
	}

	sqlArr = append(sqlArr, createSql)
	if tableCommentSql != "" {
		sqlArr = append(sqlArr, tableCommentSql)
	}

	if len(columnComments) > 0 {
		sqlArr = append(sqlArr, columnComments...)
	}

	return sqlArr
}

func (sg *SQLGenerator) GenIndexDDL(table dbi.Table, indexs []dbi.Index) []string {
	quote := sg.Dialect.Quoter().Quote
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

		sqls = append(sqls, fmt.Sprintf("create %s index %s on %s(%s)", unique, quote(index.IndexName), quote(table.TableName), strings.Join(colNames, ",")))
	}
	return sqls
}

func (sg *SQLGenerator) GenInsert(tableName string, columns []dbi.Column, values [][]any, duplicateStrategy int) []string {
	quoter := sg.Dialect.Quoter()
	quote := quoter.Quote

	if duplicateStrategy == dbi.DuplicateStrategyNone {
		identityInsert := ""
		// 有自增列的才加上这个语句
		if collx.AnyMatch(columns, func(column dbi.Column) bool { return column.IsIdentity }) {
			identityInsert = fmt.Sprintf("set identity_insert %s on;", quote(tableName))
		}

		// 达梦数据库只能一条条的执行insert语句，所以这里需要将values拆分成多条insert语句
		return collx.ArrayMap(values, func(value []any) string {
			columnStr, valuesStrs := dbi.GenInsertSqlColumnAndValues(sg.Dialect, DbTypeDM, columns, values)
			return fmt.Sprintf("%s insert into %s %s values %s", identityInsert, quote(tableName), columnStr, strings.Join(valuesStrs, ",\n"))
		})
	}

	// 查询主键字段
	uniqueCols := make([]string, 0)
	caseSqls := make([]string, 0)
	metadata := sg.Metadata
	tableCols, _ := metadata.GetColumns(tableName)
	identityCols := make([]string, 0)
	for _, col := range tableCols {
		if col.IsPrimaryKey {
			uniqueCols = append(uniqueCols, col.ColumnName)
			caseSqls = append(caseSqls, fmt.Sprintf("( T1.%s = T2.%s )", quote(col.ColumnName), quote(col.ColumnName)))
		}
		if col.IsIdentity {
			// 自增字段不放入insert内，即使是设置了identity_insert on也不起作用
			identityCols = append(identityCols, quote(col.ColumnName))
		}
	}
	// 查询唯一索引涉及到的字段，并组装到match条件内
	indexs, _ := metadata.GetTableIndex(tableName)
	for _, index := range indexs {
		if index.IsUnique {
			cols := strings.Split(index.ColumnName, ",")
			tmp := make([]string, 0)
			for _, col := range cols {
				uniqueCols = append(uniqueCols, col)
				tmp = append(tmp, fmt.Sprintf(" T1.%s = T2.%s ", quote(col), quote(col)))
			}
			caseSqls = append(caseSqls, fmt.Sprintf("( %s )", strings.Join(tmp, " AND ")))
		}
	}

	// 重复数据处理策略
	phs := make([]string, 0)
	insertVals := make([]string, 0)
	upds := make([]string, 0)
	insertCols := make([]string, 0)
	for _, column := range columns {
		columnName := column.ColumnName
		phs = append(phs, fmt.Sprintf("? %s", columnName))
		if !collx.ArrayContains(uniqueCols, quoter.Trim(columnName)) {
			upds = append(upds, fmt.Sprintf("T1.%s = T2.%s", columnName, columnName))
		}
		if !collx.ArrayContains(identityCols, columnName) {
			insertCols = append(insertCols, columnName)
			insertVals = append(insertVals, fmt.Sprintf("T2.%s", columnName))
		}

	}
	t2s := make([]string, 0)
	for i := 0; i < len(values); i++ {
		t2s = append(t2s, fmt.Sprintf("SELECT %s FROM dual", strings.Join(phs, ",")))
	}
	t2 := strings.Join(t2s, " UNION ALL ")

	sqlTemp := "MERGE INTO " + quote(tableName) + " T1 USING (" + t2 + ") T2 ON " + strings.Join(caseSqls, " OR ")
	sqlTemp += "WHEN NOT MATCHED THEN INSERT (" + strings.Join(insertCols, ",") + ") VALUES (" + strings.Join(insertVals, ",") + ")"
	sqlTemp += "WHEN MATCHED THEN UPDATE SET " + strings.Join(upds, ",")

	return collx.AsArray(sqlTemp)
}

func (sg *SQLGenerator) genColumnBasicSql(quoter dbi.Quoter, column dbi.Column) string {
	colName := quoter.Quote(column.ColumnName)
	dataType := column.DataType

	incr := ""
	if column.IsIdentity {
		incr = " IDENTITY"
	}

	nullAble := ""
	if !column.Nullable {
		nullAble = " NOT NULL"
	}

	defVal := "" // 默认值需要判断引号，如函数是不需要引号的 // 为了防止跨源函数不支持 当默认值是函数时，不需要设置默认值
	if column.ColumnDefault != "" && !strings.Contains(column.ColumnDefault, "(") {
		// 哪些字段类型默认值需要加引号
		mark := false
		if regexp.MustCompile(`'.*'`).MatchString(column.ColumnDefault) {
			// 字符串默认值
			mark = false
		} else if collx.ArrayAnyMatches([]string{"char", "text", "date", "time", "lob"}, strings.ToLower(dataType)) {
			// 当数据类型是日期时间，默认值是日期时间函数时，默认值不需要引号
			if collx.ArrayAnyMatches([]string{"date", "time"}, strings.ToLower(dataType)) &&
				collx.ArrayAnyMatches([]string{"DATE", "TIME"}, strings.ToUpper(column.ColumnDefault)) {
				mark = false
			} else {
				mark = true
			}
			// 空
			if column.ColumnDefault == "NULL" {
				mark = false
			}
		}
		if mark {
			defVal = fmt.Sprintf(" DEFAULT '%s'", column.ColumnDefault)
		} else {
			defVal = fmt.Sprintf(" DEFAULT %s", column.ColumnDefault)
		}
	}

	columnSql := fmt.Sprintf(" %s %s%s%s%s", colName, column.GetColumnType(), incr, nullAble, defVal)
	return columnSql
}

package dm

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"strings"
)

var _ dbi.SQLGenerator = (*SQLGenerator)(nil)

type SQLGenerator struct {
	Dialect dbi.Dialect
}

func (sg *SQLGenerator) GenTableDDL(table dbi.Table, columns []dbi.Column, dropBeforeCreate bool) []string {
	quoter := sg.Dialect.Quoter()
	quote := quoter.QuoteIdent
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
	quote := sg.Dialect.Quoter().QuoteIdent
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

func (sg *SQLGenerator) GenInsert(tableName string, columns []dbi.Column, values [][]any, duplicateStrategy int, targetTableMeta *dbi.TargetTableMeta) []string {
	quoter := sg.Dialect.Quoter()
	quote := quoter.QuoteIdent

	if duplicateStrategy != dbi.DuplicateStrategyUpdate || targetTableMeta == nil || len(targetTableMeta.UniqueColumns) == 0 {
		// 直接插入（无法生成 merge 语句时也退化为直接插入，避免静默丢失数据由数据库主键约束报错提示）
		return sg.genSimpleInserts(tableName, columns, values)
	}

	uniqueCols := make([]string, 0)
	caseSqls := make([]string, 0)
	identityCols := targetTableMeta.IdentityColumns
	for _, col := range targetTableMeta.UniqueColumns {
		uniqueCols = append(uniqueCols, col)
		caseSqls = append(caseSqls, fmt.Sprintf("( T1.%s = T2.%s )", quote(col), quote(col)))
	}

	// 重复数据处理策略
	upds := make([]string, 0)
	insertCols := make([]string, 0)
	insertVals := make([]string, 0)
	for _, column := range columns {
		columnName := column.ColumnName
		quoteName := quote(columnName)
		// 标识列既不可插入也不可更新（DM与Oracle同源语义，更新标识列直接报错），直接跳过
		if collx.ArrayContains(identityCols, quoter.Trim(columnName)) {
			continue
		}
		if !collx.ArrayContains(uniqueCols, quoter.Trim(columnName)) {
			upds = append(upds, fmt.Sprintf("T1.%s = T2.%s", quoteName, quoteName))
		}
		insertCols = append(insertCols, quoteName)
		insertVals = append(insertVals, fmt.Sprintf("T2.%s", quoteName))

	}
	if len(upds) == 0 {
		// 所有列均为唯一键列，无法生成update子句，退化为直接插入
		return sg.genSimpleInserts(tableName, columns, values)
	}

	// GenInsert返回的SQL由调用方Exec无参数绑定执行，无法使用?占位符；
	// 需将行值以字面量形式内联到USING子查询中（参照mssql的merge实现）
	valueSql := make([]string, 0, len(values))
	for _, value := range values {
		valArr := make([]string, 0, len(columns))
		for j, column := range columns {
			val := dbi.GetDbDataType(DbTypeDM, column.DataType).DataType.SQLValue(value[j])
			valArr = append(valArr, fmt.Sprintf("%s %s", val, quote(column.ColumnName)))
		}
		valueSql = append(valueSql, fmt.Sprintf("SELECT %s FROM dual", strings.Join(valArr, ", ")))
	}
	t2 := strings.Join(valueSql, " UNION ALL ")

	sqlTemp := "MERGE INTO " + quote(tableName) + " T1 USING (" + t2 + ") T2 ON " + strings.Join(caseSqls, " OR ")
	sqlTemp += "WHEN NOT MATCHED THEN INSERT (" + strings.Join(insertCols, ",") + ") VALUES (" + strings.Join(insertVals, ",") + ")"
	sqlTemp += "WHEN MATCHED THEN UPDATE SET " + strings.Join(upds, ",")

	return collx.AsArray(sqlTemp)
}

// genSimpleInserts 生成直接插入语句，达梦只能一条条执行insert，所以将values拆分为多条insert语句
func (sg *SQLGenerator) genSimpleInserts(tableName string, columns []dbi.Column, values [][]any) []string {
	quoter := sg.Dialect.Quoter()
	quote := quoter.QuoteIdent

	var res []string
	var hasIdentity = false
	identityInsertOn := ""
	identityInsertOff := ""
	// 有自增列的才加上这个语句
	if collx.AnyMatch(columns, func(column dbi.Column) bool { return column.AutoIncrement }) {
		identityInsertOn = fmt.Sprintf("set identity_insert %s on;", quote(tableName))
		identityInsertOff = fmt.Sprintf("set identity_insert %s off;", quote(tableName))
		hasIdentity = true
		res = append(res, identityInsertOn)
	}

	// 达梦数据库只能一条条的执行insert语句，所以这里需要将values拆分成多条insert语句
	sqls := collx.ArrayMap(values, func(value []any) string {
		columnStr, valuesStrs := dbi.GenInsertSqlColumnAndValues(sg.Dialect, DbTypeDM, columns, [][]any{value})
		return fmt.Sprintf("insert into %s %s values %s", quote(tableName), columnStr, valuesStrs[0])
	})

	res = append(res, sqls...)

	if hasIdentity {
		res = append(res, identityInsertOff)
	}
	return res
}

func (sg *SQLGenerator) genColumnBasicSql(quoter dbi.Quoter, column dbi.Column) string {
	incr := ""
	if column.AutoIncrement {
		incr = " IDENTITY"
	}

	nullAble := ""
	if !column.Nullable {
		nullAble = " NOT NULL"
	}

	colName := quoter.QuoteIdent(column.ColumnName)
	// 达梦的 data_default 保留书写的引号与双写转义（字面量形态），旧实现“含左括号即丢弃”
	// 会使 '(0)'、'unknown (pending)' 这类默认值静默丢失；源库为MySQL 8.0时默认值是不带引号的
	// 原始值，故必须用宽松版按字面量重新引用，不能因形态陌生而丢弃
	defVal := dbi.GenColumnDefaultSqlOf(&column, column.DataType, dbi.QuoteEscape)

	columnSql := fmt.Sprintf(" %s %s%s%s%s", colName, column.GetColumnType(), incr, nullAble, defVal)
	return columnSql
}

package mssql

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"strings"
)

type SQLGenerator struct {
	dc *dbi.DbConn
}

// quoteTableName 生成带schema的表名，如 [schema].[table]；schema为空（如仅生成SQL的伪连接场景）时退化为 [table]
func (sg *SQLGenerator) quoteTableName(quote func(string) string, tableName string) string {
	if schema := sg.dc.Info.CurrentSchema(); schema != "" {
		return fmt.Sprintf("%s.%s", quote(schema), quote(tableName))
	}
	return quote(tableName)
}

func (sg *SQLGenerator) GenTableDDL(table dbi.Table, columns []dbi.Column, dropBeforeCreate bool) []string {
	tbName := table.TableName
	quoter := sg.dc.GetDialect().Quoter()
	quote := quoter.Quote
	quoteTable := sg.quoteTableName(quote, tbName)

	sqlArr := make([]string, 0)

	// 删除表
	if dropBeforeCreate {
		sqlArr = append(sqlArr, fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteTable))
	}

	// 组装建表语句
	createSql := fmt.Sprintf("CREATE TABLE %s (\n", quoteTable)
	fields := make([]string, 0)
	pks := make([]string, 0)
	columnComments := make([]string, 0)

	for _, column := range columns {
		if column.IsPrimaryKey {
			pks = append(pks, quote(column.ColumnName))
		}
		fields = append(fields, sg.genColumnBasicSql(quoter, column))
		commentTmp := "EXECUTE sp_addextendedproperty N'MS_Description', N'%s', N'SCHEMA', N'%s', N'TABLE', N'%s', N'COLUMN', N'%s'"

		// 防止注释内含有特殊字符串导致sql出错
		if column.ColumnComment != "" {
			comment := dbi.QuoteEscape(column.ColumnComment)
			columnComments = append(columnComments, fmt.Sprintf(commentTmp, comment, sg.dc.Info.CurrentSchema(), tbName, column.ColumnName))
		}
	}

	// create
	createSql += strings.Join(fields, ",\n")
	if len(pks) > 0 {
		createSql += fmt.Sprintf(", \n PRIMARY KEY CLUSTERED (%s)", strings.Join(pks, ","))
	}
	createSql += "\n)"

	// comment
	tableCommentSql := ""
	if table.TableComment != "" {
		commentTmp := "EXECUTE sp_addextendedproperty N'MS_Description', N'%s', N'SCHEMA', N'%s', N'TABLE', N'%s'"

		tableCommentSql = fmt.Sprintf(commentTmp, dbi.QuoteEscape(table.TableComment), sg.dc.Info.CurrentSchema(), tbName)
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
	quote := sg.dc.GetDialect().Quoter().Quote
	tbName := table.TableName
	sqls := make([]string, 0)
	comments := make([]string, 0)
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

		sqls = append(sqls, fmt.Sprintf("create %s NONCLUSTERED index %s on %s(%s)", unique, quote(index.IndexName), sg.quoteTableName(quote, tbName), strings.Join(colNames, ",")))
		if index.IndexComment != "" {
			comment := dbi.QuoteEscape(index.IndexComment)
			comments = append(comments, fmt.Sprintf("EXECUTE sp_addextendedproperty N'MS_Description', N'%s', N'SCHEMA', N'%s', N'TABLE', N'%s', N'INDEX', N'%s'", comment, sg.dc.Info.CurrentSchema(), tbName, index.IndexName))
		}
	}
	if len(comments) > 0 {
		sqls = append(sqls, comments...)
	}

	return sqls
}

func (sg *SQLGenerator) GenInsert(tableName string, columns []dbi.Column, values [][]any, duplicateStrategy int, targetTableMeta *dbi.TargetTableMeta) []string {

	if duplicateStrategy == dbi.DuplicateStrategyUpdate {
		return sg.batchInsertMerge(tableName, columns, values, duplicateStrategy)
	}

	return sg.batchInsertSimple(tableName, columns, values, duplicateStrategy, targetTableMeta)
}

func (sg *SQLGenerator) batchInsertSimple(tableName string, columns []dbi.Column, values [][]any, duplicateStrategy int, targetTableMeta *dbi.TargetTableMeta) []string {
	var res []string

	// 把二维数组转为一维数组
	var args []any
	var singleSize int // 一条数据的参数个数
	for i, v := range values {
		if i == 0 {
			singleSize = len(v)
		}
		args = append(args, v...)
	}

	// 判断如果参数超过2000，则分批次执行，mssql允许最大参数为2100，保险起见，这里限制到2000
	if len(args) > 2000 {

		rows := 2000 / singleSize // 每批次最大数据条数
		mp := make(map[any][][]any)

		// 把values拆成多份，每份不能超过rows条
		length := len(values)
		for i := 0; i < length; i += rows {
			if i+rows <= length {
				mp[i] = values[i : i+rows]
			} else {
				mp[i] = values[i:length]
			}
		}

		var strs []string
		for _, v := range mp {
			res := sg.batchInsertSimple(tableName, columns, v, duplicateStrategy, targetTableMeta)
			strs = append(strs, res...)
		}
		return strs
	}

	ignoreDupSql := ""
	if duplicateStrategy == dbi.DuplicateStrategyIgnore {
		// 收集唯一索引涉及到的字段（由调用方预查传入，生成过程中不查询数据库）
		uniqueColumns := make([]string, 0)
		if targetTableMeta != nil {
			for _, col := range targetTableMeta.UniqueColumns {
				if !collx.ArrayContains(uniqueColumns, col) {
					uniqueColumns = append(uniqueColumns, col)
				}
			}
		}
		if len(uniqueColumns) > 0 {
			// 设置忽略重复键
			// ALTER TABLE dbo.TEST ADD CONSTRAINT uniqueRows UNIQUE (ColA, ColB, ColC, ColD) WITH (IGNORE_DUP_KEY = ON)
			ignoreDupSql = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT uniqueRows UNIQUE (%s) WITH (IGNORE_DUP_KEY = {sign})", sg.quoteTableName(sg.dc.GetDialect().Quoter().Quote, tableName), strings.Join(uniqueColumns, ","))
			res = append(res, strings.ReplaceAll(ignoreDupSql, "{sign}", "ON"))
		}
	}

	quote := sg.dc.GetDialect().Quoter().Quote
	baseTable := sg.quoteTableName(quote, tableName)

	// 设置允许填充自增列之后，显示指定列名可以插入自增列
	identityInsertOn := ""
	for _, column := range columns {
		if column.AutoIncrement {
			identityInsertOn = fmt.Sprintf("SET IDENTITY_INSERT %s ON", baseTable)
		}
	}

	columnStr, valuesStrs := dbi.GenInsertSqlColumnAndValues(sg.dc.GetDialect(), DbTypeMssql, columns, values)
	insertSql := fmt.Sprintf("%s insert into %s %s VALUES \n%s", identityInsertOn, baseTable, columnStr, strings.Join(valuesStrs, ",\n"))
	res = append(res, insertSql)

	// 执行完之后，设置忽略重复键
	if ignoreDupSql != "" {
		res = append(res, strings.ReplaceAll(ignoreDupSql, "{sign}", "OFF"))
	}
	return res
}

func (sg *SQLGenerator) batchInsertMerge(tableName string, columns []dbi.Column, values [][]any, duplicateStrategy int) []string {
	var res []string
	quote := sg.dc.GetDialect().Quoter().Quote

	// 收集MERGE 语句的 ON 子句条件
	caseArr := make([]string, 0)
	// 搜集主键字段
	pkCols := make([]string, 0)
	// 查询取出自增列字段, merge update不能修改自增列
	identityCols := make([]string, 0)
	// 标记是否有自增字段
	hashIdentity := false

	for _, col := range columns {
		if col.AutoIncrement {
			hashIdentity = true
			identityCols = append(identityCols, col.ColumnName)
		}
		if col.IsPrimaryKey {
			pkCols = append(pkCols, col.ColumnName)
			name := quote(col.ColumnName)
			caseArr = append(caseArr, fmt.Sprintf(" T1.%s = T2.%s ", name, name))
		}
	}
	if len(pkCols) == 0 {
		// 无主键无法生成merge语句，退化为简单插入（含ignore策略处理）
		return sg.batchInsertSimple(tableName, columns, values, duplicateStrategy, nil)
	}
	// 重复数据处理策略
	updSqls := make([]string, 0)
	insertVals := make([]string, 0)
	insertCols := make([]string, 0)

	for _, column := range columns {
		columnName := column.ColumnName
		quoteName := quote(columnName)
		if !collx.ArrayContains(identityCols, sg.dc.GetDialect().Quoter().Trim(columnName)) {
			updSqls = append(updSqls, fmt.Sprintf("T1.%s = T2.%s", columnName, columnName))
		}
		insertCols = append(insertCols, quoteName)
		insertVals = append(insertVals, fmt.Sprintf("T2.%s", columnName))
	}

	// 把values二维数组转为一维数组
	valueSql := make([]string, 0)
	for _, value := range values {
		// 注意：valArr必须每行重新收集，否则多行时前一行的值会累积到后续行导致参数错位
		valArr := make([]string, 0, len(columns))
		for j, column := range columns {
			val := dbi.GetDbDataType(DbTypeMssql, column.DataType).DataType.SQLValue(value[j])
			valArr = append(valArr, fmt.Sprintf("%s %s", val, column.ColumnName))
		}
		valueSql = append(valueSql, fmt.Sprintf("select %s", strings.Join(valArr, ", ")))
	}

	quoteTable := sg.quoteTableName(quote, tableName)
	unionSql := strings.Join(valueSql, " UNION ALL ")
	caseSql := strings.Join(caseArr, " AND ")

	sqlTemp := "MERGE INTO " + quoteTable + " T1 USING (" + unionSql + ") T2 ON " + caseSql
	sqlTemp += "WHEN NOT MATCHED THEN INSERT (" + strings.Join(insertCols, ",") + ") VALUES (" + strings.Join(insertVals, ",") + ") "
	sqlTemp += "WHEN MATCHED THEN UPDATE SET " + strings.Join(updSqls, ",")

	identityInsertOn := ""
	if hashIdentity {
		// 设置允许填充自增列之后，显示指定列名可以插入自增列
		identityInsertOn = fmt.Sprintf("SET IDENTITY_INSERT %s ON", quoteTable)

	}
	// 执行merge sql,必须要以分号结尾
	res = append(res, fmt.Sprintf("%s %s", identityInsertOn, sqlTemp))

	return res
}

func (sg *SQLGenerator) genColumnBasicSql(quoter dbi.Quoter, column dbi.Column) string {
	colName := quoter.Quote(column.ColumnName)
	dataType := column.DataType

	incr := ""
	if column.AutoIncrement {
		incr = " IDENTITY(1,1)"
	}

	nullAble := ""
	if !column.Nullable {
		nullAble = " NOT NULL"
	}

	defVal := "" // 默认值需要判断引号，如函数是不需要引号的 // 为了防止跨源函数不支持 当默认值是函数时，不需要设置默认值
	if column.ColumnDefault != "" && !strings.Contains(column.ColumnDefault, "(") {
		// 哪些字段类型默认值需要加引号
		mark := false
		if collx.ArrayAnyMatches([]string{"char", "text", "date", "time", "lob"}, dataType) {
			// 当数据类型是日期时间，默认值是日期时间函数时，默认值不需要引号
			if collx.ArrayAnyMatches([]string{"date", "time"}, strings.ToLower(dataType)) &&
				collx.ArrayAnyMatches([]string{"DATE", "TIME"}, strings.ToUpper(column.ColumnDefault)) {
				mark = false
			} else {
				mark = true
			}
		}

		if mark {
			// 默认值可能含单引号（如 it's），需双写转义，避免 DDL 语法错误或注入
			defVal = fmt.Sprintf(" DEFAULT '%s'", dbi.QuoteEscape(column.ColumnDefault))
		} else {
			defVal = fmt.Sprintf(" DEFAULT %s", column.ColumnDefault)
		}
	}

	columnSql := fmt.Sprintf(" %s %s%s%s%s", colName, column.GetColumnType(), incr, nullAble, defVal)
	return columnSql
}

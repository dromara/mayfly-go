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

func (sg *SQLGenerator) GenTableDDL(table dbi.Table, columns []dbi.Column, dropBeforeCreate bool) []string {
	tbName := table.TableName
	schemaName := sg.dc.Info.CurrentSchema()

	quoter := sg.dc.GetDialect().Quoter()
	quote := quoter.Quote

	sqlArr := make([]string, 0)

	// 删除表
	if dropBeforeCreate {
		sqlArr = append(sqlArr, fmt.Sprintf("DROP TABLE IF EXISTS %s.%s", quote(schemaName), quote(tbName)))
	}

	// 组装建表语句
	createSql := fmt.Sprintf("CREATE TABLE %s.%s (\n", quote(schemaName), quote(tbName))
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

		sqls = append(sqls, fmt.Sprintf("create %s NONCLUSTERED index %s on %s.%s(%s)", unique, quote(index.IndexName), quote(sg.dc.Info.CurrentSchema()), quote(tbName), strings.Join(colNames, ",")))
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

func (sg *SQLGenerator) GenInsert(tableName string, columns []dbi.Column, values [][]any, duplicateStrategy int) []string {

	if duplicateStrategy == dbi.DuplicateStrategyUpdate {
		return sg.batchInsertMerge(tableName, columns, values, duplicateStrategy)
	}

	return sg.batchInsertSimple(tableName, columns, values, duplicateStrategy)
}

func (sg *SQLGenerator) batchInsertSimple(tableName string, columns []dbi.Column, values [][]any, duplicateStrategy int) []string {
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
			res := sg.batchInsertSimple(tableName, columns, v, duplicateStrategy)
			strs = append(strs, res...)
		}
		return strs
	}

	msMetadata := sg.dc.GetMetadata()
	schema := sg.dc.Info.CurrentSchema()
	ignoreDupSql := ""
	if duplicateStrategy == dbi.DuplicateStrategyIgnore {
		// ALTER TABLE dbo.TEST ADD CONSTRAINT uniqueRows UNIQUE (ColA, ColB, ColC, ColD) WITH (IGNORE_DUP_KEY = ON)
		indexs, _ := msMetadata.(*MssqlMetadata).getTableIndexWithPK(tableName)
		// 收集唯一索引涉及到的字段
		uniqueColumns := make([]string, 0)
		for _, index := range indexs {
			if index.IsUnique {
				cols := strings.Split(index.ColumnName, ",")
				for _, col := range cols {
					if !collx.ArrayContains(uniqueColumns, col) {
						uniqueColumns = append(uniqueColumns, col)
					}
				}
			}
		}
		if len(uniqueColumns) > 0 {
			// 设置忽略重复键
			ignoreDupSql = fmt.Sprintf("ALTER TABLE %s.%s ADD CONSTRAINT uniqueRows UNIQUE (%s) WITH (IGNORE_DUP_KEY = {sign})", schema, tableName, strings.Join(uniqueColumns, ","))
			res = append(res, strings.ReplaceAll(ignoreDupSql, "{sign}", "ON"))
		}
	}

	// 生成占位符字符串：如：(?,?)
	// 重复字符串并用逗号连接
	repeated := strings.Repeat("?,", len(columns))
	// 去除最后一个逗号，占位符由括号包裹
	placeholder := fmt.Sprintf("(%s)", strings.TrimSuffix(repeated, ","))

	// 重复占位符字符串n遍
	repeated = strings.Repeat(placeholder+",", len(values))
	// 去除最后一个逗号
	placeholder = strings.TrimSuffix(repeated, ",")

	quote := sg.dc.GetDialect().Quoter().Quote
	baseTable := fmt.Sprintf("%s.%s", quote(schema), quote(tableName))

	// 设置允许填充自增列之后，显示指定列名可以插入自增列
	identityInsertOn := ""
	for _, column := range columns {
		if column.AutoIncrement {
			identityInsertOn = fmt.Sprintf("SET IDENTITY_INSERT [%s].[%s] ON", schema, tableName)
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
	schema := sg.dc.Info.CurrentSchema()
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
		return sg.batchInsertSimple(tableName, columns, values, duplicateStrategy)
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
	valArr := make([]string, 0)

	valueSql := make([]string, 0)
	for _, value := range values {
		for j, column := range columns {
			val := dbi.GetDbDataType(DbTypeMssql, column.DataType).DataType.SQLValue(value[j])
			valArr = append(valArr, fmt.Sprintf("%s %s", val, column.ColumnName))
		}
		valueSql = append(valueSql, fmt.Sprintf("select %s", strings.Join(valArr, ", ")))
	}

	quoteTable := fmt.Sprintf("%s.%s", quote(schema), quote(tableName))
	unionSql := strings.Join(valueSql, " UNION ALL ")
	caseSql := strings.Join(caseArr, " AND ")

	sqlTemp := "MERGE INTO " + quoteTable + " T1 USING (" + unionSql + ") T2 ON " + caseSql
	sqlTemp += "WHEN NOT MATCHED THEN INSERT (" + strings.Join(insertCols, ",") + ") (" + strings.Join(insertVals, ",") + ") "
	sqlTemp += "WHEN MATCHED THEN UPDATE SET " + strings.Join(updSqls, ",")

	identityInsertOn := ""
	if hashIdentity {
		// 设置允许填充自增列之后，显示指定列名可以插入自增列
		identityInsertOn = fmt.Sprintf("SET IDENTITY_INSERT [%s].[%s] ON", schema, tableName)

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
			defVal = fmt.Sprintf(" DEFAULT '%s'", column.ColumnDefault)
		} else {
			defVal = fmt.Sprintf(" DEFAULT %s", column.ColumnDefault)
		}
	}

	columnSql := fmt.Sprintf(" %s %s%s%s%s", colName, column.GetColumnType(), incr, nullAble, defVal)
	return columnSql
}

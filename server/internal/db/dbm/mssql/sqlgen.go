package mssql

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"strings"
)

var _ dbi.SQLGenerator = (*SQLGenerator)(nil)

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
	quote := quoter.QuoteIdent
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
	quote := sg.dc.GetDialect().Quoter().QuoteIdent
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
		var strs []string
		// 按原始顺序切片分批，保证批次顺序与行序确定性；
		// map分批会因迭代随机导致插入行序不确定
		for i := 0; i < len(values); i += rows {
			end := i + rows
			if end > len(values) {
				end = len(values)
			}
			batchRes := sg.batchInsertSimple(tableName, columns, values[i:end], duplicateStrategy, targetTableMeta)
			strs = append(strs, batchRes...)
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
			ignoreDupSql = fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT uniqueRows UNIQUE (%s) WITH (IGNORE_DUP_KEY = {sign})", sg.quoteTableName(sg.dc.GetDialect().Quoter().QuoteIdent, tableName), strings.Join(uniqueColumns, ","))
			res = append(res, strings.ReplaceAll(ignoreDupSql, "{sign}", "ON"))
		}
	}

	quote := sg.dc.GetDialect().Quoter().QuoteIdent
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
	quote := sg.dc.GetDialect().Quoter().QuoteIdent

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
		// 计算列/生成列既不可插入也不可更新（SQL Server报“不能向计算列插入值”），
		// 其值由表达式派生，写入INSERT/UPDATE子句会使整条merge失败
		if dbi.PreservableGeneratedColumn(column, DbTypeMssql) || column.IsGenerated {
			continue
		}
		if !collx.ArrayContains(identityCols, sg.dc.GetDialect().Quoter().Trim(columnName)) {
			// update子句中的列名需引用，避免保留字/特殊字符列名导致语法错误
			updSqls = append(updSqls, fmt.Sprintf("T1.%s = T2.%s", quoteName, quoteName))
		}
		insertCols = append(insertCols, quoteName)
		insertVals = append(insertVals, fmt.Sprintf("T2.%s", quoteName))
	}
	if len(insertCols) == 0 || len(updSqls) == 0 {
		// 除计算列/自增列/主键列外无可写入列，无法生成merge语句，退化为简单插入
		return sg.batchInsertSimple(tableName, columns, values, duplicateStrategy, nil)
	}

	// 把values二维数组转为一维数组
	valueSql := make([]string, 0)
	for _, value := range values {
		// 注意：valArr必须每行重新收集，否则多行时前一行的值会累积到后续行导致参数错位
		valArr := make([]string, 0, len(columns))
		for j, column := range columns {
			val := dbi.GetDbDataType(DbTypeMssql, column.DataType).DataType.SQLValue(value[j])
			// select别名列名需引用，与T2.列名引用保持一致
			valArr = append(valArr, fmt.Sprintf("%s %s", val, quote(column.ColumnName)))
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
	// MERGE语句必须以分号结尾，否则报Msg 10713（A MERGE statement must be terminated by a semi-colon）
	mergeSql := sqlTemp + ";"
	if identityInsertOn != "" {
		mergeSql = identityInsertOn + "\n" + mergeSql
	}
	res = append(res, mergeSql)

	return res
}

func (sg *SQLGenerator) genColumnBasicSql(quoter dbi.Quoter, column dbi.Column) string {
	colName := quoter.QuoteIdent(column.ColumnName)
	dataType := column.DataType

	// 计算列（仅源与目标同为SQL Server时才能重建，定义原文属于T-SQL）：形态为「col AS (expr) [PERSISTED]」，
	// 不接受类型/NULL/DEFAULT/IDENTITY声明（声明即语法错误）；
	// 不重建则目标列退化为普通列，与INSERT阶段的剔除叠加会使计算列值静默变NULL
	if dbi.PreservableGeneratedColumn(column, DbTypeMssql) {
		expr := strings.TrimSpace(dbi.GeneratedColumnExpr(column))
		if !strings.HasPrefix(expr, "(") {
			expr = "(" + expr + ")"
		}
		persisted := ""
		if dbi.GeneratedColumnStored(column) {
			persisted = " PERSISTED"
		}
		return fmt.Sprintf(" %s AS %s%s", colName, expr, persisted)
	}

	incr := ""
	if column.AutoIncrement {
		incr = " IDENTITY(1,1)"
	}

	nullAble := ""
	if !column.Nullable {
		nullAble = " NOT NULL"
	}

	// SQL Server的 object_definition 返回带最外层括号的定义原文（如 ('abc')、(N'abc')、(3)、(getdate())），
	// 旧实现直接按含括号判定为函数而丢弃，导致所有默认值在结构迁移时静默丢失；
	// 统一由dbi做括号平衡剔除与分类（内层字面量还原、表达式跳过），源库为MySQL 8.0的裸值也能存活
	defVal := dbi.GenColumnDefaultSqlOf(&column, dataType, dbi.QuoteEscape)

	columnSql := fmt.Sprintf(" %s %s%s%s%s", colName, column.GetColumnType(), incr, nullAble, defVal)
	return columnSql
}

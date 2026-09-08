package oracle

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
	quoteTableName := quote(table.TableName)
	sqlArr := make([]string, 0)

	if dropBeforeCreate {
		dropSqlTmp := `
declare
      num number;
begin
    select count(1) into num from user_tables where table_name = '%s' and owner = (SELECT sys_context('USERENV', 'CURRENT_SCHEMA') FROM dual) ;
    if num > 0 then
        execute immediate 'drop table "%s"' ;
    end if;
end`
		sqlArr = append(sqlArr, fmt.Sprintf(dropSqlTmp, table.TableName, table.TableName))
	}

	// 组装建表语句
	createSql := fmt.Sprintf("CREATE TABLE %s ( \n", quoteTableName)
	fields := make([]string, 0)
	pks := make([]string, 0)
	columnComments := make([]string, 0)
	// 把通用类型转换为达梦类型
	for _, column := range columns {
		if column.IsPrimaryKey {
			pks = append(pks, quote(column.ColumnName))
		}
		quote := quoter.QuoteIdent
		fields = append(fields, sg.genColumnBasicSql(quoter, column))
		// 防止注释内含有特殊字符串导致sql出错
		if column.ColumnComment != "" {
			comment := dbi.QuoteEscape(column.ColumnComment)
			columnComments = append(columnComments, fmt.Sprintf("COMMENT ON COLUMN %s.%s IS '%s'", quoteTableName, quote(column.ColumnName), comment))
		}
	}

	// 建表
	createSql += strings.Join(fields, ",\n")
	if len(pks) > 0 {
		createSql += fmt.Sprintf(", \nPRIMARY KEY (%s)", strings.Join(pks, ","))
	}
	createSql += "\n)"
	sqlArr = append(sqlArr, createSql)

	// 表注释
	tableCommentSql := ""
	if table.TableComment != "" {
		tableCommentSql = fmt.Sprintf("COMMENT ON TABLE %s is '%s'", quote(table.TableName), dbi.QuoteEscape(table.TableComment))
		sqlArr = append(sqlArr, tableCommentSql)
	}

	// 列注释
	if len(columnComments) > 0 {
		sqlArr = append(sqlArr, columnComments...)
	}

	otherSql := sg.GenerateTableOtherDDL(table, quoteTableName, columns)
	if len(otherSql) > 0 {
		sqlArr = append(sqlArr, otherSql...)
	}

	return sqlArr
}

func (sg *SQLGenerator) GenIndexDDL(table dbi.Table, indexs []dbi.Index) []string {
	sqls := make([]string, 0)
	comments := make([]string, 0)
	quote := sg.Dialect.Quoter().QuoteIdent
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

		sqls = append(sqls, fmt.Sprintf("CREATE %s INDEX %s ON %s(%s)", unique, quote(index.IndexName), quote(table.TableName), strings.Join(colNames, ",")))
	}

	sqlArr := make([]string, 0)

	sqlArr = append(sqlArr, sqls...)

	if len(comments) > 0 {
		sqlArr = append(sqlArr, comments...)
	}

	return sqlArr
}

func (sg *SQLGenerator) GenInsert(tableName string, columns []dbi.Column, values [][]any, duplicateStrategy int, targetTableMeta *dbi.TargetTableMeta) []string {
	if duplicateStrategy != dbi.DuplicateStrategyUpdate || targetTableMeta == nil || len(targetTableMeta.UniqueColumns) == 0 {
		// 直接插入（无法生成 merge 语句时也退化为直接插入，避免静默丢失数据由数据库主键约束报错提示）
		return sg.genSimpleInserts(tableName, columns, values)
	}

	quoter := sg.Dialect.Quoter()
	quote := quoter.QuoteIdent

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
		// 标识列（GENERATED ALWAYS AS IDENTITY）既不可插入也不可更新（ORA-32792），直接跳过
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
			val := dbi.GetDbDataType(DbTypeOracle, column.DataType).DataType.SQLValue(value[j])
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

// genSimpleInserts 生成直接插入语句，oracle/达梦类数据库只能一条条执行insert，所以将values拆分为多条insert语句
func (sg *SQLGenerator) genSimpleInserts(tableName string, columns []dbi.Column, values [][]any) []string {
	quote := sg.Dialect.Quoter().QuoteIdent
	identityInsert := ""
	// 有自增列的才加上这个语句
	if collx.AnyMatch(columns, func(column dbi.Column) bool { return column.AutoIncrement }) {
		identityInsert = fmt.Sprintf("set identity_insert %s on;", quote(tableName))
	}

	// oracle数据库只能一条条的执行insert语句，所以这里需要将values拆分成多条insert语句
	// 注意：columnStr已带括号（如("id", "name")），不能再包一层括号，否则生成非法的双括号语句
	return collx.ArrayMap(values, func(value []any) string {
		columnStr, valuesStrs := dbi.GenInsertSqlColumnAndValues(sg.Dialect, DbTypeOracle, columns, [][]any{value})
		return fmt.Sprintf("%s insert into %s %s values %s", identityInsert, quote(tableName), columnStr, strings.Join(valuesStrs, ",\n"))
	})
}

func (msg *SQLGenerator) genColumnBasicSql(quoter dbi.Quoter, column dbi.Column) string {
	colName := quoter.QuoteIdent(column.ColumnName)

	if column.AutoIncrement {
		// 如果是自增，不需要设置默认值和空值，自增列数据类型必须是number
		return fmt.Sprintf(" %s NUMBER generated by default as IDENTITY", colName)
	}

	nullAble := ""
	if !column.Nullable {
		nullAble = " NOT NULL"
	}

	// Oracle的 data_default 保留书写的引号与双写转义（字面量形态），旧实现未作任何判定直接裸拼：
	// 既会把 to_char(sysdate,...) 等函数表达式原样带到不支持它的目标库，也不会对默认值内容
	// 做任何引用/转义校验，统一由dbi按字面量语义处理（函数与无法判定的括号表达式跳过）
	defVal := dbi.GenColumnDefaultSqlOf(&column, column.DataType, dbi.QuoteEscape)

	columnSql := fmt.Sprintf(" %s %s%s%s", colName, column.GetColumnType(), defVal, nullAble)
	return columnSql
}

// 11g及以下版本会设置自增序列
func (sg *SQLGenerator) GenerateTableOtherDDL(tableInfo dbi.Table, quoteTableName string, columns []dbi.Column) []string {
	return nil
}

// 11g及以下版本会设置自增序列和触发器
func (sg *SQLGenerator) Oracle11GenerateTableOtherDDL(tableInfo dbi.Table, quoteTableName string, columns []dbi.Column) []string {
	result := make([]string, 0)
	for _, col := range columns {
		if col.AutoIncrement {
			seqName := fmt.Sprintf("%s_%s_seq", tableInfo.TableName, col.ColumnName)
			trgName := fmt.Sprintf("%s_%s_trg", tableInfo.TableName, col.ColumnName)
			result = append(result, fmt.Sprintf("CREATE SEQUENCE %s START WITH 1 INCREMENT BY 1", seqName))
			result = append(result, fmt.Sprintf("CREATE OR REPLACE TRIGGER %s BEFORE INSERT ON %s FOR EACH ROW WHEN (NEW.%s IS NULL) BEGIN SELECT %s.nextval INTO :new.%s FROM dual; END", trgName, quoteTableName, col.ColumnName, seqName, col.ColumnName))
		}
	}

	return result
}

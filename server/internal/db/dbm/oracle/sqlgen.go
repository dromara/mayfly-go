package oracle

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/utils/collx"
	"strings"
)

type SQLGenerator struct {
	Dialect  dbi.Dialect
	Metadata dbi.Metadata
}

func (sg *SQLGenerator) GenTableDDL(table dbi.Table, columns []dbi.Column, dropBeforeCreate bool) []string {
	quoter := sg.Dialect.Quoter()
	quote := quoter.Quote
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
		quote := quoter.Quote
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
	quote := sg.Dialect.Quoter().Quote
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

func (sg *SQLGenerator) GenInsert(tableName string, columns []dbi.Column, values [][]any, duplicateStrategy int) []string {
	quoter := sg.Dialect.Quoter()
	quote := quoter.Quote

	if duplicateStrategy == dbi.DuplicateStrategyNone {
		identityInsert := fmt.Sprintf("set identity_insert %s on;", quote(tableName))

		// 达梦数据库只能一条条的执行insert语句，所以这里需要将values拆分成多条insert语句
		return collx.ArrayMap(values, func(value []any) string {
			columnStr, valuesStrs := dbi.GenInsertSqlColumnAndValues(sg.Dialect, DbTypeOracle, columns, [][]any{value})
			return fmt.Sprintf("%s insert into %s (%s) values %s", identityInsert, quote(tableName), columnStr, strings.Join(valuesStrs, ",\n"))
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

func (msg *SQLGenerator) genColumnBasicSql(quoter dbi.Quoter, column dbi.Column) string {
	colName := quoter.Quote(column.ColumnName)

	if column.IsIdentity {
		// 如果是自增，不需要设置默认值和空值，自增列数据类型必须是number
		return fmt.Sprintf(" %s NUMBER generated by default as IDENTITY", colName)
	}

	nullAble := ""
	if !column.Nullable {
		nullAble = " NOT NULL"
	}

	defVal := ""
	if column.ColumnDefault != "" {
		defVal = fmt.Sprintf(" DEFAULT %v", column.ColumnDefault)
	}

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
		if col.IsIdentity {
			seqName := fmt.Sprintf("%s_%s_seq", tableInfo.TableName, col.ColumnName)
			trgName := fmt.Sprintf("%s_%s_trg", tableInfo.TableName, col.ColumnName)
			result = append(result, fmt.Sprintf("CREATE SEQUENCE %s START WITH 1 INCREMENT BY 1", seqName))
			result = append(result, fmt.Sprintf("CREATE OR REPLACE TRIGGER %s BEFORE INSERT ON %s FOR EACH ROW WHEN (NEW.%s IS NULL) BEGIN SELECT %s.nextval INTO :new.%s FROM dual; END", trgName, quoteTableName, col.ColumnName, seqName, col.ColumnName))
		}
	}

	return result
}

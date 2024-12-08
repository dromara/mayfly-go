package dbi

import (
	"fmt"
	"mayfly-go/pkg/logx"
	"strings"
)

type StmtType string

const (
	StmtTypeSelect StmtType = "select"
	StmtTypeInsert StmtType = "insert"
	StmtTypeUpdate StmtType = "update"
	StmtTypeDelete StmtType = "delete"
	StmtTypeDDL    StmtType = "ddl"
)

// GenTableDDL 生成通用表DDL
func GenTableDDL(dialect Dialect, md Metadata, tableName string, dropBeforeCreate bool) (string, error) {
	// 1.获取表信息
	tbs, err := md.GetTables(tableName)
	if len(tbs) == 0 {
		logx.Errorf("get table error: %s", tableName)
		return "", err
	}
	table := tbs[0]

	// 2.获取列信息
	columns, err := md.GetColumns(tableName)
	if err != nil {
		logx.Errorf("get columns error: %s", tableName)
		return "", err
	}

	sqlGenerator := dialect.GetSQLGenerator()

	tableDDLArr := sqlGenerator.GenTableDDL(table, columns, dropBeforeCreate)
	// 3.获取索引信息
	indexs, err := md.GetTableIndex(tableName)
	if err != nil {
		logx.Errorf("get indexs error: %s", tableName)
		return "", err
	}

	// 组装返回
	tableDDLArr = append(tableDDLArr, sqlGenerator.GenIndexDDL(table, indexs)...)
	return strings.Join(tableDDLArr, ";\n"), nil
}

// GenCommonInsert 生成通用insert sql
//
//	insert into table_name (column1, column2, ...) values (value1, value2, ...), (value1, value2, ...), ...
func GenCommonInsert(dialect Dialect, dbType DbType, tableName string, columns []Column, values [][]any) string {
	quote := dialect.Quoter().Quote
	columnStr, valuesStrs := GenInsertSqlColumnAndValues(dialect, dbType, columns, values)

	// insert into table_name (column1, column2, ...) values (value1, value2, ...), (value1, value2, ...), ...
	return fmt.Sprintf("INSERT INTO %s %s VALUES \n%s", quote(tableName), columnStr, strings.Join(valuesStrs, ",\n"))
}

// GenInsertSqlColumnAndValues 生成insert sql对应的 columes信息和values信息
//
//	columnsStr -> (column1, column2, column3, ...)
//	valuesStrs -> ['(value1, value2, value3, ...)', '(value1, value2, value3, ...)', ...]
func GenInsertSqlColumnAndValues(dialect Dialect, dbType DbType, columns []Column, values [][]any) (columnsStr string, valuesStrs []string) {
	quote := dialect.Quoter().Quote

	columnNames := make([]string, 0, len(columns))
	columnTypes := make([]*DbDataType, len(columns))

	strValueArr := make([]string, 0, len(values))

	for i, column := range columns {
		columnNames = append(columnNames, quote(column.ColumnName))
		columnType := GetDbDataType(dbType, column.DataType)
		columnTypes[i] = columnType
	}

	for _, value := range values {
		vs := make([]string, 0, len(value))
		for i, v := range value {
			vs = append(vs, columnTypes[i].DataType.SQLValue(v))
		}
		strValueArr = append(strValueArr, fmt.Sprintf("(%s)", strings.Join(vs, ", ")))
	}

	return fmt.Sprintf("(%s)", strings.Join(columnNames, ", ")), strValueArr
}

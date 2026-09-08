package dbi

import (
	"fmt"
	"mayfly-go/pkg/errorx"
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

// OmitGeneratedColumns 从插入列集与对应行值中剔除「在目标方言下已被重建为生成列」的列：
// 生成列的值由表达式派生，显式插入必然报错（MySQL Error 3105、pg Error 3402、SQL Server“不能向计算列插入值”），
// 会使整表数据迁移/导出导入全量失败。
//
// 仅剔除 PreservableGeneratedColumn 成立的列（目标表该列确实为生成列）：若目标列只能退化为普通列
// （无派生表达式或跨方言），则必须插入源值，否则目标列静默变NULL造成数据丢失。
// 无生成列时直接返回原切片，避免不必要的拷贝
func OmitGeneratedColumns(columns []Column, values [][]any, targetDbType DbType) ([]Column, [][]any) {
	hasGenerated := false
	for _, col := range columns {
		if PreservableGeneratedColumn(col, targetDbType) {
			hasGenerated = true
			break
		}
	}
	if !hasGenerated {
		return columns, values
	}

	keepIdx := make([]int, 0, len(columns))
	keptColumns := make([]Column, 0, len(columns))
	for i, col := range columns {
		if PreservableGeneratedColumn(col, targetDbType) {
			continue
		}
		keepIdx = append(keepIdx, i)
		keptColumns = append(keptColumns, col)
	}
	// 剔除后无列可插（整表均为生成列，理论上不存在）或行值与列数不匹配（无法安全对齐），保持原样交由后续校验快速失败
	if len(keptColumns) == 0 {
		return columns, values
	}
	for _, row := range values {
		if len(row) != len(columns) {
			return columns, values
		}
	}

	keptValues := make([][]any, 0, len(values))
	for _, row := range values {
		r := make([]any, 0, len(keepIdx))
		for _, i := range keepIdx {
			r = append(r, row[i])
		}
		keptValues = append(keptValues, r)
	}
	return keptColumns, keptValues
}

// GenTableDDL 生成通用表DDL
func GenTableDDL(dialect Dialect, md Metadata, tableName string, dropBeforeCreate bool) (string, error) {
	// 1.获取表信息
	tbs, err := md.GetTables(tableName)
	if err != nil {
		logx.Errorf("get table error: %s", tableName)
		return "", err
	}
	if len(tbs) == 0 {
		return "", errorx.NewBizf("table [%s] not found", tableName)
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
	quote := dialect.Quoter().QuoteIdent
	columnStr, valuesStrs := GenInsertSqlColumnAndValues(dialect, dbType, columns, values)

	// insert into table_name (column1, column2, ...) values (value1, value2, ...), (value1, value2, ...), ...
	return fmt.Sprintf("INSERT INTO %s %s VALUES \n%s", quote(tableName), columnStr, strings.Join(valuesStrs, ",\n"))
}

// GenInsertSqlColumnAndValues 生成insert sql对应的 columes信息和values信息
//
//	columnsStr -> (column1, column2, column3, ...)
//	valuesStrs -> ['(value1, value2, value3, ...)', '(value1, value2, value3, ...)', ...]
func GenInsertSqlColumnAndValues(dialect Dialect, dbType DbType, columns []Column, values [][]any) (columnsStr string, valuesStrs []string) {
	// 生成列不可显式插入，需先从列集与行值中同步剔除（仅剔除目标侧确实为生成列的）
	columns, values = OmitGeneratedColumns(columns, values, dbType)

	quote := dialect.Quoter().QuoteIdent

	columnNames := make([]string, 0, len(columns))
	columnTypes := make([]*DbDataType, len(columns))

	strValueArr := make([]string, 0, len(values))

	for i, column := range columns {
		columnNames = append(columnNames, quote(column.ColumnName))
		columnType := GetDbDataType(dbType, column.DataType)
		columnTypes[i] = columnType
	}

	for _, value := range values {
		// 行值数与列数不一致时生成SQL必然列值错位，需快速失败并给出明确上下文，
		// 避免columnTypes越界panic或静默错位
		if len(value) != len(columns) {
			panic(fmt.Sprintf("gen insert sql for db [%s]: row has %d values but %d columns are provided", dbType, len(value), len(columns)))
		}
		vs := make([]string, 0, len(value))
		for i, v := range value {
			// 布尔列必须走SQLValueBool：源方言bool读回为"true"/"false"文本（如pg boolean），
			// 若走数值列的SQLValueNumeric通道会退化为字符串字面量'true'，
			// 写入目标数值/布尔列直接报错（mysql报1366 Incorrect integer value）。
			// SQLValueBool统一输出true/false字面量（mysql tinyint与pg/sqlite的bool/int均接受）
			if columnTypes[i].CommonType == CTBool {
				vs = append(vs, SQLValueBool(v))
				continue
			}
			vs = append(vs, columnTypes[i].DataType.SQLValue(v))
		}
		strValueArr = append(strValueArr, fmt.Sprintf("(%s)", strings.Join(vs, ", ")))
	}

	return fmt.Sprintf("(%s)", strings.Join(columnNames, ", ")), strValueArr
}

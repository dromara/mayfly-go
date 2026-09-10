package clickhouse

import (
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/pgsql"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
	"strings"
	"time"
)

var _ dbi.Dialect = (*ClickHouseDialect)(nil)

type ClickHouseDialect struct {
	dc *dbi.DbConn
}

func (cd *ClickHouseDialect) Quoter() dbi.Quoter {
	return dbi.Quoter{
		Prefix:     '`',
		Suffix:     '`',
		IsReserved: dbi.AlwaysReserve,
	}
}

func (cd *ClickHouseDialect) GetDumpHelper() dbi.DumpHelper {
	return new(dbi.DefaultDumpHelper)
}

func (cd *ClickHouseDialect) GetSQLParser() sqlparser.SqlParser {
	return new(pgsql.PgsqlParser)
}

// GetSQLSplitter clickhouse 切割器：# 与 -- 均为行注释（-- 不要求后随空白），反引号为标识符引用符
func (cd *ClickHouseDialect) GetSQLSplitter() sqlparser.SQLSplitter {
	return sqlparser.NewSplitter(tokenizer.ClickhouseConfig)
}

func (cd *ClickHouseDialect) CopyTable(copy *dbi.DbCopyTable) error {
	quote := cd.Quoter().QuoteIdent
	tableName := copy.TableName

	// 生成新表名，为老表名+_copy_时间戳
	newTableName := tableName + "_copy_" + time.Now().Format("20060102150405")

	// clickhouse不支持create table like，使用create table as复制表结构与引擎（不复制数据）
	if _, err := cd.dc.Exec(fmt.Sprintf("create table %s as %s", quote(newTableName), quote(tableName))); err != nil {
		return err
	}

	// 复制数据（异步执行，执行失败仅记录日志）
	if copy.CopyData {
		gox.Go(func() {
			if _, err := cd.dc.Exec(fmt.Sprintf("insert into %s select * from %s", quote(newTableName), quote(tableName))); err != nil {
				logx.Errorf("clickhouse copy table [%s] data failed: %s", tableName, err.Error())
			}
		})
	}
	return nil
}

func (cd *ClickHouseDialect) GetSQLGenerator() dbi.SQLGenerator {
	return &ClickHouseSQLGenerator{dialect: cd}
}

// ClickHouseSQLGenerator implements the SQLGenerator interface for ClickHouse
var _ dbi.SQLGenerator = (*ClickHouseSQLGenerator)(nil)

type ClickHouseSQLGenerator struct {
	dialect *ClickHouseDialect
}

func (csg *ClickHouseSQLGenerator) GenTableDDL(table dbi.Table, columns []dbi.Column, dropBeforeCreate bool) []string {
	var sqls []string

	quote := csg.dialect.Quoter().QuoteIdent

	if dropBeforeCreate {
		sqls = append(sqls, fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table.TableName)))
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", quote(table.TableName)))

	// 主键列作为MergeTree排序键（MergeTree的ORDER BY列不可为Nullable）
	pkColumns := make([]string, 0)
	for i, col := range columns {
		if i > 0 {
			sb.WriteString(",\n")
		}
		sb.WriteString(fmt.Sprintf("  %s %s", quote(col.ColumnName), csg.columnType(col)))

		if col.ColumnComment != "" {
			sb.WriteString(fmt.Sprintf(" COMMENT '%s'", strings.ReplaceAll(col.ColumnComment, "'", "''")))
		}

		if col.IsPrimaryKey {
			pkColumns = append(pkColumns, quote(col.ColumnName))
		}
	}

	sb.WriteString("\n) ENGINE = MergeTree()")
	// 有主键则按主键排序（MergeTree主键即排序键，同主键数据可依引擎去重），否则退化为tuple()
	if len(pkColumns) > 0 {
		sb.WriteString(fmt.Sprintf(" ORDER BY (%s)", strings.Join(pkColumns, ", ")))
	} else {
		sb.WriteString(" ORDER BY tuple()")
	}

	if table.TableComment != "" {
		sb.WriteString(fmt.Sprintf(" COMMENT '%s'", strings.ReplaceAll(table.TableComment, "'", "''")))
	}

	sqls = append(sqls, sb.String())
	return sqls
}

// columnType 输出clickhouse完整的列类型
//   - Decimal参数化：clickhouse的Decimal必须显式携带(P,S)，精度保留自源列；NumScale为0时GetColumnType仅输出(P)，需补齐S
//   - Nullable包装：源列允许NULL时必须包装Nullable，否则插入NULL会报错（MergeTree排序键列除外）；
//     同库dump时源类型可能自带Nullable前缀，不重复包装
func (csg *ClickHouseSQLGenerator) columnType(col dbi.Column) string {
	colType := col.GetColumnType()

	if strings.HasPrefix(colType, "Decimal(") && !strings.Contains(colType, ",") {
		colType = strings.TrimSuffix(colType, ")") + ", 0)"
	}

	if col.Nullable && !col.IsPrimaryKey && !strings.HasPrefix(colType, "Nullable(") {
		colType = fmt.Sprintf("Nullable(%s)", colType)
	}
	return colType
}

func (csg *ClickHouseSQLGenerator) GenIndexDDL(table dbi.Table, indexs []dbi.Index) []string {
	// MergeTree引擎不支持传统二级索引，源库索引在迁移时统一忽略；
	// 数据去重依赖建表时的主键排序键（见GenTableDDL）
	return []string{}
}

func (csg *ClickHouseSQLGenerator) GenInsert(tableName string, columns []dbi.Column, values [][]any, duplicateStrategy int, targetTableMeta *dbi.TargetTableMeta) []string {
	if len(values) == 0 {
		return []string{}
	}

	quote := csg.dialect.Quoter().QuoteIdent

	// Build column list
	var columnNames []string
	var columnTypes []*dbi.DbDataType

	for _, column := range columns {
		columnNames = append(columnNames, quote(column.ColumnName))
		columnType := dbi.GetDbDataType(DbTypeClickHouse, column.DataType)
		columnTypes = append(columnTypes, columnType)
	}

	// Build values
	var valueRows []string
	for _, row := range values {
		var rowValues []string
		for i, value := range row {
			if i >= len(columnTypes) {
				break
			}
			rowValues = append(rowValues, columnTypes[i].DataType.SQLValue(value))
		}
		valueRows = append(valueRows, fmt.Sprintf("(%s)", strings.Join(rowValues, ", ")))
	}

	insertSql := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		quote(tableName),
		strings.Join(columnNames, ", "),
		strings.Join(valueRows, ", "))

	// ClickHouse 不支持 INSERT IGNORE 与 ON CONFLICT 语法：
	// - None/Ignore 策略或未提供唯一列元信息时，退化为直接插入（去重依赖建表时指定的 MergeTree 引擎）
	// - Update 策略且提供了唯一列时，先删除目标表中与本次插入数据冲突的旧数据，再插入
	if duplicateStrategy != dbi.DuplicateStrategyUpdate || targetTableMeta == nil || len(targetTableMeta.UniqueColumns) == 0 {
		return []string{insertSql}
	}

	// 找出唯一列在插入列中的下标
	uniqueIdx := make(map[int]bool)
	for i, column := range columns {
		if i < len(columnTypes) && collx.ArrayContains(targetTableMeta.UniqueColumns, column.ColumnName) {
			uniqueIdx[i] = true
		}
	}
	if len(uniqueIdx) == 0 {
		return []string{insertSql}
	}

	var keyColumnNames []string
	for i := range columns {
		if uniqueIdx[i] {
			keyColumnNames = append(keyColumnNames, columnNames[i])
		}
	}

	// 构建唯一列的值元组，用于删除目标表中的重复数据
	var keyTuples []string
	for _, row := range values {
		var keyVals []string
		for i, value := range row {
			if i < len(columnTypes) && uniqueIdx[i] {
				keyVals = append(keyVals, columnTypes[i].DataType.SQLValue(value))
			}
		}
		if len(keyVals) == len(keyColumnNames) {
			keyTuples = append(keyTuples, fmt.Sprintf("(%s)", strings.Join(keyVals, ", ")))
		}
	}
	if len(keyTuples) == 0 {
		return []string{insertSql}
	}

	deleteSql := fmt.Sprintf("ALTER TABLE %s DELETE WHERE (%s) IN (%s)",
		quote(tableName),
		strings.Join(keyColumnNames, ", "),
		strings.Join(keyTuples, ", "))
	return []string{deleteSql, insertSql}
}

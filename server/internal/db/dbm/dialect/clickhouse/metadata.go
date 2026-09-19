package clickhouse

import (
	_ "embed"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"strconv"
	"strings"

	"github.com/spf13/cast"
)

//go:embed meta.sql
var metaSqlFile string

// metaSql 方言元数据SQL模板（按备注key解析并缓存，格式见dbi.SqlTemplates）
var metaSql = dbi.NewSqlTemplates(metaSqlFile)

var (
	_ dbi.ServerInfo       = (*ClickHouseMetadata)(nil)
	_ dbi.MetadataProvider = (*ClickHouseMetadata)(nil)
)

type ClickHouseMetadata struct {
	di *dbi.DbInfo
}

func (cm *ClickHouseMetadata) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := cm.di.Query("SELECT version() as version, 'ClickHouse' as database")
	if err != nil {
		return nil, err
	}

	if len(res) > 0 {
		server := &dbi.DbServer{}
		if version, ok := res[0]["version"]; ok {
			if versionStr, ok := version.(string); ok {
				server.Version = versionStr
			}
		}
		dbi.ParseDbVersion(server)
		return server, nil
	}

	return &dbi.DbServer{Version: "unknown"}, nil
}

func (cm *ClickHouseMetadata) GetCompatibleDbVersion() dbi.DbVersion {
	return ""
}

func (cm *ClickHouseMetadata) GetDefaultDb() string {
	return "default"
}

func (cm *ClickHouseMetadata) GetSchemas() ([]string, error) {
	// ClickHouse doesn't have schemas in the traditional sense
	// It has databases that serve a similar purpose
	return cm.GetDbNames()
}

func (cm *ClickHouseMetadata) GetDbNames() ([]string, error) {
	_, res, err := cm.di.Query("SELECT name FROM system.databases WHERE name NOT IN ('system', 'information_schema', 'INFORMATION_SCHEMA') ORDER BY name")
	if err != nil {
		return nil, err
	}

	databases := make([]string, 0)
	for _, row := range res {
		if name, ok := row["name"].(string); ok {
			databases = append(databases, name)
		}
	}

	return databases, nil
}

func (cm *ClickHouseMetadata) GetTables(tableNames ...string) ([]dbi.Table, error) {
	query := "SELECT name, engine, comment FROM system.tables WHERE database = ?"
	args := []any{cm.di.GetDatabase()}
	// 指定表名时需过滤，否则导出指定表时会误导出全库所有表
	if len(tableNames) > 0 {
		placeholders := make([]string, len(tableNames))
		for i, name := range tableNames {
			placeholders[i] = "?"
			args = append(args, name)
		}
		query += fmt.Sprintf(" AND name IN (%s)", strings.Join(placeholders, ","))
	}
	query += " ORDER BY name"

	_, res, err := cm.di.Query(query, args...)
	if err != nil {
		return nil, err
	}

	tables := make([]dbi.Table, 0)
	for _, row := range res {
		tables = append(tables, dbi.Table{
			TableName:    cast.ToString(row["name"]),
			TableComment: cast.ToString(row["comment"]),
		})
	}

	return tables, nil
}

func (cm *ClickHouseMetadata) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	if len(tableNames) == 0 {
		return []dbi.Column{}, nil
	}

	// 逐表查询列信息（多表导出时每张表都需要列信息，只查首表会导致其余表无法生成DDL/插入语句）
	columns := make([]dbi.Column, 0)
	for _, tableName := range tableNames {
		cols, err := cm.getTableColumns(tableName)
		if err != nil {
			return nil, err
		}
		columns = append(columns, cols...)
	}

	return columns, nil
}

func (cm *ClickHouseMetadata) getTableColumns(tableName string) ([]dbi.Column, error) {
	_, res, err := cm.di.Query(`SELECT 
		name,
		type,
		'' as column_comment,
		0 as is_nullable,
		0 as is_primary_key,
		'' as column_default,
		0 as is_auto_increment
	FROM system.columns 
	WHERE database = ? AND table = ? 
	ORDER BY position`, cm.di.GetDatabase(), tableName)
	if err != nil {
		return nil, err
	}

	columns := make([]dbi.Column, 0)
	for _, row := range res {
		column := dbi.Column{
			TableName:     tableName,
			ColumnName:    cast.ToString(row["name"]),
			DataType:      cast.ToString(row["type"]),
			ColumnComment: cast.ToString(row["column_comment"]),
			Nullable:      cast.ToInt(row["is_nullable"]) == 1,
			IsPrimaryKey:  cast.ToInt(row["is_primary_key"]) == 1,
			ColumnDefault: cast.ToString(row["column_default"]),
			AutoIncrement: cast.ToInt(row["is_auto_increment"]) == 1,
		}
		columns = append(columns, column)
	}

	// Fix column data types
	for i := range columns {
		fixColumn(&columns[i])
	}

	return columns, nil
}

func (cm *ClickHouseMetadata) GetPrimaryKey(tableName string) (string, error) {
	// ClickHouse primary keys are defined in the table engine settings
	// This is a simplified implementation
	_, res, err := cm.di.Query(`SELECT primary_key FROM system.tables WHERE database = ? AND name = ?`,
		cm.di.GetDatabase(), tableName)
	if err != nil {
		return "", err
	}

	if len(res) > 0 {
		if pk := cast.ToString(res[0]["primary_key"]); pk != "" {
			// Primary key might be a comma-separated list of columns
			pkParts := strings.Split(pk, ",")
			if len(pkParts) > 0 {
				return strings.TrimSpace(pkParts[0]), nil
			}
		}
	}

	// If no primary key, return the first column
	columns, err := cm.GetColumns(tableName)
	if err != nil || len(columns) == 0 {
		return "", err
	}

	return columns[0].ColumnName, nil
}

func (cm *ClickHouseMetadata) GetTableIndex(tableName string) ([]dbi.Index, error) {
	// ClickHouse doesn't have traditional indexes like other databases
	// It uses primary keys and sorting keys in MergeTree engines
	_, res, err := cm.di.Query(`SELECT 
		name,
		type,
		'' as comment
	FROM system.indexes 
	WHERE database = ? AND table = ?`, cm.di.GetDatabase(), tableName)
	if err != nil {
		// If system.indexes doesn't exist or is not accessible, return empty slice
		return []dbi.Index{}, nil
	}

	indexes := make([]dbi.Index, 0)
	for _, row := range res {
		index := dbi.Index{
			IndexName: cast.ToString(row["name"]),
			IndexType: func() string {
				if t := cast.ToString(row["type"]); t != "" {
					return t
				}
				return "INDEX"
			}(),
			IndexComment: cast.ToString(row["comment"]),
		}
		indexes = append(indexes, index)
	}

	return indexes, nil
}

func (cm *ClickHouseMetadata) GetTableDDL(tableName string, dropBeforeCreate bool) (string, error) {
	// Get the CREATE TABLE statement from system tables
	_, res, err := cm.di.Query(`SELECT create_table_query FROM system.tables WHERE database = ? AND name = ?`,
		cm.di.GetDatabase(), tableName)
	if err != nil {
		return "", err
	}

	if len(res) > 0 {
		if createSql, ok := res[0]["create_table_query"].(string); ok {
			return createSql, nil
		}
	}

	return "", nil
}

// fixColumn 解析 ClickHouse 列元数据：拆封 Nullable/LowCardinality 包装，并解析参数化类型的精度参数。
// 确保 Decimal(p,s)、DateTime64(p) 等类型的精度信息不丢失，保证同方言 DDL 导出/重建的保真度。
func fixColumn(column *dbi.Column) {
	// 1. 拆封 Nullable 包装
	if strings.HasPrefix(column.DataType, "Nullable(") {
		column.Nullable = true
		column.DataType = unwrapWrapper(column.DataType, "Nullable(")
	}

	// 2. 拆封 LowCardinality 包装
	if strings.HasPrefix(column.DataType, "LowCardinality(") {
		column.DataType = unwrapWrapper(column.DataType, "LowCardinality(")
	}

	// 3. 解析参数化类型的精度参数
	parseParameterizedType(column)
}

// unwrapWrapper 拆封 ClickHouse 的类型包装器（Nullable/LowCardinality 等），返回内部类型字符串。
// 正确处理嵌套括号（如 Nullable(Array(String))）。
func unwrapWrapper(s, prefix string) string {
	inner := strings.TrimPrefix(s, prefix)
	// 从外向内匹配平衡的括号
	depth := 1
	for i := 0; i < len(inner); i++ {
		switch inner[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return inner[:i]
			}
		}
	}
	// 未匹配到闭合括号，去除最后一个字符（原始逻辑的兜底）
	return strings.TrimSuffix(inner, ")")
}

// parseParameterizedType 解析 ClickHouse 参数化类型的精度/小数位参数。
// 覆盖 Decimal(P,S)/Decimal32(S)/Decimal64(S)/Decimal128(S)/Decimal256(S) 和 DateTime64(P)。
func parseParameterizedType(column *dbi.Column) {
	t := column.DataType

	// Decimal(P, S) — 提取精度 P 和小数位 S
	if strings.HasPrefix(t, "Decimal(") {
		params := extractParams(t, "Decimal(")
		if len(params) >= 2 {
			column.DataType = "Decimal"
			column.NumPrecision, _ = strconv.Atoi(params[0])
			column.NumScale, _ = strconv.Atoi(params[1])
		}
		return
	}

	// Decimal32(S) / Decimal64(S) / Decimal128(S) / Decimal256(S) — 提取小数位 S
	for _, prefix := range []string{"Decimal32(", "Decimal64(", "Decimal128(", "Decimal256("} {
		if strings.HasPrefix(t, prefix) {
			params := extractParams(t, prefix)
			if len(params) >= 1 {
				baseName := strings.TrimSuffix(prefix, "(")
				column.DataType = baseName
				column.NumScale, _ = strconv.Atoi(params[0])
			}
			return
		}
	}

	// DateTime64(P) — 提取小数秒精度 P
	if strings.HasPrefix(t, "DateTime64(") {
		params := extractParams(t, "DateTime64(")
		if len(params) >= 1 {
			column.DataType = "DateTime64"
			column.NumPrecision, _ = strconv.Atoi(params[0])
			column.NumScale = 0
		}
		return
	}
}

// extractParams 从类型字符串中提取括号内的逗号分隔参数列表。
// 如 extractParams("Decimal(20, 4)", "Decimal(") → ["20", "4"]
func extractParams(s, prefix string) []string {
	inner := strings.TrimPrefix(s, prefix)
	// 找到匹配的闭合括号
	depth := 1
	end := -1
	for i := 0; i < len(inner); i++ {
		switch inner[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				end = i
			}
		}
	}
	if end < 0 {
		return nil
	}
	inner = inner[:end]
	parts := strings.Split(inner, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		result = append(result, strings.TrimSpace(p))
	}
	return result
}

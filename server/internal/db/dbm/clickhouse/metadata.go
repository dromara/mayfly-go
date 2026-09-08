package clickhouse

import (
	_ "embed"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"strings"
)

//go:embed meta.sql
var metaSqlFile string

// metaSql 方言元数据SQL模板（按备注key解析并缓存，格式见dbi.SqlTemplates）
var metaSql = dbi.NewSqlTemplates(metaSqlFile)

var _ dbi.Metadata = (*ClickHouseMetadata)(nil)

type ClickHouseMetadata struct {
	dc *dbi.DbConn
}

func (cm *ClickHouseMetadata) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := cm.dc.Query("SELECT version() as version, 'ClickHouse' as database")
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
	_, res, err := cm.dc.Query("SELECT name FROM system.databases WHERE name NOT IN ('system', 'information_schema', 'INFORMATION_SCHEMA') ORDER BY name")
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
	args := []any{cm.dc.Info.GetDatabase()}
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

	_, res, err := cm.dc.Query(query, args...)
	if err != nil {
		return nil, err
	}

	tables := make([]dbi.Table, 0)
	for _, row := range res {
		name, _ := row["name"].(string)
		comment, _ := row["comment"].(string)
		tables = append(tables, dbi.Table{
			TableName:    name,
			TableComment: comment,
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
	_, res, err := cm.dc.Query(`SELECT 
		name,
		type,
		'' as column_comment,
		0 as is_nullable,
		0 as is_primary_key,
		'' as column_default,
		0 as is_auto_increment
	FROM system.columns 
	WHERE database = ? AND table = ? 
	ORDER BY position`, cm.dc.Info.GetDatabase(), tableName)
	if err != nil {
		return nil, err
	}

	columns := make([]dbi.Column, 0)
	for _, row := range res {
		name, _ := row["name"].(string)
		typ, _ := row["type"].(string)
		column := dbi.Column{
			TableName:  tableName,
			ColumnName: name,
			DataType:   typ,
			ColumnComment: func() string {
				if comment, ok := row["column_comment"].(string); ok {
					return comment
				}
				return ""
			}(),
			Nullable: func() bool {
				if nullable, ok := row["is_nullable"].(int64); ok {
					return nullable == 1
				}
				return false
			}(),
			IsPrimaryKey: func() bool {
				if pk, ok := row["is_primary_key"].(int64); ok {
					return pk == 1
				}
				return false
			}(),
			ColumnDefault: func() string {
				if def, ok := row["column_default"].(string); ok {
					return def
				}
				return ""
			}(),
			AutoIncrement: func() bool {
				if auto, ok := row["is_auto_increment"].(int64); ok {
					return auto == 1
				}
				return false
			}(),
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
	_, res, err := cm.dc.Query(`SELECT primary_key FROM system.tables WHERE database = ? AND name = ?`,
		cm.dc.Info.GetDatabase(), tableName)
	if err != nil {
		return "", err
	}

	if len(res) > 0 {
		if pk, ok := res[0]["primary_key"].(string); ok && pk != "" {
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
	_, res, err := cm.dc.Query(`SELECT 
		name,
		type,
		'' as comment
	FROM system.indexes 
	WHERE database = ? AND table = ?`, cm.dc.Info.GetDatabase(), tableName)
	if err != nil {
		// If system.indexes doesn't exist or is not accessible, return empty slice
		return []dbi.Index{}, nil
	}

	indexes := make([]dbi.Index, 0)
	for _, row := range res {
		name, _ := row["name"].(string)
		index := dbi.Index{
			IndexName: name,
			IndexType: func() string {
				if typ, ok := row["type"].(string); ok {
					return typ
				}
				return "INDEX"
			}(),
			IndexComment: func() string {
				if comment, ok := row["comment"].(string); ok {
					return comment
				}
				return ""
			}(),
		}
		indexes = append(indexes, index)
	}

	return indexes, nil
}

func (cm *ClickHouseMetadata) GetTableDDL(tableName string, dropBeforeCreate bool) (string, error) {
	// Get the CREATE TABLE statement from system tables
	_, res, err := cm.dc.Query(`SELECT create_table_query FROM system.tables WHERE database = ? AND name = ?`,
		cm.dc.Info.GetDatabase(), tableName)
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

// fixColumn fixes column metadata for ClickHouse specific types
func fixColumn(column *dbi.Column) {
	// ClickHouse specific fixes can be added here
	// For example, handling of Nullable types, Array types, etc.

	// Handle Nullable types
	if strings.HasPrefix(column.DataType, "Nullable(") {
		column.Nullable = true
		// Extract the inner type
		innerType := strings.TrimPrefix(column.DataType, "Nullable(")
		innerType = strings.TrimSuffix(innerType, ")")
		column.DataType = innerType
	}

	// Handle Array types
	if strings.HasPrefix(column.DataType, "Array(") {
		// For array types, we might want to store additional metadata
		// This is a simplified approach
	}

	// Handle LowCardinality types
	if strings.HasPrefix(column.DataType, "LowCardinality(") {
		// Extract the inner type
		innerType := strings.TrimPrefix(column.DataType, "LowCardinality(")
		innerType = strings.TrimSuffix(innerType, ")")
		column.DataType = innerType
	}
}

package application

import (
	"fmt"
	"mayfly-go/pkg/biz"
)

// ---------------------------------- pgsql元数据 -----------------------------------
const (
	// postgres 表信息元数据
	PGSQL_TABLE_MA = `SELECT obj_description(c.oid) AS "tableComment", c.relname AS "tableName" FROM pg_class c 
	JOIN pg_namespace n ON c.relnamespace = n.oid WHERE n.nspname = (select current_schema()) AND c.reltype > 0`

	PGSQL_TABLE_INFO = `SELECT obj_description(c.oid) AS "tableComment", c.relname AS "tableName" FROM pg_class c 
	JOIN pg_namespace n ON c.relnamespace = n.oid WHERE n.nspname = (select current_schema()) AND c.reltype > 0`

	PGSQL_INDEX_INFO = `SELECT indexname AS "indexName", indexdef AS "indexComment"
	FROM pg_indexes WHERE schemaname =  (select current_schema()) AND tablename = '%s'`

	PGSQL_COLUMN_MA = `SELECT
		C.relname AS "tableName",
		A.attname AS "columnName",
		tc.is_nullable AS "nullable",
		concat_ws ( '', t.typname, SUBSTRING ( format_type ( a.atttypid, a.atttypmod ) FROM '\(.*\)' ) ) AS "columnType",
		d.description AS "columnComment" 
	FROM
		pg_attribute a LEFT JOIN pg_description d ON d.objoid = a.attrelid 
		AND d.objsubid = A.attnum
		LEFT JOIN pg_class c ON A.attrelid = c.oid
		LEFT JOIN pg_namespace pn ON c.relnamespace = pn.oid
		LEFT JOIN pg_type t ON a.atttypid = t.oid 
		JOIN information_schema.columns tc ON tc.column_name = a.attname AND tc.table_name = C.relname AND tc.table_schema = pn.nspname
	WHERE
		A.attnum >= 0 
		AND pn.nspname = (select current_schema())
		AND C.relname in (%s)
	ORDER BY
		C.relname DESC,
		A.attnum ASC	
	`
)

type PgsqlMetadata struct {
	di *DbInstance
}

// 获取表基础元信息, 如表名等
func (pm *PgsqlMetadata) GetTables() []map[string]interface{} {
	res, err := pm.di.innerSelect(PGSQL_TABLE_MA)
	biz.ErrIsNilAppendErr(err, "获取表基本信息失败: %s")
	return res
}

// 获取列元信息, 如列名等
func (pm *PgsqlMetadata) GetColumns(tableNames ...string) []map[string]interface{} {
	tableName := ""
	for i := 0; i < len(tableNames); i++ {
		if i != 0 {
			tableName = tableName + ", "
		}
		tableName = tableName + "'" + tableNames[i] + "'"
	}
	result, err := pm.di.innerSelect(fmt.Sprintf(PGSQL_COLUMN_MA, tableName))
	biz.ErrIsNilAppendErr(err, "获取数据库列信息失败: %s")
	return result
}

// 获取表主键字段名，默认第一个字段
func (pm *PgsqlMetadata) GetPrimaryKey(tablename string) string {
	columns := pm.GetColumns(tablename)
	if len(columns) == 0 {
		panic(biz.NewBizErr(fmt.Sprintf("[%s] 表不存在", tablename)))
	}
	return columns[0]["columnName"].(string)
}

// 获取表信息，比GetTables获取更详细的表信息
func (pm *PgsqlMetadata) GetTableInfos() []map[string]interface{} {
	res, err := pm.di.innerSelect(PGSQL_TABLE_INFO)
	biz.ErrIsNilAppendErr(err, "获取表信息失败: %s")
	return res
}

// 获取表索引信息
func (pm *PgsqlMetadata) GetTableIndex(tableName string) []map[string]interface{} {
	res, err := pm.di.innerSelect(fmt.Sprintf(PGSQL_INDEX_INFO, tableName))
	biz.ErrIsNilAppendErr(err, "获取表索引信息失败: %s")
	return res
}

// 获取建表ddl
func (mm *PgsqlMetadata) GetCreateTableDdl(tableName string) []map[string]interface{} {
	biz.IsTrue(tableName == "", "暂不支持获取pgsql建表DDL")
	return nil
}

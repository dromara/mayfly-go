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
		concat_ws ( '', t.typname, SUBSTRING ( format_type ( a.atttypid, a.atttypmod ) FROM '\(.*\)' ) ) AS "columnType",
		d.description AS "columnComment" 
	FROM
		pg_attribute a LEFT JOIN pg_description d ON d.objoid = a.attrelid 
		AND d.objsubid = A.attnum
		LEFT JOIN pg_class c ON A.attrelid = c.oid
		LEFT JOIN pg_namespace pn ON c.relnamespace = pn.oid
		LEFT JOIN pg_type t ON a.atttypid = t.oid 
	WHERE
		A.attnum >= 0 
		AND pn.nspname = (select current_schema())
		AND C.relname in (%s)
	ORDER BY
		C.relname DESC,
		A.attnum ASC
	OFFSET %d LIMIT %d	
	`

	PGSQL_COLUMN_MA_COUNT = `SELECT COUNT(*) "maNum"
	FROM
		pg_attribute a LEFT JOIN pg_description d ON d.objoid = a.attrelid 
		AND d.objsubid = A.attnum
		LEFT JOIN pg_class c ON A.attrelid = c.oid
		LEFT JOIN pg_namespace pn ON c.relnamespace = pn.oid
		LEFT JOIN pg_type t ON a.atttypid = t.oid 
	WHERE
		A.attnum >= 0 
		AND pn.nspname = (select current_schema())
		AND C.relname in (%s)
	`
)

type PgsqlMetadata struct {
	di *DbInstance
}

// 获取表基础元信息, 如表名等
func (pm *PgsqlMetadata) GetTables() []map[string]interface{} {
	_, res, _ := pm.di.SelectData(PGSQL_TABLE_MA)
	return res
}

// 获取列元信息, 如列名等
func (pm *PgsqlMetadata) GetColumns(tableNames ...string) []map[string]interface{} {
	var sql, tableName string
	for i := 0; i < len(tableNames); i++ {
		if i != 0 {
			tableName = tableName + ", "
		}
		tableName = tableName + "'" + tableNames[i] + "'"
	}

	pageNum := 1
	// 如果大于一个表，则统计列数并分页获取
	if len(tableNames) > 1 {
		countSql := fmt.Sprintf(PGSQL_COLUMN_MA_COUNT, tableName)
		_, countRes, _ := pm.di.SelectData(countSql)
		maCount := 0
		// 查询出所有列信息总数，手动分页获取所有数据
		if count64, is64 := countRes[0]["maNum"].(int64); is64 {
			maCount = int(count64)
		} else {
			maCount = countRes[0]["maNum"].(int)
		}
		// 计算需要查询的页数
		pageNum = maCount / DEFAULT_COLUMN_SIZE
		if maCount%DEFAULT_COLUMN_SIZE > 0 {
			pageNum++
		}
	}

	res := make([]map[string]interface{}, 0)
	for index := 0; index < pageNum; index++ {
		sql = fmt.Sprintf(PGSQL_COLUMN_MA, tableName, index*DEFAULT_COLUMN_SIZE, DEFAULT_COLUMN_SIZE)
		_, result, err := pm.di.SelectData(sql)
		biz.ErrIsNilAppendErr(err, "获取数据库列信息失败: %s")
		res = append(res, result...)
	}
	return res
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
	_, res, _ := pm.di.SelectData(PGSQL_TABLE_INFO)
	return res
}

// 获取表索引信息
func (pm *PgsqlMetadata) GetTableIndex(tableName string) []map[string]interface{} {
	_, res, _ := pm.di.SelectData(fmt.Sprintf(PGSQL_INDEX_INFO, tableName))
	return res
}

// 获取建表ddl
func (mm *PgsqlMetadata) GetCreateTableDdl(tableName string) []map[string]interface{} {
	return nil
}

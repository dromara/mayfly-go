package application

import (
	"fmt"
	"mayfly-go/pkg/biz"
)

// ---------------------------------- mysql元数据 -----------------------------------
const (
	// mysql 表信息元数据
	MYSQL_TABLE_MA = `SELECT table_name tableName, table_comment tableComment from information_schema.tables WHERE table_schema = (SELECT database())`

	// mysql 表信息
	MYSQL_TABLE_INFO = `SELECT table_name tableName, table_comment tableComment, table_rows tableRows,
	data_length dataLength, index_length indexLength, create_time createTime 
	FROM information_schema.tables 
    WHERE table_schema = (SELECT database())`

	// mysql 索引信息
	MYSQL_INDEX_INFO = `SELECT index_name indexName, group_concat(column_name) columnName, index_type indexType, non_unique nonUnique, 
	SEQ_IN_INDEX seqInIndex, INDEX_COMMENT indexComment
	FROM information_schema.STATISTICS 
    WHERE table_schema = (SELECT database()) AND table_name = '%s' GROUP by index_name`

	// mysql 列信息元数据
	MYSQL_COLUMN_MA = `SELECT table_name tableName, column_name columnName, column_type columnType, column_default columnDefault,
	column_comment columnComment, column_key columnKey, extra, is_nullable nullable from information_schema.columns
	WHERE table_schema = (SELECT database()) AND table_name in (%s) ORDER BY tableName, ordinal_position`
)

type MysqlMetadata struct {
	di *DbInstance
}

// 获取表基础元信息, 如表名等
func (mm *MysqlMetadata) GetTables() []map[string]interface{} {
	res, err := mm.di.innerSelect(MYSQL_TABLE_MA)
	biz.ErrIsNilAppendErr(err, "获取表基本信息失败: %s")
	return res
}

// 获取列元信息, 如列名等
func (mm *MysqlMetadata) GetColumns(tableNames ...string) []map[string]interface{} {
	tableName := ""
	for i := 0; i < len(tableNames); i++ {
		if i != 0 {
			tableName = tableName + ", "
		}
		tableName = tableName + "'" + tableNames[i] + "'"
	}
	result, err := mm.di.innerSelect(fmt.Sprintf(MYSQL_COLUMN_MA, tableName))
	biz.ErrIsNilAppendErr(err, "获取数据库列信息失败: %s")
	return result
}

// 获取表主键字段名，默认第一个字段
func (mm *MysqlMetadata) GetPrimaryKey(tablename string) string {
	columns := mm.GetColumns(tablename)
	if len(columns) == 0 {
		panic(biz.NewBizErr(fmt.Sprintf("[%s] 表不存在", tablename)))
	}
	return columns[0]["columnName"].(string)
}

// 获取表信息，比GetTableMetedatas获取更详细的表信息
func (mm *MysqlMetadata) GetTableInfos() []map[string]interface{} {
	res, err := mm.di.innerSelect(MYSQL_TABLE_INFO)
	biz.ErrIsNilAppendErr(err, "获取表信息失败: %s")
	return res
}

// 获取表索引信息
func (mm *MysqlMetadata) GetTableIndex(tableName string) []map[string]interface{} {
	res, err := mm.di.innerSelect(fmt.Sprintf(MYSQL_INDEX_INFO, tableName))
	biz.ErrIsNilAppendErr(err, "获取表索引信息失败: %s")
	return res
}

// 获取建表ddl
func (mm *MysqlMetadata) GetCreateTableDdl(tableName string) []map[string]interface{} {
	res, _ := mm.di.innerSelect(fmt.Sprintf("show create table %s ", tableName))
	return res
}

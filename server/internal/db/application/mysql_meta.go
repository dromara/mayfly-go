package application

import (
	"fmt"
	"mayfly-go/pkg/biz"
)

// ---------------------------------- mysql元数据 -----------------------------------
const (
	// mysql 表信息元数据
	MYSQL_TABLE_MA = `SELECT table_name tableName, engine, table_comment tableComment, 
	create_time createTime from information_schema.tables
	WHERE table_schema = (SELECT database()) LIMIT 2000`

	// mysql 表信息
	MYSQL_TABLE_INFO = `SELECT table_name tableName, table_comment tableComment, table_rows tableRows,
	data_length dataLength, index_length indexLength, create_time createTime 
	FROM information_schema.tables 
    WHERE table_schema = (SELECT database()) LIMIT 2000`

	// mysql 索引信息
	MYSQL_INDEX_INFO = `SELECT index_name indexName, column_name columnName, index_type indexType,
	SEQ_IN_INDEX seqInIndex, INDEX_COMMENT indexComment
	FROM information_schema.STATISTICS 
    WHERE table_schema = (SELECT database()) AND table_name = '%s' LIMIT 500`

	// mysql 列信息元数据
	MYSQL_COLUMN_MA = `SELECT table_name tableName, column_name columnName, column_type columnType,
	column_comment columnComment, column_key columnKey, extra, is_nullable nullable from information_schema.columns
	WHERE table_name in (%s) AND table_schema = (SELECT database()) ORDER BY tableName, ordinal_position LIMIT %d, %d`

	// mysql 列信息元数据总数
	MYSQL_COLOUMN_MA_COUNT = `SELECT COUNT(*) maNum from information_schema.columns
	WHERE table_name in (%s) AND table_schema = (SELECT database())`
)

type MysqlMetadata struct {
	di *DbInstance
}

// 获取表基础元信息, 如表名等
func (mm *MysqlMetadata) GetTables() []map[string]interface{} {
	_, res, _ := mm.di.SelectData(MYSQL_TABLE_MA)
	return res
}

// 获取列元信息, 如列名等
func (mm *MysqlMetadata) GetColumns(tableNames ...string) []map[string]interface{} {
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
		countSql := fmt.Sprintf(MYSQL_COLOUMN_MA_COUNT, tableName)
		_, countRes, _ := mm.di.SelectData(countSql)
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
		sql = fmt.Sprintf(MYSQL_COLUMN_MA, tableName, index*DEFAULT_COLUMN_SIZE, DEFAULT_COLUMN_SIZE)
		_, result, err := mm.di.SelectData(sql)
		biz.ErrIsNilAppendErr(err, "获取数据库列信息失败: %s")
		res = append(res, result...)
	}
	return res
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
	_, res, _ := mm.di.SelectData(MYSQL_TABLE_INFO)
	return res
}

// 获取表索引信息
func (mm *MysqlMetadata) GetTableIndex(tableName string) []map[string]interface{} {
	_, res, _ := mm.di.SelectData(fmt.Sprintf(MYSQL_INDEX_INFO, tableName))
	return res
}

// 获取建表ddl
func (mm *MysqlMetadata) GetCreateTableDdl(tableName string) []map[string]interface{} {
	_, res, _ := mm.di.SelectData(fmt.Sprintf("show create table %s ", tableName))
	return res
}

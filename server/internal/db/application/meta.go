package application

// -----------------------------------元数据接口定义------------------------------------------
// 数据库元信息接口（表、列等元信息）
// 所有数据查出来直接用map接收，注意不同数据库实现该接口返回的map中的key需要统一.
// 即: 使用别名统一即可。如table_name AS tableName
type DbMetadata interface {

	// 获取表基础元信息
	// 表名: tableName, 备注: tableComment
	GetTables() []map[string]interface{}

	// 获取指定表名的所有列元信息
	// 表名: tableName, 列名: columnName, 列类型: columnType, 备注: columnComment, 是否可为null: nullable, 其他信息: extra
	GetColumns(tableNames ...string) []map[string]interface{}

	// 获取表主键字段名，目前默认第一个字段
	GetPrimaryKey(tablename string) string

	// 获取表信息，比GetTables获取更详细的表信息
	GetTableInfos() []map[string]interface{}

	// 获取表索引信息
	GetTableIndex(tableName string) []map[string]interface{}

	// 获取建表ddl
	GetCreateTableDdl(tableName string) []map[string]interface{}
}

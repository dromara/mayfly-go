package db

import "fmt"

const (
	// mysql 表信息元数据
	MYSQL_TABLE_MA = `SELECT table_name tableName, engine, table_comment tableComment, 
	create_time createTime from information_schema.tables
	WHERE table_schema = (SELECT database())`

	// mysql 列信息元数据
	MYSQL_COLOUMN_MA = `SELECT column_name columnName, column_type columnType,
	column_comment columnComment, column_key columnKey, extra from information_schema.columns
	WHERE table_name = '%s' AND table_schema = (SELECT database()) ORDER BY ordinal_position`
)

func (d *DbInstance) GetTableMetedatas() []map[string]string {
	var sql string
	if d.Type == "mysql" {
		sql = MYSQL_TABLE_MA
	}
	res, _ := d.SelectData(sql)
	return res
}

func (d *DbInstance) GetColumnMetadatas(tableName string) []map[string]string {
	var sql string
	if d.Type == "mysql" {
		sql = fmt.Sprintf(MYSQL_COLOUMN_MA, tableName)
	}
	res, _ := d.SelectData(sql)
	return res
}

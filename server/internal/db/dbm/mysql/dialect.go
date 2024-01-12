package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/utils/anyx"
	"regexp"
	"strings"
)

const (
	MYSQL_META_FILE      = "metasql/mysql_meta.sql"
	MYSQL_DBS            = "MYSQL_DBS"
	MYSQL_TABLE_INFO_KEY = "MYSQL_TABLE_INFO"
	MYSQL_INDEX_INFO_KEY = "MYSQL_INDEX_INFO"
	MYSQL_COLUMN_MA_KEY  = "MYSQL_COLUMN_MA"
)

type MysqlDialect struct {
	dc *dbi.DbConn
}

func (md *MysqlDialect) GetDbServer() (*dbi.DbServer, error) {
	_, res, err := md.dc.Query("SELECT VERSION() version")
	if err != nil {
		return nil, err
	}
	ds := &dbi.DbServer{
		Version: anyx.ConvString(res[0]["version"]),
	}
	return ds, nil
}

func (md *MysqlDialect) GetDbNames() ([]string, error) {
	_, res, err := md.dc.Query(dbi.GetLocalSql(MYSQL_META_FILE, MYSQL_DBS))
	if err != nil {
		return nil, err
	}

	databases := make([]string, 0)
	for _, re := range res {
		databases = append(databases, anyx.ConvString(re["dbname"]))
	}

	return databases, nil
}

// 获取表基础元信息, 如表名等
func (md *MysqlDialect) GetTables() ([]dbi.Table, error) {
	_, res, err := md.dc.Query(dbi.GetLocalSql(MYSQL_META_FILE, MYSQL_TABLE_INFO_KEY))
	if err != nil {
		return nil, err
	}

	tables := make([]dbi.Table, 0)
	for _, re := range res {
		tables = append(tables, dbi.Table{
			TableName:    re["tableName"].(string),
			TableComment: anyx.ConvString(re["tableComment"]),
			CreateTime:   anyx.ConvString(re["createTime"]),
			TableRows:    anyx.ConvInt(re["tableRows"]),
			DataLength:   anyx.ConvInt64(re["dataLength"]),
			IndexLength:  anyx.ConvInt64(re["indexLength"]),
		})
	}
	return tables, nil
}

// 获取列元信息, 如列名等
func (md *MysqlDialect) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	tableName := ""
	for i := 0; i < len(tableNames); i++ {
		if i != 0 {
			tableName = tableName + ", "
		}
		tableName = tableName + "'" + tableNames[i] + "'"
	}

	_, res, err := md.dc.Query(fmt.Sprintf(dbi.GetLocalSql(MYSQL_META_FILE, MYSQL_COLUMN_MA_KEY), tableName))
	if err != nil {
		return nil, err
	}

	columns := make([]dbi.Column, 0)
	for _, re := range res {
		columns = append(columns, dbi.Column{
			TableName:     re["tableName"].(string),
			ColumnName:    re["columnName"].(string),
			ColumnType:    anyx.ConvString(re["columnType"]),
			ColumnComment: anyx.ConvString(re["columnComment"]),
			Nullable:      anyx.ConvString(re["nullable"]),
			ColumnKey:     anyx.ConvString(re["columnKey"]),
			ColumnDefault: anyx.ConvString(re["columnDefault"]),
			NumScale:      anyx.ConvString(re["numScale"]),
		})
	}
	return columns, nil
}

// 获取表主键字段名，不存在主键标识则默认第一个字段
func (md *MysqlDialect) GetPrimaryKey(tablename string) (string, error) {
	columns, err := md.GetColumns(tablename)
	if err != nil {
		return "", err
	}
	if len(columns) == 0 {
		return "", errorx.NewBiz("[%s] 表不存在", tablename)
	}

	for _, v := range columns {
		if v.ColumnKey == "PRI" {
			return v.ColumnName, nil
		}
	}

	return columns[0].ColumnName, nil
}

// 获取表索引信息
func (md *MysqlDialect) GetTableIndex(tableName string) ([]dbi.Index, error) {
	_, res, err := md.dc.Query(dbi.GetLocalSql(MYSQL_META_FILE, MYSQL_INDEX_INFO_KEY), tableName)
	if err != nil {
		return nil, err
	}

	indexs := make([]dbi.Index, 0)
	for _, re := range res {
		indexs = append(indexs, dbi.Index{
			IndexName:    re["indexName"].(string),
			ColumnName:   anyx.ConvString(re["columnName"]),
			IndexType:    anyx.ConvString(re["indexType"]),
			IndexComment: anyx.ConvString(re["indexComment"]),
			NonUnique:    anyx.ConvInt(re["nonUnique"]),
			SeqInIndex:   anyx.ConvInt(re["seqInIndex"]),
		})
	}
	// 把查询结果以索引名分组，索引字段以逗号连接
	result := make([]dbi.Index, 0)
	key := ""
	for _, v := range indexs {
		// 当前的索引名
		in := v.IndexName
		if key == in {
			// 索引字段已根据名称和顺序排序，故取最后一个即可
			i := len(result) - 1
			// 同索引字段以逗号连接
			result[i].ColumnName = result[i].ColumnName + "," + v.ColumnName
		} else {
			key = in
			result = append(result, v)
		}
	}
	return result, nil
}

// 获取建表ddl
func (md *MysqlDialect) GetTableDDL(tableName string) (string, error) {
	_, res, err := md.dc.Query(fmt.Sprintf("show create table `%s` ", tableName))
	if err != nil {
		return "", err
	}
	return res[0]["Create Table"].(string) + ";", nil
}

func (md *MysqlDialect) WalkTableRecord(tableName string, walkFn dbi.WalkQueryRowsFunc) error {
	return md.dc.WalkQueryRows(context.Background(), fmt.Sprintf("SELECT * FROM %s", tableName), walkFn)
}

func (md *MysqlDialect) GetSchemas() ([]string, error) {
	return nil, nil
}

// GetDbProgram 获取数据库程序模块，用于数据库备份与恢复
func (md *MysqlDialect) GetDbProgram() dbi.DbProgram {
	return NewDbProgramMysql(md.dc)
}

func (md *MysqlDialect) GetDataType(dbColumnType string) dbi.DataType {
	if regexp.MustCompile(`(?i)int|double|float|number|decimal|byte|bit`).MatchString(dbColumnType) {
		return dbi.DataTypeNumber
	}
	// 日期时间类型
	if regexp.MustCompile(`(?i)datetime|timestamp`).MatchString(dbColumnType) {
		return dbi.DataTypeDateTime
	}
	// 日期类型
	if regexp.MustCompile(`(?i)date`).MatchString(dbColumnType) {
		return dbi.DataTypeDate
	}
	// 时间类型
	if regexp.MustCompile(`(?i)time`).MatchString(dbColumnType) {
		return dbi.DataTypeTime
	}
	return dbi.DataTypeString
}

func (md *MysqlDialect) BatchInsert(tx *sql.Tx, tableName string, columns []string, values [][]any) (int64, error) {
	// 生成占位符字符串：如：(?,?)
	// 重复字符串并用逗号连接
	repeated := strings.Repeat("?,", len(columns))
	// 去除最后一个逗号，占位符由括号包裹
	placeholder := fmt.Sprintf("(%s)", strings.TrimSuffix(repeated, ","))

	// 执行批量insert sql，mysql支持批量insert语法
	// insert into table_name (column1, column2, ...) values (value1, value2, ...), (value1, value2, ...), ...

	// 重复占位符字符串n遍
	repeated = strings.Repeat(placeholder+",", len(values))
	// 去除最后一个逗号
	placeholder = strings.TrimSuffix(repeated, ",")

	sqlStr := fmt.Sprintf("insert into %s (%s) values %s", md.dc.Info.Type.QuoteIdentifier(tableName), strings.Join(columns, ","), placeholder)
	// 执行批量insert sql
	// 把二维数组转为一维数组
	var args []any
	for _, v := range values {
		args = append(args, v...)
	}
	return md.dc.TxExec(tx, sqlStr, args...)
}

func (md *MysqlDialect) FormatStrData(dbColumnValue string, dataType dbi.DataType) string {
	// mysql不需要格式化时间日期等
	return dbColumnValue
}

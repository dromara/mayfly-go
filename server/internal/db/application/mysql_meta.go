package application

import (
	"context"
	"database/sql"
	"fmt"
	"mayfly-go/internal/db/domain/entity"
	machineapp "mayfly-go/internal/machine/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/utils/anyx"
	"net"

	"github.com/go-sql-driver/mysql"
)

func getMysqlDB(d *entity.Db, db string) (*sql.DB, error) {
	// SSH Conect
	if d.SshTunnelMachineId > 0 {
		sshTunnelMachine := machineapp.GetMachineApp().GetSshTunnelMachine(d.SshTunnelMachineId)
		mysql.RegisterDialContext(d.Network, func(ctx context.Context, addr string) (net.Conn, error) {
			return sshTunnelMachine.GetDialConn("tcp", addr)
		})
	}
	// 设置dataSourceName  -> 更多参数参考：https://github.com/go-sql-driver/mysql#dsn-data-source-name
	dsn := fmt.Sprintf("%s:%s@%s(%s:%d)/%s?timeout=8s", d.Username, d.Password, d.Network, d.Host, d.Port, db)
	if d.Params != "" {
		dsn = fmt.Sprintf("%s&%s", dsn, d.Params)
	}
	return sql.Open(d.Type, dsn)
}

// ---------------------------------- mysql元数据 -----------------------------------
const (
	MYSQL_META_FILE      = "metasql/mysql_meta.sql"
	MYSQL_TABLE_MA_KEY   = "MYSQL_TABLE_MA"
	MYSQL_TABLE_INFO_KEY = "MYSQL_TABLE_INFO"
	MYSQL_INDEX_INFO_KEY = "MYSQL_INDEX_INFO"
	MYSQL_COLUMN_MA_KEY  = "MYSQL_COLUMN_MA"
)

type MysqlMetadata struct {
	di *DbInstance
}

// 获取表基础元信息, 如表名等
func (mm *MysqlMetadata) GetTables() []Table {
	_, res, err := mm.di.SelectData(GetLocalSql(MYSQL_META_FILE, MYSQL_TABLE_MA_KEY))
	biz.ErrIsNilAppendErr(err, "获取表基本信息失败: %s")

	tables := make([]Table, 0)
	for _, re := range res {
		tables = append(tables, Table{
			TableName:    re["tableName"].(string),
			TableComment: anyx.ConvString(re["tableComment"]),
		})
	}
	return tables
}

// 获取列元信息, 如列名等
func (mm *MysqlMetadata) GetColumns(tableNames ...string) []Column {
	tableName := ""
	for i := 0; i < len(tableNames); i++ {
		if i != 0 {
			tableName = tableName + ", "
		}
		tableName = tableName + "'" + tableNames[i] + "'"
	}

	_, res, err := mm.di.SelectData(fmt.Sprintf(GetLocalSql(MYSQL_META_FILE, MYSQL_COLUMN_MA_KEY), tableName))
	biz.ErrIsNilAppendErr(err, "获取数据库列信息失败: %s")
	columns := make([]Column, 0)
	for _, re := range res {
		columns = append(columns, Column{
			TableName:     re["tableName"].(string),
			ColumnName:    re["columnName"].(string),
			ColumnType:    anyx.ConvString(re["columnType"]),
			ColumnComment: anyx.ConvString(re["columnComment"]),
			Nullable:      anyx.ConvString(re["nullable"]),
			ColumnKey:     anyx.ConvString(re["columnKey"]),
			ColumnDefault: anyx.ConvString(re["columnDefault"]),
		})
	}
	return columns
}

// 获取表主键字段名，不存在主键标识则默认第一个字段
func (mm *MysqlMetadata) GetPrimaryKey(tablename string) string {
	columns := mm.GetColumns(tablename)
	biz.IsTrue(len(columns) > 0, "[%s] 表不存在", tablename)
	for _, v := range columns {
		if v.ColumnKey == "PRI" {
			return v.ColumnName
		}
	}

	return columns[0].ColumnName
}

// 获取表信息，比GetTableMetedatas获取更详细的表信息
func (mm *MysqlMetadata) GetTableInfos() []Table {
	_, res, err := mm.di.SelectData(GetLocalSql(MYSQL_META_FILE, MYSQL_TABLE_INFO_KEY))
	biz.ErrIsNilAppendErr(err, "获取表信息失败: %s")

	tables := make([]Table, 0)
	for _, re := range res {
		tables = append(tables, Table{
			TableName:    re["tableName"].(string),
			TableComment: anyx.ConvString(re["tableComment"]),
			CreateTime:   anyx.ConvString(re["createTime"]),
			TableRows:    anyx.ConvInt(re["tableRows"]),
			DataLength:   anyx.ConvInt64(re["dataLength"]),
			IndexLength:  anyx.ConvInt64(re["indexLength"]),
		})
	}
	return tables
}

// 获取表索引信息
func (mm *MysqlMetadata) GetTableIndex(tableName string) []Index {
	_, res, err := mm.di.SelectData(fmt.Sprintf(GetLocalSql(MYSQL_META_FILE, MYSQL_INDEX_INFO_KEY), tableName))
	biz.ErrIsNilAppendErr(err, "获取表索引信息失败: %s")
	indexs := make([]Index, 0)
	for _, re := range res {
		indexs = append(indexs, Index{
			IndexName:    re["indexName"].(string),
			ColumnName:   anyx.ConvString(re["columnName"]),
			IndexType:    anyx.ConvString(re["indexType"]),
			IndexComment: anyx.ConvString(re["indexComment"]),
			NonUnique:    anyx.ConvInt(re["nonUnique"]),
			SeqInIndex:   anyx.ConvInt(re["seqInIndex"]),
		})
	}
	// 把查询结果以索引名分组，索引字段以逗号连接
	result := make([]Index, 0)
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
	return result
}

// 获取建表ddl
func (mm *MysqlMetadata) GetCreateTableDdl(tableName string) string {
	_, res, _ := mm.di.SelectData(fmt.Sprintf("show create table %s ", tableName))
	return res[0]["Create Table"].(string)
}

func (mm *MysqlMetadata) GetTableRecord(tableName string, pageNum, pageSize int) ([]string, []map[string]any, error) {
	return mm.di.SelectData(fmt.Sprintf("SELECT * FROM %s LIMIT %d, %d", tableName, (pageNum-1)*pageSize, pageSize))
}

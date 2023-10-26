package application

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"mayfly-go/internal/db/domain/entity"
	machineapp "mayfly-go/internal/machine/application"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"net"
	"strings"
	"time"

	"github.com/lib/pq"
)

func getPgsqlDB(d *entity.Instance, db string) (*sql.DB, error) {
	driverName := string(d.Type)
	// SSH Conect
	if d.SshTunnelMachineId > 0 {
		// 如果使用了隧道，则使用`postgres:ssh:隧道机器id`注册名
		driverName = fmt.Sprintf("postgres:ssh:%d", d.SshTunnelMachineId)
		if !collx.ArrayContains(sql.Drivers(), driverName) {
			sql.Register(driverName, &PqSqlDialer{sshTunnelMachineId: d.SshTunnelMachineId})
		}
		sql.Drivers()
	}

	var dbParam string
	if db != "" {
		dbParam = "dbname=" + db
	}
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s %s sslmode=disable", d.Host, d.Port, d.Username, d.Password, dbParam)
	if d.Params != "" {
		dsn = fmt.Sprintf("%s %s", dsn, strings.Join(strings.Split(d.Params, "&"), " "))
	}
	return sql.Open(driverName, dsn)
}

// pgsql dialer
type PqSqlDialer struct {
	sshTunnelMachineId int
}

func (d *PqSqlDialer) Open(name string) (driver.Conn, error) {
	return pq.DialOpen(d, name)
}

func (pd *PqSqlDialer) Dial(network, address string) (net.Conn, error) {
	sshTunnel, err := machineapp.GetMachineApp().GetSshTunnelMachine(pd.sshTunnelMachineId)
	if err != nil {
		return nil, err
	}
	if sshConn, err := sshTunnel.GetDialConn("tcp", address); err == nil {
		return sshConn, nil
	} else {
		return nil, err
	}
}

func (pd *PqSqlDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	return pd.Dial(network, address)
}

// ---------------------------------- pgsql元数据 -----------------------------------
const (
	PGSQL_META_FILE      = "metasql/pgsql_meta.sql"
	PGSQL_TABLE_MA_KEY   = "PGSQL_TABLE_MA"
	PGSQL_TABLE_INFO_KEY = "PGSQL_TABLE_INFO"
	PGSQL_INDEX_INFO_KEY = "PGSQL_INDEX_INFO"
	PGSQL_COLUMN_MA_KEY  = "PGSQL_COLUMN_MA"
	PGSQL_TABLE_DDL_KEY  = "PGSQL_TABLE_DDL_FUNC"
)

type PgsqlMetadata struct {
	di *DbConnection
}

// 获取表基础元信息, 如表名等
func (pm *PgsqlMetadata) GetTables() ([]Table, error) {
	_, res, err := pm.di.SelectData(GetLocalSql(PGSQL_META_FILE, PGSQL_TABLE_MA_KEY))
	if err != nil {
		return nil, err
	}

	tables := make([]Table, 0)
	for _, re := range res {
		tables = append(tables, Table{
			TableName:    re["tableName"].(string),
			TableComment: anyx.ConvString(re["tableComment"]),
		})
	}
	return tables, nil
}

// 获取列元信息, 如列名等
func (pm *PgsqlMetadata) GetColumns(tableNames ...string) ([]Column, error) {
	tableName := ""
	for i := 0; i < len(tableNames); i++ {
		if i != 0 {
			tableName = tableName + ", "
		}
		tableName = tableName + "'" + tableNames[i] + "'"
	}

	_, res, err := pm.di.SelectData(fmt.Sprintf(GetLocalSql(PGSQL_META_FILE, PGSQL_COLUMN_MA_KEY), tableName))
	if err != nil {
		return nil, err
	}

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
	return columns, nil
}

func (pm *PgsqlMetadata) GetPrimaryKey(tablename string) (string, error) {
	columns, err := pm.GetColumns(tablename)
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

// 获取表信息，比GetTables获取更详细的表信息
func (pm *PgsqlMetadata) GetTableInfos() ([]Table, error) {
	_, res, err := pm.di.SelectData(GetLocalSql(PGSQL_META_FILE, PGSQL_TABLE_INFO_KEY))
	if err != nil {
		return nil, err
	}

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
	return tables, nil
}

// 获取表索引信息
func (pm *PgsqlMetadata) GetTableIndex(tableName string) ([]Index, error) {
	_, res, err := pm.di.SelectData(fmt.Sprintf(GetLocalSql(PGSQL_META_FILE, PGSQL_INDEX_INFO_KEY), tableName))
	if err != nil {
		return nil, err
	}

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
	return indexs, nil
}

// 获取建表ddl
func (pm *PgsqlMetadata) GetCreateTableDdl(tableName string) (string, error) {
	_, err := pm.di.Exec(GetLocalSql(PGSQL_META_FILE, PGSQL_TABLE_DDL_KEY))
	if err != nil {
		return "", err
	}

	_, schemaRes, _ := pm.di.SelectData("select current_schema() as schema")
	schemaName := schemaRes[0]["schema"].(string)

	ddlSql := fmt.Sprintf("select showcreatetable('%s','%s') as sql", schemaName, tableName)
	_, res, err := pm.di.SelectData(ddlSql)
	if err != nil {
		return "", err
	}

	return res[0]["sql"].(string), nil
}

func (pm *PgsqlMetadata) GetTableRecord(tableName string, pageNum, pageSize int) ([]string, []map[string]any, error) {
	return pm.di.SelectData(fmt.Sprintf("SELECT * FROM %s OFFSET %d LIMIT %d", tableName, (pageNum-1)*pageSize, pageSize))
}

func (pm *PgsqlMetadata) WalkTableRecord(tableName string, walk func(record map[string]any, columns []string)) error {
	return pm.di.WalkTableRecord(fmt.Sprintf("SELECT * FROM %s", tableName), walk)
}

package dbm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	machineapp "mayfly-go/internal/machine/application"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/netx"
	"net"
	"regexp"
	"strings"
	"time"

	pq "gitee.com/liuzongyang/libpq"
)

func getPgsqlDB(d *DbInfo) (*sql.DB, error) {
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

	db := d.Database
	var dbParam string
	exsitSchema := false
	if db != "" {
		// postgres database可以使用db/schema表示，方便连接指定schema, 若不存在schema则使用默认schema
		ss := strings.Split(db, "/")
		if len(ss) > 1 {
			exsitSchema = true
			dbParam = fmt.Sprintf("dbname=%s search_path=%s", ss[0], ss[len(ss)-1])
		} else {
			dbParam = "dbname=" + db
		}
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s %s sslmode=disable connect_timeout=8", d.Host, d.Port, d.Username, d.Password, dbParam)
	// 存在额外指定参数，则拼接该连接参数
	if d.Params != "" {
		// 存在指定的db，则需要将dbInstance配置中的parmas排除掉dbname和search_path
		if db != "" {
			paramArr := strings.Split(d.Params, "&")
			paramArr = collx.ArrayRemoveFunc(paramArr, func(param string) bool {
				if strings.HasPrefix(param, "dbname=") {
					return true
				}
				if exsitSchema && strings.HasPrefix(param, "search_path") {
					return true
				}
				return false
			})
			d.Params = strings.Join(paramArr, " ")
		}
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
		// 将ssh conn包装，否则会返回错误: ssh: tcpChan: deadline not supported
		return &netx.WrapSshConn{Conn: sshConn}, nil
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
	PGSQL_DB_SCHEMAS     = "PGSQL_DB_SCHEMAS"
	PGSQL_TABLE_INFO_KEY = "PGSQL_TABLE_INFO"
	PGSQL_INDEX_INFO_KEY = "PGSQL_INDEX_INFO"
	PGSQL_COLUMN_MA_KEY  = "PGSQL_COLUMN_MA"
	PGSQL_TABLE_DDL_KEY  = "PGSQL_TABLE_DDL_FUNC"
)

type PgsqlDialect struct {
	dc *DbConn
}

func (pd *PgsqlDialect) GetDbServer() (*DbServer, error) {
	_, res, err := pd.dc.Query("SHOW server_version")
	if err != nil {
		return nil, err
	}
	ds := &DbServer{
		Version: anyx.ConvString(res[0]["server_version"]),
	}
	return ds, nil
}

func (pd *PgsqlDialect) GetDbNames() ([]string, error) {
	_, res, err := pd.dc.Query("SELECT datname AS dbname FROM pg_database WHERE datistemplate = false AND has_database_privilege(datname, 'CONNECT')")
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
func (pd *PgsqlDialect) GetTables() ([]Table, error) {
	_, res, err := pd.dc.Query(GetLocalSql(PGSQL_META_FILE, PGSQL_TABLE_INFO_KEY))
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

// 获取列元信息, 如列名等
func (pd *PgsqlDialect) GetColumns(tableNames ...string) ([]Column, error) {
	tableName := ""
	for i := 0; i < len(tableNames); i++ {
		if i != 0 {
			tableName = tableName + ", "
		}
		tableName = tableName + "'" + tableNames[i] + "'"
	}

	_, res, err := pd.dc.Query(fmt.Sprintf(GetLocalSql(PGSQL_META_FILE, PGSQL_COLUMN_MA_KEY), tableName))
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
			NumScale:      anyx.ConvString(re["numScale"]),
		})
	}
	return columns, nil
}

func (pd *PgsqlDialect) GetPrimaryKey(tablename string) (string, error) {
	columns, err := pd.GetColumns(tablename)
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
func (pd *PgsqlDialect) GetTableIndex(tableName string) ([]Index, error) {
	_, res, err := pd.dc.Query(fmt.Sprintf(GetLocalSql(PGSQL_META_FILE, PGSQL_INDEX_INFO_KEY), tableName))
	if err != nil {
		return nil, err
	}

	indexs := make([]Index, 0)
	for _, re := range res {
		indexs = append(indexs, Index{
			IndexName:    re["indexName"].(string),
			ColumnName:   anyx.ConvString(re["columnName"]),
			IndexType:    anyx.ConvString(re["IndexType"]),
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
	return result, nil
}

// 获取建表ddl
func (pd *PgsqlDialect) GetTableDDL(tableName string) (string, error) {
	_, err := pd.dc.Exec(GetLocalSql(PGSQL_META_FILE, PGSQL_TABLE_DDL_KEY))
	if err != nil {
		return "", err
	}

	_, schemaRes, _ := pd.dc.Query("select current_schema() as schema")
	schemaName := schemaRes[0]["schema"].(string)

	ddlSql := fmt.Sprintf("select showcreatetable('%s','%s') as sql", schemaName, tableName)
	_, res, err := pd.dc.Query(ddlSql)
	if err != nil {
		return "", err
	}

	return res[0]["sql"].(string), nil
}

func (pd *PgsqlDialect) WalkTableRecord(tableName string, walkFn WalkQueryRowsFunc) error {
	return pd.dc.WalkQueryRows(context.Background(), fmt.Sprintf("SELECT * FROM %s", tableName), walkFn)
}

// 获取pgsql当前连接的库可访问的schemaNames
func (pd *PgsqlDialect) GetSchemas() ([]string, error) {
	sql := GetLocalSql(PGSQL_META_FILE, PGSQL_DB_SCHEMAS)
	_, res, err := pd.dc.Query(sql)
	if err != nil {
		return nil, err
	}
	schemaNames := make([]string, 0)
	for _, re := range res {
		schemaNames = append(schemaNames, anyx.ConvString(re["schemaName"]))
	}
	return schemaNames, nil
}

// GetDbProgram 获取数据库程序模块，用于数据库备份与恢复
func (pd *PgsqlDialect) GetDbProgram() DbProgram {
	panic("implement me")
}

func (pd *PgsqlDialect) WrapName(name string) string {
	return fmt.Sprintf(`"%s"`, name)
}

func (pd *PgsqlDialect) GetDataType(dbColumnType string) DataType {
	if regexp.MustCompile(`(?i)int|double|float|number|decimal|byte|bit`).MatchString(dbColumnType) {
		return DataTypeNumber
	}
	// 日期时间类型
	if regexp.MustCompile(`(?i)datetime|timestamp`).MatchString(dbColumnType) {
		return DataTypeDateTime
	}
	// 日期类型
	if regexp.MustCompile(`(?i)date`).MatchString(dbColumnType) {
		return DataTypeDate
	}
	// 时间类型
	if regexp.MustCompile(`(?i)time`).MatchString(dbColumnType) {
		return DataTypeTime
	}
	return DataTypeString
}

func (pd *PgsqlDialect) BatchInsert(tx *sql.Tx, tableName string, columns []string, values [][]any) (int64, error) {
	// 执行批量insert sql，跟mysql一样  pg或高斯支持批量insert语法
	// insert into table_name (column1, column2, ...) values (value1, value2, ...), (value1, value2, ...), ...

	// 把二维数组转为一维数组
	var args []any
	for _, v := range values {
		args = append(args, v...)
	}

	// 构建占位符字符串 "($1, $2, $3), ($4, $5, $6), ..." 用于指定参数
	var placeholders []string
	for i := 0; i < len(args); i += len(columns) {
		var placeholder []string
		for j := 0; j < len(columns); j++ {
			placeholder = append(placeholder, fmt.Sprintf("$%d", i+j+1))
		}
		placeholders = append(placeholders, "("+strings.Join(placeholder, ", ")+")")
	}

	sqlStr := fmt.Sprintf("insert into %s (%s) values %s", pd.WrapName(tableName), strings.Join(columns, ","), strings.Join(placeholders, ", "))
	// 执行批量insert sql

	return pd.dc.TxExec(tx, sqlStr, args...)
}

func (pd *PgsqlDialect) FormatStrData(dbColumnValue string, dataType DataType) string {
	switch dataType {
	case DataTypeDateTime: // "2024-01-02T22:16:28.545377+08:00"
		res, _ := time.Parse(time.RFC3339, dbColumnValue)
		return res.Format(time.DateTime)
	case DataTypeDate: //  "2024-01-02T00:00:00Z"
		res, _ := time.Parse(time.RFC3339, dbColumnValue)
		return res.Format(time.DateOnly)
	case DataTypeTime: // "0000-01-01T22:16:28.545075+08:00"
		res, _ := time.Parse(time.RFC3339, dbColumnValue)
		return res.Format(time.TimeOnly)
	}
	return dbColumnValue
}

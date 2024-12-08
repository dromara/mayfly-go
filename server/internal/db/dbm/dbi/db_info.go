package dbi

import (
	"fmt"
	machineapp "mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"strings"
)

type DbType string

const (
	DbTypeMysql      DbType = "mysql"
	DbTypeMariadb    DbType = "mariadb"
	DbTypePostgres   DbType = "postgres"
	DbTypeGauss      DbType = "gauss"
	DbTypeDM         DbType = "dm"
	DbTypeOracle     DbType = "oracle"
	DbTypeSqlite     DbType = "sqlite"
	DbTypeMssql      DbType = "mssql"
	DbTypeKingbaseEs DbType = "kingbaseEs"
	DbTypeVastbase   DbType = "vastbase"
)

func ToDbType(dbType string) DbType {
	return DbType(dbType)
}

func (dbType DbType) Equal(typ string) bool {
	return ToDbType(typ) == dbType
}

type DbInfo struct {
	InstanceId uint64 // 实例id
	Id         uint64 // dbId
	Name       string

	Type     DbType // 类型，mysql postgres等
	Host     string
	Port     int
	Extra    string // 连接需要的其他额外参数（json字符串），如oracle数据库需要指定sid等
	Network  string
	Username string
	Password string
	Params   string
	Database string // 若有schema的库则为'database/scheam'格式

	Version        DbVersion // 数据库版本信息，用于语法兼容
	DefaultVersion bool      // 经过查询数据库版本信息后，是否仍然使用默认版本

	CodePath           []string
	SshTunnelMachineId int

	Meta Meta
}

// 获取记录日志的描述
func (d *DbInfo) GetLogDesc() string {
	return fmt.Sprintf("DB[id=%d, tag=%s, name=%s, ip=%s:%d, database=%s]", d.Id, d.CodePath, d.Name, d.Host, d.Port, d.Database)
}

// 连接数据库
func (dbInfo *DbInfo) Conn(meta Meta) (*DbConn, error) {
	if meta == nil {
		return nil, errorx.NewBiz("the database meta information interface cannot be empty")
	}

	// 赋值Meta，方便后续获取dialect等
	dbInfo.Meta = meta
	database := dbInfo.Database
	// 如果数据库为空，则使用默认数据库进行连接
	if database == "" {
		database = meta.GetMetadata(&DbConn{Info: dbInfo}).GetDefaultDb()
		dbInfo.Database = database
	}

	conn, err := meta.GetSqlDb(dbInfo)
	if err != nil {
		logx.Errorf("db connection failed: %s:%d/%s, err:%s", dbInfo.Host, dbInfo.Port, database, err.Error())
		return nil, errorx.NewBiz(fmt.Sprintf("db connection failed: %s", err.Error()))
	}

	err = conn.Ping()
	if err != nil {
		logx.Errorf("db ping failed: %s:%d/%s, err:%s", dbInfo.Host, dbInfo.Port, database, err.Error())
		return nil, errorx.NewBiz(fmt.Sprintf("db connection failed: %s", err.Error()))
	}

	dbc := &DbConn{Id: GetDbConnId(dbInfo.Id, database), Info: dbInfo}

	// 最大连接周期，超过时间的连接就close
	// conn.SetConnMaxLifetime(100 * time.Second)
	// 设置最大连接数
	conn.SetMaxOpenConns(5)
	// 设置闲置连接数
	conn.SetMaxIdleConns(1)
	dbc.db = conn
	logx.Infof("db connection: %s:%d/%s", dbInfo.Host, dbInfo.Port, database)

	return dbc, nil
}

// 如果使用了ssh隧道，将其host port改变其本地映射host port
func (di *DbInfo) IfUseSshTunnelChangeIpPort() error {
	// 开启ssh隧道
	if di.SshTunnelMachineId > 0 {
		sshTunnelMachine, err := GetSshTunnel(di.SshTunnelMachineId)
		if err != nil {
			return err
		}
		exposedIp, exposedPort, err := sshTunnelMachine.OpenSshTunnel(fmt.Sprintf("db:%d", di.Id), di.Host, di.Port)
		if err != nil {
			return err
		}
		di.Host = exposedIp
		di.Port = exposedPort
	}
	return nil
}

// 获取当前库的schema（兼容 database/schema模式）
func (di *DbInfo) CurrentSchema() string {
	dbName := di.Database
	schema := ""
	arr := strings.Split(dbName, "/")
	if len(arr) == 2 {
		schema = arr[1]
	}
	return schema
}

// 获取当前数据库（兼容 database/schema模式）
func (di *DbInfo) GetDatabase() string {
	dbName := di.Database
	ss := strings.Split(dbName, "/")
	if len(ss) > 1 {
		return ss[0]
	}
	return dbName
}

// 根据ssh tunnel机器id返回ssh tunnel
func GetSshTunnel(sshTunnelMachineId int) (*mcm.SshTunnelMachine, error) {
	return machineapp.GetMachineApp().GetSshTunnelMachine(sshTunnelMachineId)
}

// 获取连接id
func GetDbConnId(dbId uint64, db string) string {
	if dbId == 0 {
		return ""
	}

	return fmt.Sprintf("%d:%s", dbId, db)
}

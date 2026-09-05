package dbi

import (
	"context"
	"fmt"
	machineapp "mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/mcm"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"strings"
	"time"
)

type DbType string

func ToDbType(dbType string) DbType {
	return DbType(dbType)
}

func (dbType DbType) Equal(typ string) bool {
	return ToDbType(typ) == dbType
}

type DbInfo struct {
	model.ExtraData // 连接需要的其他额外参数（json字符串），如oracle数据库需要指定sid等

	InstanceId uint64 // 实例id
	Id         uint64 // dbId
	DbCode     string
	Name       string

	Type     DbType // 类型，mysql postgres等
	Host     string
	Port     int
	Network  string
	Username string
	Password string
	Params   string
	Database string // 若有schema的库则为'database/scheam'格式

	Version        DbVersion // 数据库版本信息，用于语法兼容
	DefaultVersion bool      // 经过查询数据库版本信息后，是否仍然使用默认版本

	CodePath           []string
	SshTunnelMachineId int
	RemoteAddr         string `json:"-"` // ssh隧道远程地址，格式 ip:port

	Meta Meta
}

var _ (mcm.SshTunnelAble) = (*DbInfo)(nil)

func (di *DbInfo) String() string {
	return fmt.Sprintf("DbInfo{Id: %d, Name: %s, Type: %s, Host: %s, Port: %d, Database: %s}", di.Id, di.Name, di.Type, di.Host, di.Port, di.Database)
}

func (di *DbInfo) GetSshTunnelMachineId() int64 {
	return int64(di.SshTunnelMachineId)
}

func (di *DbInfo) GetRemoteAddr() string {
	if di.RemoteAddr != "" {
		return di.RemoteAddr
	}
	return fmt.Sprintf("%s:%d", di.Host, di.Port)
}

// 获取记录日志的描述
func (di *DbInfo) GetLogDesc() string {
	return fmt.Sprintf("DB[id=%d, tag=%s, name=%s, ip=%s:%d, database=%s]", di.Id, di.CodePath, di.Name, di.Host, di.Port, di.Database)
}

// 连接池参数（也可按需调整为实例配置）
const (
	connMaxLifetime = 5 * time.Hour  // 连接最大存活时间
	connMaxIdleTime = 3 * time.Hour  // 闲置连接最大存活时间
	maxOpenConns    = 10             // 最大打开连接数
	maxIdleConns    = 1              // 最大闲置连接数
)

// 连接数据库
func (di *DbInfo) Conn(ctx context.Context, meta Meta) (*DbConn, error) {
	if meta == nil {
		return nil, errorx.NewBiz("the database meta information interface cannot be empty")
	}

	// 赋值Meta，方便后续获取dialect等
	di.Meta = meta
	database := di.Database
	// 如果数据库为空，则使用默认数据库进行连接
	if database == "" {
		database = meta.GetMetadata(&DbConn{Info: di}).GetDefaultDb()
		di.Database = database
	}

	if err := di.IfUseSshTunnelChangeIpPort(ctx); err != nil {
		return nil, err
	}
	conn, err := meta.GetSqlDb(ctx, di)
	if err != nil {
		logx.Errorf("db connection failed: %s:%d/%s, err:%s", di.Host, di.Port, database, err.Error())
		return nil, errorx.NewBizf("db connection failed: %s", err.Error())
	}

	err = conn.Ping()
	if err != nil {
		logx.Errorf("db ping failed: %s:%d/%s, err:%s", di.Host, di.Port, database, err.Error())
		return nil, errorx.NewBizf("db connection failed: %s", err.Error())
	}

	dbc := &DbConn{Id: GetDbConnId(di.Id, database), Info: di}

	conn.SetConnMaxLifetime(connMaxLifetime)
	conn.SetConnMaxIdleTime(connMaxIdleTime)
	// 设置最大连接数
	conn.SetMaxOpenConns(maxOpenConns)
	// 设置闲置连接
	conn.SetMaxIdleConns(maxIdleConns)

	dbc.db = conn
	logx.Infof("db connection: %s:%d/%s", di.Host, di.Port, database)

	return dbc, nil
}

// 如果使用了ssh隧道，将其host port改变其本地映射host port
func (di *DbInfo) IfUseSshTunnelChangeIpPort(ctx context.Context) error {
	// 开启ssh隧道
	if di.SshTunnelMachineId > 0 {
		// 防止同一DbInfo重复建立隧道：仅在首次记录原始远程地址，
		// 否则二次调用时 GetRemoteAddr 会将已映射的本地地址当作原始地址记录，导致隧道重建后连错目标
		if di.RemoteAddr == "" {
			di.RemoteAddr = fmt.Sprintf("%s:%d", di.Host, di.Port)
		}
		sshTunnelMachine, err := GetSshTunnel(ctx, di.SshTunnelMachineId)
		if err != nil {
			return err
		}
		exposedIp, exposedPort, err := sshTunnelMachine.OpenSshTunnel(di)
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
func GetSshTunnel(ctx context.Context, sshTunnelMachineId int) (*mcm.SshTunnelMachine, error) {
	return machineapp.GetMachineApp().GetSshTunnelMachine(ctx, sshTunnelMachineId)
}

// 获取连接id
func GetDbConnId(dbId uint64, db string) string {
	if dbId == 0 {
		return ""
	}

	return fmt.Sprintf("db-%d:%s", dbId, db)
}

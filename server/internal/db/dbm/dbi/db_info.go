package dbi

import (
	"context"
	"database/sql"
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

// DbConnPoolConfig 连接池配置：支持实例级自定义，未配置时使用默认值。
type DbConnPoolConfig struct {
	ConnMaxLifetime time.Duration // 连接最大存活时间
	ConnMaxIdleTime time.Duration // 闲置连接最大存活时间
	MaxOpenConns    int           // 最大打开连接数
	MaxIdleConns    int           // 最大闲置连接数
}

// DefaultDbConnPoolConfig 默认连接池配置（全局回退值）
var DefaultDbConnPoolConfig = DbConnPoolConfig{
	ConnMaxLifetime: 5 * time.Hour,
	ConnMaxIdleTime: 3 * time.Hour,
	MaxOpenConns:    10,
	MaxIdleConns:    1,
}

// Resolve 返回生效的池参数：实例配置非零则覆盖默认值
func (c DbConnPoolConfig) Resolve() (maxLifetime, maxIdleTime time.Duration, maxOpen, maxIdle int) {
	maxLifetime = DefaultDbConnPoolConfig.ConnMaxLifetime
	maxIdleTime = DefaultDbConnPoolConfig.ConnMaxIdleTime
	maxOpen = DefaultDbConnPoolConfig.MaxOpenConns
	maxIdle = DefaultDbConnPoolConfig.MaxIdleConns
	if c.ConnMaxLifetime > 0 {
		maxLifetime = c.ConnMaxLifetime
	}
	if c.ConnMaxIdleTime > 0 {
		maxIdleTime = c.ConnMaxIdleTime
	}
	if c.MaxOpenConns > 0 {
		maxOpen = c.MaxOpenConns
	}
	if c.MaxIdleConns > 0 {
		maxIdle = c.MaxIdleConns
	}
	return
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

	Backend DbBackend

	// PoolConfig 实例级连接池配置（可选，零值使用全局默认配置）
	PoolConfig DbConnPoolConfig

	// db 底层连接池，由 Conn() 建立后赋值，方言通过 GetDb() 访问
	db *sql.DB
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

// GetLogDesc 获取记录日志的描述
func (di *DbInfo) GetLogDesc() string {
	return fmt.Sprintf("DB[id=%d, tag=%s, name=%s, ip=%s:%d, database=%s]", di.Id, di.CodePath, di.Name, di.Host, di.Port, di.Database)
}

// GetDb 获取底层连接池。方言的 Metadata/Dialect 实现通过此方法执行 SQL，
// 不再依赖 *DbConn，解耦工厂方法与连接容器。
func (di *DbInfo) GetDb() *sql.DB {
	return di.db
}

// QueryContext 执行查询语句，返回列信息、结果集和错误。
// 支持 context 控制超时/取消。
func (di *DbInfo) QueryContext(ctx context.Context, querySql string, args ...any) ([]*QueryColumn, []map[string]any, error) {
	rows, err := di.db.QueryContext(ctx, querySql, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, nil, err
	}
	cols := make([]*QueryColumn, len(colTypes))
	scans := make([]any, len(colTypes))
	for k, colType := range colTypes {
		colName := colType.Name()
		if colName == "" {
			colName = fmt.Sprintf("<anonymous%d>", k+1)
		}
		qc := NewQueryColumn(colName, GetDbDataType(di.Type, colType.DatabaseTypeName()))
		cols[k] = qc
		scans[k] = qc.getValuePtr()
	}

	result := make([]map[string]any, 0, 16)
	for rows.Next() {
		if err := rows.Scan(scans...); err != nil {
			return cols, nil, err
		}
		rowData := make(map[string]any, len(cols))
		for _, col := range cols {
			rowData[col.Name] = col.value()
		}
		result = append(result, rowData)
	}
	return cols, result, rows.Err()
}

// Query 便捷方法：执行查询语句，返回列信息、结果集和错误。
// 使用 context.Background()，如需控制超时请使用 QueryContext。
func (di *DbInfo) Query(querySql string, args ...any) ([]*QueryColumn, []map[string]any, error) {
	return di.QueryContext(context.Background(), querySql, args...)
}

// ExecContext 执行 SQL 语句，返回影响行数和错误。
// 支持 context 控制超时/取消。
func (di *DbInfo) ExecContext(ctx context.Context, sql string, args ...any) (int64, error) {
	res, err := di.db.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Exec 便捷方法：执行 SQL 语句，返回影响行数和错误。
// 使用 context.Background()，如需控制超时请使用 ExecContext。
func (di *DbInfo) Exec(sql string, args ...any) (int64, error) {
	return di.ExecContext(context.Background(), sql, args...)
}

// GetDialect 便捷方法：获取方言实例。
func (di *DbInfo) GetDialect() Dialect {
	return di.Backend.GetDialect(di)
}

// GetDbDataType 便捷方法：按类型名查找方言的数据类型。
func (di *DbInfo) GetDbDataType(dataType string) *DbDataType {
	return GetDbDataType(di.Type, dataType)
}

// Metadata 便捷方法：创建 Schema 元数据访问入口。
func (di *DbInfo) Metadata() *Metadata {
	return NewMetadata(di.Backend, di.Backend.GetMetadataProvider(di), di.Backend.GetServerInfo(di))
}

// 连接数据库
func (di *DbInfo) Conn(ctx context.Context, backend DbBackend) (*DbConn, error) {
	if backend == nil {
		return nil, errorx.NewBiz("the database backend interface cannot be empty")
	}

	// 赋值Backend，方便后续获取dialect等
	di.Backend = backend
	database := di.Database
	// 如果数据库为空，则使用默认数据库进行连接
	if database == "" {
		database = backend.GetServerInfo(di).GetDefaultDb()
		di.Database = database
	}

	if err := di.IfUseSshTunnelChangeIpPort(ctx); err != nil {
		return nil, err
	}
	conn, err := backend.GetSqlDb(ctx, di)
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

	// 使用实例级连接池配置（有则覆盖，无则使用默认值）
	maxLifetime, maxIdleTime, maxOpen, maxIdle := di.PoolConfig.Resolve()
	conn.SetConnMaxLifetime(maxLifetime)
	conn.SetConnMaxIdleTime(maxIdleTime)
	conn.SetMaxOpenConns(maxOpen)
	conn.SetMaxIdleConns(maxIdle)

	// 将底层连接存入 DbInfo，方言通过 di.GetDb() 访问
	di.db = conn

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

// CurrentSchema 获取当前库的schema（兼容 database/schema模式）
func (di *DbInfo) CurrentSchema() string {
	dbName := di.Database
	schema := ""
	arr := strings.Split(dbName, "/")
	if len(arr) == 2 {
		schema = arr[1]
	}
	return schema
}

// GetDatabase 获取当前数据库（兼容 database/schema模式）
func (di *DbInfo) GetDatabase() string {
	dbName := di.Database
	ss := strings.Split(dbName, "/")
	if len(ss) > 1 {
		return ss[0]
	}
	return dbName
}

// GetSshTunnel 根据ssh tunnel机器id返回ssh tunnel
func GetSshTunnel(ctx context.Context, sshTunnelMachineId int) (*mcm.SshTunnelMachine, error) {
	return machineapp.GetMachineApp().GetSshTunnelMachine(ctx, sshTunnelMachineId)
}

// GetDbConnId 获取连接id
func GetDbConnId(dbId uint64, db string) string {
	if dbId == 0 {
		return ""
	}

	return fmt.Sprintf("db-%d:%s", dbId, db)
}

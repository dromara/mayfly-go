package dbi

import (
	"context"
	"database/sql"
	"fmt"
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

	CodePath []string
	// SshTunnelMachineId 需要经其中转机建立通道时使用的机器标识（>0 表示启用）
	SshTunnelMachineId int
	// RemoteAddr 建立通道前的目标原始地址，格式 ip:port；仅首次改写前记录，避免通道重建时误用已映射地址
	RemoteAddr string `json:"-"`

	Backend DbBackend

	// PoolConfig 实例级连接池配置（可选，零值使用全局默认配置）
	PoolConfig DbConnPoolConfig

	// db 底层连接池，由 Conn() 建立后赋值，方言通过 GetDb() 访问
	db *sql.DB

	// schemaCache 服务端元数据缓存（跨请求存活，按逻辑库连接单例），由 Conn() 建立连接时初始化；
	// 未建立连接的 DbInfo（如纯 SQL 生成）为 nil。用指针（非内嵌互斥量）以保持 DbInfo 可安全按值拷贝。
	schemaCache *schemaCache
}

func (di *DbInfo) String() string {
	return fmt.Sprintf("DbInfo{Id: %d, Name: %s, Type: %s, Host: %s, Port: %d, Database: %s}", di.Id, di.Name, di.Type, di.Host, di.Port, di.Database)
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
//
// 与 DbConn.walkQueryRows 共用 buildQueryColumns/scanRowMap，保证重复列名消歧、
// QueryColumn.Key 回写与 rows.Err() 检查在各查询路径上行为一致（元数据查询常含
// SELECT a.id, b.id 之类同名列，若不去重会静默丢列）。
func (di *DbInfo) QueryContext(ctx context.Context, querySQL string, args ...any) ([]*QueryColumn, []map[string]any, error) {
	rows, err := di.db.QueryContext(ctx, querySQL, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, nil, err
	}
	cols, scans := buildQueryColumns(colTypes, di.Type)

	result := make([]map[string]any, 0, 16)
	for rows.Next() {
		if err := rows.Scan(scans...); err != nil {
			return cols, nil, err
		}
		result = append(result, scanRowMap(cols))
	}
	// rows.Next() 返回 false 也可能是游标中途 IO/网络错误，必须显式检查，避免静默返回截断结果
	if err := rows.Err(); err != nil {
		return cols, nil, wrapSQLError(err)
	}
	return cols, result, nil
}

// Query 便捷方法：执行查询语句，返回列信息、结果集和错误。
// 使用 context.Background()，如需控制超时请使用 QueryContext。
func (di *DbInfo) Query(querySQL string, args ...any) ([]*QueryColumn, []map[string]any, error) {
	return di.QueryContext(context.Background(), querySQL, args...)
}

// ExecContext 执行 SQL 语句，返回影响行数和错误。
// 支持 context 控制超时/取消。
func (di *DbInfo) ExecContext(ctx context.Context, sql string, args ...any) (int64, error) {
	res, err := di.db.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, err
	}
	// 执行成功的 DDL 改变了库结构：立即失效服务端元数据缓存（判据见 invalidateIfDDL）
	di.invalidateIfDDL(sql)
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

// SchemaCache 返回该连接的元数据缓存；仅在连接建立（Conn）后非 nil。
func (di *DbInfo) SchemaCache() *schemaCache {
	return di.schemaCache
}

// InvalidateSchemaCache 清空该连接的元数据缓存：使补全与资源树立即反映最新结构。
// 它是缓存失效的唯一写入口（DDL 执行后的自动失效见 invalidateIfDDL，也可直接调用本方法）。
// TTL 为兜底安全网，此处主动失效只为消除改表后的陈旧窗口；无缓存时为 no-op。
func (di *DbInfo) InvalidateSchemaCache() {
	if di.schemaCache != nil {
		di.schemaCache.invalidate()
	}
}

// Metadata 便捷方法：创建 Schema 元数据访问入口，并接入该连接的跨请求元数据缓存。
func (di *DbInfo) Metadata() *MetadataReader {
	return NewMetadataReader(di.Backend, di.Backend.GetMetadataProvider(di), di.Backend.GetServerInfo(di), di.SchemaCache())
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

	// 若配置了中转通道则先建立，拿到实际可拨地址与通道句柄
	tunnel, err := di.dialTunnel(ctx)
	if err != nil {
		return nil, err
	}

	conn, err := backend.GetSQLDb(ctx, di)
	if err != nil {
		tunnel.Close() // 建连失败，释放已建立的通道，避免其引用计数泄漏
		logx.Errorf("db connection failed: %s:%d/%s, err:%s", di.Host, di.Port, database, err.Error())
		return nil, errorx.NewBizf("db connection failed: %s", err.Error())
	}

	err = conn.Ping()
	if err != nil {
		tunnel.Close() // 探活失败同样要释放已建立的通道
		logx.Errorf("db ping failed: %s:%d/%s, err:%s", di.Host, di.Port, database, err.Error())
		return nil, errorx.NewBizf("db connection failed: %s", err.Error())
	}

	// 通道句柄随连接持有，连接关闭时由其显式释放
	dbc := &DbConn{Id: GetDbConnId(di.Id, database), Info: di, tunnel: tunnel}

	// 使用实例级连接池配置（有则覆盖，无则使用默认值）
	maxLifetime, maxIdleTime, maxOpen, maxIdle := di.PoolConfig.Resolve()
	conn.SetConnMaxLifetime(maxLifetime)
	conn.SetConnMaxIdleTime(maxIdleTime)
	conn.SetMaxOpenConns(maxOpen)
	conn.SetMaxIdleConns(maxIdle)

	// 将底层连接存入 DbInfo，方言通过 di.GetDb() 访问
	di.db = conn

	// 建立连接即初始化该逻辑库的元数据缓存；DbInfo 随连接池单例存活，后续读取 happens-after 本次写入
	di.schemaCache = newSchemaCache()

	logx.Infof("db connection: %s:%d/%s", di.Host, di.Port, database)

	return dbc, nil
}

// dialTunnel 若连接配置了中转通道，则经注册的 TunnelOpener 建立通道，
// 将 DbInfo 的目标地址改写为通道暴露的本地地址，并返回持有释放钩子的通道句柄。
// 无通道需求时返回 (nil, nil)（其 Close 对 nil 接收者安全）。
func (di *DbInfo) dialTunnel(ctx context.Context) (*Tunnel, error) {
	if di.SshTunnelMachineId <= 0 {
		return nil, nil
	}
	if tunnelOpener == nil {
		return nil, errorx.NewBiz("tunnel is required but no tunnel opener is registered")
	}
	// 仅首次记录目标原始地址：通道建立后 Host/Port 被改为本地映射地址，
	// 若重复进入此处会把本地地址误当原始地址，导致通道重建后连错目标
	if di.RemoteAddr == "" {
		di.RemoteAddr = fmt.Sprintf("%s:%d", di.Host, di.Port)
	}
	tunnel, err := tunnelOpener.Open(ctx, TunnelSpec{MachineId: di.SshTunnelMachineId, RemoteAddr: di.RemoteAddr})
	if err != nil {
		return nil, err
	}
	di.Host = tunnel.Host
	di.Port = tunnel.Port
	return tunnel, nil
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

// GetDbConnId 获取连接id
func GetDbConnId(dbId uint64, db string) string {
	if dbId == 0 {
		return ""
	}

	return fmt.Sprintf("db-%d:%s", dbId, db)
}

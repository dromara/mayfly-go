package dbi

import (
	"context"
	"database/sql"
	"sync"
)

var (
	backendsMu sync.RWMutex // 保护 backends 的并发访问
	backends   = make(map[DbType]DbBackend)

	// instanceIndex 实例ID到连接ID的反向索引，避免 GetDbConnByInstanceId 线性遍历所有连接池
	instanceIndexMu sync.RWMutex
	instanceIndex   = make(map[uint64][]string) // instanceId → []connId
)

type DbVersion string

// DbBackend 数据库方言后端（工厂）
// 每个 DbType 注册一个 DbBackend，负责创建连接、提供方言、元数据等能力
//
// 工厂方法统一接收 *DbInfo（连接信息），不再依赖 *DbConn（连接实例），
// 解耦工厂与产品容器，支持在无真实连接的场景下（如纯 SQL 生成、单元测试）独立使用方言能力。
type DbBackend interface {
	// GetSqlDb 根据数据库信息建立底层 *sql.DB 连接
	GetSqlDb(context.Context, *DbInfo) (*sql.DB, error)

	// GetDialect 获取数据库方言。接收 *DbInfo 而非 *DbConn，
	// 方言内部需要执行 SQL 时通过 di.GetDb() 获取底层连接
	GetDialect(*DbInfo) Dialect

	// GetServerInfo 获取数据库服务器信息
	GetServerInfo(*DbInfo) ServerInfo

	// GetMetadataProvider 获取方言的 Schema 内省能力
	GetMetadataProvider(*DbInfo) MetadataProvider

	// GetCapabilities 返回方言的元数据能力声明
	GetCapabilities() MetadataCapabilities

	// CommitTargetTx 提交目标库写事务。
	// 默认实现直接 tx.Commit()；方言后端可覆写以处理驱动怪癖（如 mssql）
	CommitTargetTx(conn *DbConn, tx *sql.Tx) error
}

// RegisterBackend 注册数据库类型与后端
// 注意：仅支持在程序启动阶段（各方言包 init 中）调用，不支持运行期并发注册
func RegisterBackend(dt DbType, backend DbBackend) {
	if backend == nil {
		panic("dbi: register nil backend for db type: " + dt)
	}
	backendsMu.Lock()
	defer backendsMu.Unlock()
	backends[dt] = backend
}

// GetBackend 根据数据库类型获取对应的 DbBackend
func GetBackend(dt DbType) DbBackend {
	backendsMu.RLock()
	defer backendsMu.RUnlock()
	return backends[dt]
}

// GetRegisteredDbTypes 返回当前已注册的所有数据库类型快照（含各方言的别名键，如mariadb/gauss）。
// 供完备性测试等场景枚举校验，返回副本，调用方修改不影响注册表
func GetRegisteredDbTypes() []DbType {
	backendsMu.RLock()
	defer backendsMu.RUnlock()

	dts := make([]DbType, 0, len(backends))
	for dt := range backends {
		dts = append(dts, dt)
	}
	return dts
}

// GetDialect 获取数据库方言，无需建立真实数据库连接。
// 构建最小化的 DbInfo 传入工厂方法，适用于纯 SQL 生成场景。
func GetDialect(dt DbType) Dialect {
	backend := GetBackend(dt)
	if backend == nil {
		return nil
	}
	di := &DbInfo{
		Type:    dt,
		Backend: backend,
	}
	return backend.GetDialect(di)
}

// ========== 实例索引管理（供 dbm 层调用）==========

// RegisterInstanceConn 注册实例ID到连接ID的映射（连接建立时调用）
func RegisterInstanceConn(instanceId uint64, connId string) {
	if instanceId == 0 || connId == "" {
		return
	}
	instanceIndexMu.Lock()
	defer instanceIndexMu.Unlock()
	instanceIndex[instanceId] = append(instanceIndex[instanceId], connId)
}

// UnregisterInstanceConn 移除实例ID到连接ID的映射（连接关闭时调用）
func UnregisterInstanceConn(instanceId uint64, connId string) {
	if instanceId == 0 || connId == "" {
		return
	}
	instanceIndexMu.Lock()
	defer instanceIndexMu.Unlock()
	connIds := instanceIndex[instanceId]
	for i, id := range connIds {
		if id == connId {
			instanceIndex[instanceId] = append(connIds[:i], connIds[i+1:]...)
			break
		}
	}
	if len(instanceIndex[instanceId]) == 0 {
		delete(instanceIndex, instanceId)
	}
}

// GetConnIdsByInstance 获取实例ID关联的所有连接ID
func GetConnIdsByInstance(instanceId uint64) []string {
	instanceIndexMu.RLock()
	defer instanceIndexMu.RUnlock()
	cp := make([]string, len(instanceIndex[instanceId]))
	copy(cp, instanceIndex[instanceId])
	return cp
}

// ========== BaseBackend：DbBackend 默认基类 ==========

// BaseBackend DbBackend 默认基类，提供 GetCapabilities 与 CommitTargetTx 的通用默认实现。
// 各方言 Backend 嵌入此结构体后，仅需覆写差异方法（如特殊能力声明或事务提交怪癖）。
type BaseBackend struct{}

// GetCapabilities 默认能力：全部支持（适用于 mssql/oracle/dm 等全功能方言）。
// 能力不足的方言（mysql/sqlite/clickhouse 等）必须覆写此方法。
func (b *BaseBackend) GetCapabilities() MetadataCapabilities {
	return MetadataCapabilities{
		SupportsSchemas:           true,
		SupportsIndexes:           true,
		SupportsForeignKeys:       true,
		SupportsComments:          true,
		SupportsDDLExport:         true,
		SupportsGeneratedColumns:  true,
		SupportsIdentityColumns:   true,
		SupportsExpressionDefault: true,
		NamespaceHierarchy: NamespaceHierarchy{
			HasDatabase: true,
			HasSchema:   true,
		},
	}
}

// CommitTargetTx 默认提交：直接 tx.Commit()。
// 方言后端若存在提交怪癖（如 mssql），覆写此方法处理。
func (b *BaseBackend) CommitTargetTx(conn *DbConn, tx *sql.Tx) error {
	if tx == nil {
		return nil
	}
	return tx.Commit()
}

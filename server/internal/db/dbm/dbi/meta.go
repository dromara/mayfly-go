package dbi

import (
	"context"
	"database/sql"
	"sync"
)

var (
	metasMu sync.RWMutex        // 保护 metas 与 metaInit 的并发访问
	metas   = make(map[DbType]Meta)
	// metaInited 记录对应数据库类型的列类型与类型转换器是否已完成注册，
	// 注册由 GetMeta 首次调用时完成且仅执行一次（写入受 metasMu 保护）
	metaInited = make(map[DbType]bool)
)

type DbVersion string

// Meta 数据库元信息，如获取sql.DB、Dialect等
type Meta interface {
	// GetSqlDb 根据数据库信息获取sql.DB
	GetSqlDb(context.Context, *DbInfo) (*sql.DB, error)

	// GetDialect 获取数据库方言, 若一些接口不需要DbConn，则可以传nil
	GetDialect(*DbConn) Dialect

	// GetMetadata 获取元数据信息接口
	//  -  *DbConn 数据库连接
	GetMetadata(*DbConn) Metadata

	// GetDbDataTypes 获取所有数据库对应的数据类型
	GetDbDataTypes() []*DbDataType

	// GetCommonTypeConverter 获取公共类型转换器，用于迁移与同步
	GetCommonTypeConverter() CommonTypeConverter
}

// Register 注册数据库类型与dbmeta
// 注意：仅支持在程序启动阶段（各方言包 init 中）调用，不支持运行期并发注册
func Register(dt DbType, meta Meta) {
	if meta == nil {
		panic("dbi: register nil meta for db type: " + dt)
	}
	metasMu.Lock()
	defer metasMu.Unlock()
	metas[dt] = meta
	metaInited[dt] = false
}

// GetMeta 根据数据库类型获取对应的Meta，首次获取时会完成该类型的列类型与转换器注册（仅一次）
func GetMeta(dt DbType) Meta {
	metasMu.Lock()
	defer metasMu.Unlock()

	meta, ok := metas[dt]
	if !ok {
		return nil
	}
	// 未初始化，则进行初始化，如注册数据库类型等。防止未使用到的数据库都被注册
	if !metaInited[dt] {
		initMeta(dt, meta)
		metaInited[dt] = true
	}
	return meta
}

// GetDialect 获取数据库方言，如果dialect方法内需要用到dbConn的，则不支持该方法
func GetDialect(dt DbType) Dialect {
	// 创建一个假连接，仅用于调用方言生成sql，不做数据库连接操作
	meta := GetMeta(dt)
	if meta == nil {
		return nil
	}
	dbConn := &DbConn{Info: &DbInfo{
		Type: dt,
		Meta: meta,
	}}
	return meta.GetDialect(dbConn)
}

// initMeta 初始化数据库类型，如注册数据库类型等（调用方需持有写锁）
func initMeta(dt DbType, meta Meta) {
	registerColumnDbDataTypes(dt, meta.GetDbDataTypes()...)
	registerCommonTypeConverter(dt, meta.GetCommonTypeConverter())
}

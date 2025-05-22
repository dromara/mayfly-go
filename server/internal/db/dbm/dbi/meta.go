package dbi

import (
	"context"
	"database/sql"
)

var (
	metas    = make(map[DbType]Meta)
	metaInit = make(map[DbType]bool)
)

type DbVersion string

// Meta 数据库元信息，如获取sql.DB、Dialect等
type Meta interface {
	// GetSqlDb 根据数据库信息获取sql.DB
	GetSqlDb(context.Context, *DbInfo) (*sql.DB, error)

	// GetDialect 获取数据库方言, 若一些接口不需要DbConn，则可以传nil
	GetDialect(*DbConn) Dialect

	// GetMetadata 获取元数据信息接口
	// @param *DbConn 数据库连接
	GetMetadata(*DbConn) Metadata

	// GetDbDataTypes 获取所有数据库对应的数据类型
	GetDbDataTypes() []*DbDataType

	// GetCommonTypeConverter 获取公共类型转换器，用于迁移与同步
	GetCommonTypeConverter() CommonTypeConverter
}

// 注册数据库类型与dbmeta
func Register(dt DbType, meta Meta) {
	metas[dt] = meta
	metaInit[dt] = false
}

// 根据数据库类型获取对应的Meta
func GetMeta(dt DbType) Meta {
	// 未初始化，则进行初始化，如注册数据库类型等。防止未使用到的数据库都被注册
	if inited := metaInit[dt]; !inited {
		initMeta(dt, metas[dt])
	}
	return metas[dt]
}

// GetDialect 获取数据库方言，如果dialect方法内需要用到dbConn的，则不支持该方法
func GetDialect(dt DbType) Dialect {
	// 创建一个假连接，仅用于调用方言生成sql，不做数据库连接操作
	meta := GetMeta(dt)
	dbConn := &DbConn{Info: &DbInfo{
		Type: dt,
		Meta: meta,
	}}
	return meta.GetDialect(dbConn)
}

// initMeta 初始化数据库类型，如注册数据库类型等
func initMeta(dt DbType, meta Meta) {
	registerColumnDbDataTypes(dt, meta.GetDbDataTypes()...)
	registerCommonTypeConverter(dt, meta.GetCommonTypeConverter())
}

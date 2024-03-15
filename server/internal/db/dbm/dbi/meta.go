package dbi

import "database/sql"

var (
	metas = make(map[DbType]Meta)
)

// 注册数据库类型与dbmeta
func Register(dt DbType, meta Meta) {
	metas[dt] = meta
}

// 根据数据库类型获取对应的Meta
func GetMeta(dt DbType) Meta {
	return metas[dt]
}

// 数据库元信息，如获取sql.DB、Dialect等
type Meta interface {
	// 根据数据库信息获取sql.DB
	GetSqlDb(*DbInfo) (*sql.DB, error)

	// 获取数据库方言
	GetDialect(*DbConn) Dialect

	// 获取元数据信息接口
	// @param *DbConn 数据库连接， 若一些元数据接口（如 GetIdentifierQuoteString）不需要DbConn，则可以传nil
	GetMetaData(*DbConn) *MetaDataX
}

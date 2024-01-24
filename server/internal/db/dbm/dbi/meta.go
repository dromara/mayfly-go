package dbi

import "database/sql"

var (
	metas map[DbType]Meta = make(map[DbType]Meta)
)

// 注册数据库类型与dbmeta
func Register(dt DbType, meta Meta) {
	metas[dt] = meta
}

// 根据数据库类型获取对应的Meta
func GetMeta(dt DbType) Meta {
	return metas[dt]
}

// 数据库元信息获取，如获取sql.DB、Dialect等
type Meta interface {
	// 根据数据库信息获取sql.DB
	GetSqlDb(*DbInfo) (*sql.DB, error)

	// 获取数据库方言，用于获取表结构等信息
	GetDialect(*DbConn) Dialect
}

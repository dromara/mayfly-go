package dbi

import "database/sql"

// 数据库元信息获取，如获取sql.DB、Dialect等
type Meta interface {
	// 根据数据库信息获取sql.DB
	GetSqlDb(*DbInfo) (*sql.DB, error)

	// 获取数据库方言，用于获取表结构等信息
	GetDialect(*DbConn) Dialect
}

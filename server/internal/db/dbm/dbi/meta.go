package dbi

import "database/sql"

// 数据库元信息获取，如获取sql.DB、Dialect等
type Meta interface {
	// 获取数据库服务实例信息
	GetSqlDb(*DbInfo) (*sql.DB, error)

	// 获取数据库方言，用于获取表结构等信息
	GetDialect(*DbConn) Dialect
}

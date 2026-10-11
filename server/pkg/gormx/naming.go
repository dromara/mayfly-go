package gormx

import "gorm.io/gorm/schema"

// MetaNamingStrategy 元数据库的「实体 → 表名」映射约定：统一 t_ 前缀、表名不加复数。
//
// 这里必须是单一真源：连接元数据库的每一处（生产启动、迁移端到端测试）都要取同一份配置。
// 漏配的后果不是报错，而是**静默测错对象**——没有 TableName() 方法的实体（如 DbSQLExec）
// 会被默认策略解析成 db_sql_exec 而非 t_db_sql_exec，于是测试里 AutoMigrate 改的列与
// 迁移内 raw SQL 写死的 t_xxx 根本不是同一张表：用例全绿也证明不了生产库能升级成功。
func MetaNamingStrategy() schema.NamingStrategy {
	return schema.NamingStrategy{
		TablePrefix:   "t_",
		SingularTable: true,
	}
}

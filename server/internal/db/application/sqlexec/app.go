package sqlexec

// SQL 执行域（保存的 SQL 脚本 + 执行 / 预检 / 回滚 / 导出 / 触发策略）的应用装配入口。
//
// 与 sync/transfer/schema 各子包同构：本子包自持 IOC getter，父包 application 只负责注册实现。
// 对库连接的依赖收敛成窄接口 DbConnReader（只取连接），运行时由父包的 Db 应用满足，
// 因此本子包不导入父包，杜绝循环依赖。

import (
	"context"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/pkg/ioc"
)

// DbConnReader 本域对父包 Db 应用的最小依赖：只按库定位取连接。
//
// 用窄接口而非直接依赖父包的 Db 接口，是为了让本子包不必导入父包
// （父包 application 单向导入本子包做 IOC 注册），从而杜绝循环依赖；
// 运行时由实现了完整 Db 的父包应用满足（inject:"T" 按结构可赋值解析，同 schema.DbConnReader）
type DbConnReader interface {
	GetDbConn(ctx context.Context, dbId uint64, dbName string) (*dbi.DbConn, error)
}

// GetDbSQLExecApp SQL 执行应用（执行 / 预检 / 回滚 / 导出 / 取消）
func GetDbSQLExecApp() DbSQLExec {
	return ioc.Get[DbSQLExec]()
}

// GetDbSQLApp 用户保存的 SQL 脚本应用
func GetDbSQLApp() DbSQL {
	return ioc.Get[DbSQL]()
}

// Package dbm 数据库管理模块入口：统一汇聚各方言包，提供连接池化的DbConn获取。
//
// 方言扩展机制（对扩展开放、对修改关闭）：
//   - dbi为契约层，定义 Meta/Dialect/Metadata/SQLGenerator/DumpHelper/Valuer 等接口与默认实现（Default*）；
//     application/domain 层仅依赖 dbi，不依赖任何具体方言包（依赖倒置）
//   - 各方言包（如mysql/postgres）自包含：自己声明DbType键、init中dbi.Register、
//     通过嵌入Default*覆写差异方法（Go方法集天然支持覆盖），方言内部差异不外溢
//
// 新增方言接入清单（以 newdb 为例，全部改动收敛在本目录内）：
//  1. 新建 dbm/newdb/ 包：
//     - meta.go:        实现 dbi.Meta（声明 DbTypeNewdb 键，init 中 dbi.Register；编译期断言 var _ dbi.Meta = (*Meta)(nil)）
//     - column.go:      数据类型清单（每个类型必须Name唯一且三要素Name/DataType/CommonType齐全，漏CommonType则跨方言迁移无法映射）
//     - dialect.go:     嵌入 dbi.DefaultDialect，必须覆写 GetSQLParser（默认返回nil由调用处fail-fast），按需覆写 Quoter/GetDumpHelper/GetSQLSplitter
//     - metadata.go:    实现 dbi.Metadata 元数据查询
//     - sqlgen.go:      实现 dbi.SQLGenerator（DDL/Insert生成）
//     - transfer.go:    实现 dbi.CommonTypeConverter（本方言类型→公共类型）
//     - helper.go:      按需覆写 dbi.DumpHelper（如pg不输出BEGIN/COMMIT、mssql的identity_insert）
//  2. 在本文件 import 追加一行： _ "mayfly-go/internal/db/dbm/newdb"
//  3. 在 dbm/dialect_registry_test.go 的 expectedDbTypes 追加新键
//  4. 运行 go test ./internal/db/dbm/ -run TestDialectRegistryCompleteness 校验注册完备性
//  5. 前端需在方言枚举/图标处同步新增（现有约定）
//
// 每个实现类型建议保留编译期断言（var _ dbi.Xxx = (*Xxx)(nil)），接口签名漂移在编译期暴露
package dbm

import (
	"context"
	_ "mayfly-go/internal/db/dbm/clickhouse"
	"mayfly-go/internal/db/dbm/dbi"
	_ "mayfly-go/internal/db/dbm/dm"
	_ "mayfly-go/internal/db/dbm/mssql"
	_ "mayfly-go/internal/db/dbm/mysql"
	_ "mayfly-go/internal/db/dbm/oracle"
	_ "mayfly-go/internal/db/dbm/postgres"
	_ "mayfly-go/internal/db/dbm/sqlite"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/pool"
)

var (
	poolGroup = pool.NewPoolGroup[*dbi.DbConn]()
)

// GetDbConn 从连接池中获取连接信息
func GetDbConn(ctx context.Context, dbId uint64, database string, getDbInfo func() (*dbi.DbInfo, error)) (*dbi.DbConn, error) {
	connId := dbi.GetDbConnId(dbId, database)

	pool, err := poolGroup.GetCachePool(connId, func() (*dbi.DbConn, error) {
		// 若缓存中不存在，则从回调函数中获取DbInfo
		dbInfo, err := getDbInfo()
		if err != nil {
			return nil, err
		}
		logx.Debugf("dbm - conn create, connId: %s, dbInfo: %v", connId, dbInfo)
		// 连接数据库
		return Conn(context.Background(), dbInfo)
	}, pool.WithIdleTimeout[*dbi.DbConn](0))

	if err != nil {
		return nil, err
	}
	// 从连接池中获取一个可用的连接
	return pool.Get(ctx)
}

// 使用指定dbInfo信息进行连接
func Conn(ctx context.Context, di *dbi.DbInfo) (*dbi.DbConn, error) {
	return di.Conn(ctx, dbi.GetMeta(di.Type))
}

// 根据实例id获取连接
func GetDbConnByInstanceId(ctx context.Context, instanceId uint64) *dbi.DbConn {
	for _, pool := range poolGroup.AllPool() {
		conn, err := pool.Get(ctx)
		if err != nil {
			continue
		}
		if conn.Info.InstanceId == instanceId {
			return conn
		}
	}
	return nil
}

// 删除db缓存并关闭该数据库所有连接
func CloseDb(dbId uint64, db string) {
	poolGroup.Close(dbi.GetDbConnId(dbId, db))
}

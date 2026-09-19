// Package dbm 数据库管理模块入口：统一汇聚各方言包，提供连接池化的DbConn获取。
//
// 方言扩展机制（对扩展开放、对修改关闭）：
//   - dbi为契约层，定义 DbBackend/Dialect/Metadata/MetadataProvider/ServerInfo/SQLGenerator/DumpHelper/Valuer 等接口与默认实现（Default*）；
//     application/domain 层仅依赖 dbi，不依赖任何具体方言包（依赖倒置）
//   - 各方言包（dbm/dialect/ 下，如 dialect/mysql、dialect/postgres）自包含：自己声明DbType键、init中dbi.RegisterBackend、
//     通过嵌入Default*覆写差异方法（Go方法集天然支持覆盖），方言内部差异不外溢
//
// 新增方言接入清单（以 newdb 为例，全部改动收敛在本目录内）：
//  1. 新建 dbm/dialect/newdb/ 包：
//     - backend.go:     实现 dbi.DbBackend（声明 DbTypeNewdb 键，init 中 dbi.RegisterBackend；编译期断言 var _ dbi.DbBackend = (*Backend)(nil)）
//     - column.go:      数据类型清单（每个类型必须Name唯一且三要素Name/DataType/TypeCategory齐全，漏TypeCategory则跨方言迁移无法映射）
//     - dialect.go:     嵌入 dbi.DefaultDialect，必须覆写 GetSQLParser（默认返回nil由调用处fail-fast），按需覆写 Quoter/GetDumpHelper/GetSQLSplitter
//     - metadata.go:    实现 dbi.ServerInfo + dbi.MetadataProvider 元数据查询
//     - sqlgen.go:      实现 dbi.SQLGenerator（DDL/Insert生成）
//     - backend.go init: 调用 dbi.RegisterTypeEngine 注册类型引擎（源类型+转换规则+降级策略）
//     - helper.go:      按需覆写 dbi.DumpHelper（如pg不输出BEGIN/COMMIT、mssql的identity_insert）
//     - ⚠️ GetCapabilities 必须显式声明或确认：嵌入的 BaseBackend 默认「全能力」，
//     不支持 schemas/索引等能力的方言（如 mysql/sqlite/clickhouse 型）漏覆写 = 谎报能力，
//     上层按能力声明决策会静默走错路径，参照最相近方言的声明拷贝修改
//  2. 在本文件 import 追加一行： _ "mayfly-go/internal/db/dbm/dialect/newdb"
//  3. 在 dbm/dialect_registry_test.go 的 expectedDbTypes 追加新键
//  4. 运行 go test ./internal/db/dbm/ -run TestDialectRegistryCompleteness 校验注册完备性
//  5. 前端需在方言枚举/图标处同步新增（现有约定）
//
// 新增元数据对象类型（视图/存储过程/序列等）：禁止往 MetadataProvider 必填接口加方法
// （会打断全部方言编译），按 dbi/metadata.go 头部的「可选能力接口」规约扩展
//
// 每个实现类型建议保留编译期断言（var _ dbi.Xxx = (*Xxx)(nil)），接口签名漂移在编译期暴露
package dbm

import (
	"context"

	// 方言插件聚合：新增方言在此追加一行 blank import（注册在各方言包 init 完成）
	_ "mayfly-go/internal/db/dbm/dialect/clickhouse"
	_ "mayfly-go/internal/db/dbm/dialect/dm"
	_ "mayfly-go/internal/db/dbm/dialect/mssql"
	_ "mayfly-go/internal/db/dbm/dialect/mysql"
	_ "mayfly-go/internal/db/dbm/dialect/oracle"
	_ "mayfly-go/internal/db/dbm/dialect/postgres"
	_ "mayfly-go/internal/db/dbm/dialect/sqlite"

	"mayfly-go/internal/db/dbm/dbi"
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
	dc, err := di.Conn(ctx, dbi.GetBackend(di.Type))
	if err != nil {
		return nil, err
	}
	// 注册实例ID到连接ID的反向索引
	dbi.RegisterInstanceConn(di.InstanceId, dc.Id)
	return dc, nil
}

// 根据实例id获取连接（使用反向索引 O(1) 查找，避免线性遍历所有连接池）
func GetDbConnByInstanceId(ctx context.Context, instanceId uint64) *dbi.DbConn {
	connIds := dbi.GetConnIdsByInstance(instanceId)
	for _, connId := range connIds {
		if p, ok := poolGroup.Get(connId); ok {
			conn, err := p.Get(ctx)
			if err != nil {
				continue
			}
			return conn
		}
	}
	return nil
}

// 删除db缓存并关闭该数据库所有连接
func CloseDb(dbId uint64, db string) {
	connId := dbi.GetDbConnId(dbId, db)
	// 注销实例索引（先获取连接信息以取得 instanceId）
	if p, ok := poolGroup.Get(connId); ok {
		if conn, err := p.Get(context.Background()); err == nil && conn != nil {
			dbi.UnregisterInstanceConn(conn.Info.InstanceId, connId)
		}
	}
	poolGroup.Close(connId)
}

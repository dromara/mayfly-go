// 方言注册完备性单测：防止"新方言漏注册/漏接入/数据类型漏定义"类缺陷回归。
//
//	背景：tinyblob漏注册导致dump字节翻倍失真、pg bit漏注册导致落string通道——
//	"漏注册"模式是dbm历史上反复出现的缺陷类别，本测试将其产品化为启动即校验。
//
// 覆盖三层断言：
//  1. 注册表完备：预期方言键（含别名 mariadb/gauss/kingbaseEs/vastbase）全部注册，且无意外键
//     （各方言init未被执行——如dbm.go漏加blank import——会在此暴露）
//  2. 方言能力完备：每个方言的 Quoter/Parser/Splitter/DumpHelper/SQLGenerator/Converter 全链非nil，
//     GetSQLParser的fail-fast语义（DefaultDialect返回nil由调用处panic）在此静态拦截
//  3. 数据类型完备：每方言类型清单非空、名字唯一（重复注册会静默覆盖）、
//     Name/DataType/CommonType三要素齐全
package dbm

import (
	"testing"

	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

// expectedDbTypes 预期注册的方言键全集（含同引擎别名）。
// 新增方言时须在此追加，并确认 dbm.go 已 blank import 方言包
var expectedDbTypes = []dbi.DbType{
	"mysql", "mariadb",
	"postgres", "gauss", "kingbaseEs", "vastbase",
	"sqlite", "mssql", "oracle", "dm", "clickhouse",
}

func TestDialectRegistryCompleteness(t *testing.T) {
	registered := dbi.GetRegisteredDbTypes()

	// 断言1：预期键全部注册（blank import遗漏在此暴露）
	regSet := make(map[dbi.DbType]bool, len(registered))
	for _, dt := range registered {
		regSet[dt] = true
	}
	for _, dt := range expectedDbTypes {
		require.True(t, regSet[dt], "方言 [%s] 未注册：检查方言包init是否执行（dbm.go是否blank import）", dt)
	}

	// 断言2：注册表无意外键（防止误注册/测试注册泄漏到生产注册表）
	expectSet := make(map[dbi.DbType]bool, len(expectedDbTypes))
	for _, dt := range expectedDbTypes {
		expectSet[dt] = true
	}
	for _, dt := range registered {
		require.True(t, expectSet[dt], "注册表出现未预期方言键 [%s]，请确认是否为本意并更新expectedDbTypes", dt)
	}

	// 断言3：每方言能力完备性
	for _, dt := range expectedDbTypes {
		meta := dbi.GetMeta(dt)
		require.NotNil(t, meta, "[%s] Meta为nil", dt)

		dialect := dbi.GetDialect(dt)
		require.NotNil(t, dialect, "[%s] Dialect为nil", dt)
		require.NotNil(t, dialect.Quoter(), "[%s] Quoter为nil", dt)
		require.NotNil(t, dialect.GetDumpHelper(), "[%s] DumpHelper为nil", dt)
		require.NotNil(t, dialect.GetSQLParser(), "[%s] SQLParser为nil（方言需显式覆写选择解析器）", dt)
		require.NotNil(t, dialect.GetSQLSplitter(), "[%s] SQLSplitter为nil", dt)
		require.NotNil(t, dialect.GetSQLGenerator(), "[%s] SQLGenerator为nil", dt)

		require.NotNil(t, meta.GetCommonTypeConverter(), "[%s] CommonTypeConverter为nil", dt)
	}

	// 断言4：每方言数据类型清单完备
	for _, dt := range expectedDbTypes {
		dataTypes := dbi.GetMeta(dt).GetDbDataTypes()
		require.NotEmpty(t, dataTypes, "[%s] 数据类型清单为空", dt)

		seen := make(map[string]bool, len(dataTypes))
		for _, column := range dataTypes {
			require.NotNil(t, column, "[%s] 存在nil数据类型", dt)
			require.NotEmpty(t, column.Name, "[%s] 存在无名称的数据类型", dt)
			require.NotEmpty(t, column.DataType, "[%s] 类型[%s]无DataType", dt, column.Name)
			require.NotEmpty(t, column.CommonType, "[%s] 类型[%s]无CommonType（跨方言迁移无法映射）", dt, column.Name)
			require.False(t, seen[column.Name], "[%s] 数据类型 [%s] 重复注册（会静默覆盖）", dt, column.Name)
			seen[column.Name] = true
		}
	}
}

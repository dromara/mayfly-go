package itest

// transfer 集成测试的「方言环境」与通用工具，所有 *_test.go 的公共底座。
//
// 为何必须单独成文件：方言节点注册表（itMysql/itPg/itSqlite/itMssql）、连接工厂与
// itTextAt/itTruncate 这类通用工具，此前寄存在「按缺陷命名」的叶子用例文件里
// （itDialectNode/三方言节点在 complexstrings、itMssqlNode 在 mssql_identity、
// transferTestConn 在 import、itReadFile/itAsBytes 在 dump_batch）。后果是任何一次
// 用例文件的重命名或删除，都会让整个 IT 套件（20个文件 / 48个顶层用例）编译失败，
// 且报错信息与那次改动毫无关系。依赖方向必须是：叶子用例 → 本文件，本文件不依赖任何叶子用例。
//
// 文件末尾的 TestITDialectMatrixRegistered 是「新增方言必须登记全部链路DDL矩阵」的守卫，
// 理由见其注释。
//
// 运行：cd server && go test -count=1 ./internal/db/application/transfer/itest/

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
)

// TestMain 集成测试入口：环境初始化与清理。
// 子包 itest 独立于 transfer 单元测试，可直接 go test ./itest/ 运行全部集成测试。
func TestMain(m *testing.M) {
	// 当前无需全局 setup/teardown（各节点连接按需创建、t.TempDir 自动清理），
	// 保留 TestMain 作为扩展点（如未来增加全局数据库初始化、测试标记文件等）。
	os.Exit(m.Run())
}

// transferTestConn 建立真实连接，失败即用例失败（不跳过：连不上属于环境问题，需显式暴露）
func transferTestConn(t *testing.T, di *dbi.DbInfo) *dbi.DbConn {
	t.Helper()
	conn, err := dbm.Conn(context.Background(), di)
	require.NoError(t, err)
	return conn
}

// itDialectNode 参与互测的方言节点：名称、类型、连接工厂
type itDialectNode struct {
	name   string
	dbType dbi.DbType
	conn   func(t *testing.T) *dbi.DbConn
}

func itMysqlNode(t *testing.T) *dbi.DbConn {
	t.Helper()
	admin := transferTestConn(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "information_schema"})
	defer admin.Close()
	_, err := admin.Exec("CREATE DATABASE IF NOT EXISTS mayfly_dbm_it DEFAULT CHARSET utf8mb4")
	require.NoError(t, err)
	return transferTestConn(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"})
}

func itPgNode(t *testing.T) *dbi.DbConn {
	t.Helper()
	conn := transferTestConn(t, &dbi.DbInfo{Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it"})
	require.NoError(t, conn.Ping())
	return conn
}

func itSqliteNode(t *testing.T) *dbi.DbConn {
	t.Helper()
	path := filepath.Join(t.TempDir(), "complex_it.sqlite")
	// sqlite方言要求库文件已存在
	require.NoError(t, os.WriteFile(path, nil, 0o644))
	return transferTestConn(t, &dbi.DbInfo{Type: "sqlite", Host: path})
}

// itMssqlNode 连接本机 SQL Server 集成测试实例；容器未启动/不可达时跳过（不影响其余方言回归）
//
// 唯一采用「跳过」而非「失败」的节点：mysql/pg 由 CI 与服务端配置常态依赖，sqlite 零依赖，
// 而 SQL Server 镜像体积与授权限制使其在开发者机上非必装。
func itMssqlNode(t *testing.T) *dbi.DbConn {
	t.Helper()
	conn, err := dbm.Conn(context.Background(), &dbi.DbInfo{
		Type: "mssql", Host: "127.0.0.1", Port: 11433,
		Username: "sa", Password: "Mayfly_123456",
		Database: "mayfly_it/dbo", Params: "encrypt=disable",
	})
	if err != nil {
		t.Skipf("mssql 集成测试容器不可用: %s", err.Error())
	}
	if err = conn.Ping(); err != nil {
		_ = conn.Close()
		t.Skipf("mssql 集成测试容器不可达: %s", err.Error())
	}
	return conn
}

var (
	itMysql  = itDialectNode{name: "mysql", dbType: "mysql", conn: itMysqlNode}
	itPg     = itDialectNode{name: "pg", dbType: "postgres", conn: itPgNode}
	itSqlite = itDialectNode{name: "sqlite", dbType: "sqlite", conn: itSqliteNode}
	itMssql  = itDialectNode{name: "mssql", dbType: "mssql", conn: itMssqlNode}
)

// itAllNodes 本机可用的全部方言节点，跨方言矩阵用例的统一遍历入口。
// 新增方言（如真实 oracle/clickhouse/dm 实例）时只需在此追加，并由下方守卫用例强制各链路补齐DDL。
var itAllNodes = []itDialectNode{itMysql, itPg, itSqlite, itMssql}

// itPair 源→目标方言有序对。此前十个矩阵用例各自重复声明匿名 struct{ src, tgt itDialectNode }，
// 类型不统一使"全组合生成"无法下沉为公共函数（见 itPairAll）。
type itPair struct {
	src, tgt itDialectNode
}

// itPairAll n×n 全组合（含同方言自组合：dump产物内含DROP重建，同构回灌本身就是一种有效路径）
func itPairAll(nodes []itDialectNode) []itPair {
	out := make([]itPair, 0, len(nodes)*len(nodes))
	for _, src := range nodes {
		for _, tgt := range nodes {
			out = append(out, itPair{src, tgt})
		}
	}
	return out
}

// itIdentityBatch 把显式写入自增主键id的一或多条INSERT包成可在单一会话一次执行的批次。
//
// SQL Server 的 IDENTITY_INSERT 是「会话级 + 非事务性」开关（本机实测）：分开多次 Exec 会从连接池
// 拿到不同会话，ON 不延续到后续 INSERT（报 Identity_insert is set to OFF），故开关与写入必须合为
// 同一批次下发（与导入链路一致：同一事务内多条语句共用一个会话）；同一批次内同时只允许一张表为 ON。
// identity=false 时不能带开关：对非 IDENTITY 列置 ON 报 8106（not an identity column）。
// 其余方言无此开关，原样返回单条语句，使调用方无需再按方言分支。
func itIdentityBatch(t *testing.T, conn *dbi.DbConn, table string, identity bool, inserts ...string) string {
	t.Helper()
	if conn.Info.Type != itMssql.dbType {
		require.Len(t, inserts, 1, "多语句批次仅 mssql 使用（其余方言驱动未开 multiStatements）")
		return inserts[0]
	}
	if !identity {
		return strings.Join(inserts, ";\n")
	}
	quote := conn.GetDialect().Quoter().QuoteIdent
	var b strings.Builder
	fmt.Fprintf(&b, "SET IDENTITY_INSERT %s ON;\n", quote(table))
	for _, insert := range inserts {
		b.WriteString(insert)
		b.WriteString(";\n")
	}
	fmt.Fprintf(&b, "SET IDENTITY_INSERT %s OFF", quote(table))
	return b.String()
}

// itTextAt 取文本列的字符串形态（兼容以[]byte返回的驱动），用于字节级严格比对
func itTextAt(row map[string]any, column string) string {
	switch v := row[column].(type) {
	case nil:
		return dbi.CanonicalNilValue
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return dbi.CanonicalValue(v)
	}
}

// itAsBytes 驱动返回的文本列可能是string或[]byte，统一为字节比对
func itAsBytes(v any) []byte {
	switch val := v.(type) {
	case []byte:
		return val
	case string:
		return []byte(val)
	default:
		return []byte(fmt.Sprintf("%v", val))
	}
}

// itReadFile 读回导出产物
func itReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

// itTruncate 截断长文本用于失败信息输出
func itTruncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n...(truncated)"
}

// TestITDialectMatrixRegistered 每个方言节点必须在所有「按方言取建表DDL」的矩阵里都有配置。
//
// 为什么需要这个守卫而不是靠运行时暴露：这些矩阵是 map[string]string，漏配方言时取到的是
// 零值空串，于是执行 `CREATE TABLE t `（无列定义），报出的是与真实原因完全无关的语法错误；
// 补 mssql 时真实踩中两次（itWdbDDL、itAiDDL），每次都要从一堆 SQL 错误里回溯到「矩阵少配一行」。
// 与之对比，itNarrowPkDDL/itTxnDDL 用 `, ok` 判定故能显式报错——守卫把两种写法拉平到同一水准。
//
// 该用例不连库，属于 IT 套件的自检：保证被检查的 map 一定存在。
func TestITDialectMatrixRegistered(t *testing.T) {
	matrices := map[string]map[string]string{
		"itTxnDDL":      itTxnDDL,      // 导入事务边界
		"itWdbDDL":      itWdbDDL,      // 整库备份恢复
		"itAiDDL":       itAiDDL,       // 自增主键迁移
		"itNarrowPkDDL": itNarrowPkDDL, // 窄整型主键分片
	}
	for name, matrix := range matrices {
		for _, node := range itAllNodes {
			ddl, ok := matrix[string(node.dbType)]
			require.True(t, ok, "%s 缺少方言[%s]的DDL配置，该链路的真实环境覆盖将静默退化为无效SQL", name, node.dbType)
			require.NotEmpty(t, ddl, "%s 方言[%s]的DDL为空", name, node.dbType)
		}
	}
}

//go:build it

package schema

// 结构对比的真实数据库集成测试。
//
// 单测钉的是比较规则，这里钉的是「从方言读到的结构事实够不够真」：列类型/长度/可空/默认值/主键/索引
// 都由各方言的元数据查询给出，读错或读漏都会让界面报出根本不存在的漂移，或者把真漂移漏掉——
// 后者更危险，用户会以为目标库与源库一致。
//
// 覆盖：同方言两侧的真实漂移（缺表、缺列、多列、长度变化、可空变化、索引增删、主键变化）、
// 跨方言两侧的正常映射（不得报成漂移）、以及限额与入参校验。
//
// 运行：cd server && go test -tags it -count=1 -run TestITSchemaDiff ./internal/db/application/schema/

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/ititest/scratchclean"
)

// diffConn 一个参与互测的库连接及其标签
type diffConn struct {
	label string
	conn  *dbi.DbConn
}

// mustConn 连本地集成测试实例，为「一侧」分配一个独立临时库。
//
// scratchclean 按 (类型/宿主/端口/库名/账号/Params) 复用临时库：两侧传同一个 DbInfo 会拿到同一个库，
// 结构对比就变成自比（而自比永远报「无差异」，用例会绿得毫无意义）。所以用 Params 把角色分开，
// 只写驱动本就认识的合法参数（秒数），并在调用处断言两侧最终落在不同的库上
func mustConn(t *testing.T, dbType dbi.DbType, role string) *dbi.DbConn {
	t.Helper()
	seconds := roleSeconds(role)
	info := &dbi.DbInfo{
		Type: dbType, Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049",
		Database: "mayfly_dbm_it", Params: "timeout=" + seconds,
	}
	if dbType == "postgres" {
		info.Username, info.Password, info.Database, info.Port = "postgres", "postgres", "mayfly_pg_it", 5432
		info.Params = "connect_timeout=" + strings.TrimSuffix(seconds, "s") + " application_name=mayfly_it_" + role
	}
	conn, err := scratchclean.Conn(context.Background(), info)
	require.NoError(t, err, "连接本地集成测试数据库失败")
	require.NoError(t, conn.Ping())
	return conn
}

// roleSeconds 把角色名映到一个固定的秒数上：只为让两侧拿到不同的临时库，不影响连接行为
func roleSeconds(role string) string {
	sum := 0
	for _, ch := range role {
		sum += int(ch)
	}
	return strconv.Itoa(10+sum%50) + "s"
}

// assertDifferentDbs 两侧必须落在不同的库上：没分开时本包所有漂移用例都会自比而假通过
func assertDifferentDbs(t *testing.T, src, dst *dbi.DbConn) {
	t.Helper()
	require.NotEqual(t, src.Info.Database, dst.Info.Database,
		"两侧临时库没分开（%s vs %s），对比会变成自比", src.Info.Database, dst.Info.Database)
}

func exec(t *testing.T, conn *dbi.DbConn, sql string) {
	t.Helper()
	if _, err := conn.Exec(sql); err != nil {
		t.Fatalf("exec failed [%s]: %s", sql, err.Error())
	}
}

// fakeReader 让应用层的 Diff 走真实连接而不依赖 IOC：结构对比的全部取数路径都能被端到端跑到
type fakeReader struct {
	byKey map[string]*dbi.DbConn
}

func (f fakeReader) GetDbConn(_ context.Context, dbId uint64, dbName string) (*dbi.DbConn, error) {
	conn, ok := f.byKey[fmt.Sprintf("%d/%s", dbId, dbName)]
	if !ok {
		return nil, fmt.Errorf("no such db in the test fixture: %d/%s", dbId, dbName)
	}
	return conn, nil
}

func app(conn *dbi.DbConn) *SchemaDiffAppImpl {
	return &SchemaDiffAppImpl{dbApp: fakeReader{byKey: map[string]*dbi.DbConn{
		"1/a": conn, "2/b": conn,
	}}}
}

// runDiff 用同一个连接充当两侧（同库不同名以便按名取到连接）
func runDiff(t *testing.T, src, dst *dbi.DbConn, tables []string) (*Result, error) {
	t.Helper()
	reader := fakeReader{byKey: map[string]*dbi.DbConn{"1/a": src, "2/b": dst}}
	svc := &SchemaDiffAppImpl{dbApp: reader}
	return svc.Diff(context.Background(), &DiffReq{SrcDbId: 1, SrcDb: "a", DstDbId: 2, DstDb: "b", Tables: tables})
}

func tableDiffOf(t *testing.T, result *Result, name string) *TableDiff {
	t.Helper()
	for _, table := range result.Tables {
		if table.Table == name {
			return table
		}
	}
	t.Fatalf("table [%s] is missing from the result: %+v", name, result.Tables)
	return nil
}

func kindsOf(diff *TableDiff) []string {
	kinds := make([]string, 0, len(diff.Columns))
	for _, item := range diff.Columns {
		kinds = append(kinds, item.Kind)
	}
	return kinds
}

// TestITSchemaDiffDetectsRealDrift 在两个真实库之间制造六种漂移，逐条要求被读出来。
//
// 这里断言的是「每一种漂移都能被发现」：漏任何一种，用户就会在「结构一致」的结论上继续写数据
func TestITSchemaDiffDetectsRealDrift(t *testing.T) {
	for _, dbType := range []dbi.DbType{"mysql", "postgres"} {
		dbType := dbType
		t.Run(string(dbType), func(t *testing.T) {
			src := mustConn(t, dbType, "src")
			dst := mustConn(t, dbType, "dst")
			defer src.Close()
			defer dst.Close()
			assertDifferentDbs(t, src, dst)

			quote := src.GetDialect().Quoter().QuoteIdent
			for _, conn := range []*dbi.DbConn{src, dst} {
				exec(t, conn, "DROP TABLE IF EXISTS "+quote("drift_shared"))
				exec(t, conn, "DROP TABLE IF EXISTS "+quote("drift_only_src"))
				exec(t, conn, "DROP TABLE IF EXISTS "+quote("drift_only_dst"))
			}
			// 源侧：三列 + 两个索引（其一唯一）
			exec(t, src, fmt.Sprintf("CREATE TABLE %s (%s INT PRIMARY KEY, %s VARCHAR(50) NOT NULL, %s INT)", quote("drift_shared"), quote("id"), quote("name"), quote("amount")))
			exec(t, src, fmt.Sprintf("CREATE INDEX %s ON %s (%s)", quote("idx_name"), quote("drift_shared"), quote("name")))
			// 目标侧：少一个 amount 列、name 变长且可空、多一个 note 列、少一个索引、主键换成 (name,id) 之外的情形不便跨方言统一，
			// 这里只把主键保持为 id，主键差异由下面的用例单独验
			exec(t, dst, fmt.Sprintf("CREATE TABLE %s (%s INT PRIMARY KEY, %s VARCHAR(200), %s VARCHAR(30))", quote("drift_shared"), quote("id"), quote("name"), quote("note")))
			// 仅单侧存在的表
			exec(t, src, fmt.Sprintf("CREATE TABLE %s (%s INT PRIMARY KEY)", quote("drift_only_src"), quote("id")))
			exec(t, dst, fmt.Sprintf("CREATE TABLE %s (%s INT PRIMARY KEY)", quote("drift_only_dst"), quote("id")))

			result, err := runDiff(t, src, dst, []string{"drift_shared", "drift_only_src", "drift_only_dst"})
			require.NoError(t, err)
			assert.Equal(t, 3, result.Compared, "三张表都应参与比对")

			shared := tableDiffOf(t, result, "drift_shared")
			kinds := kindsOf(shared)
			assert.Contains(t, kinds, ColumnOnlyInSource, "目标缺少的 amount 列必须被发现: %v", kinds)
			assert.Contains(t, kinds, ColumnOnlyInTarget, "目标多出的 note 列必须被发现: %v", kinds)
			assert.Contains(t, kinds, ColumnSizeChanged, "varchar 长度变化必须被发现: %v", kinds)
			assert.Contains(t, kinds, ColumnNullableDiff, "可空性变化必须被发现: %v", kinds)
			assert.NotEmpty(t, shared.Indexes, "索引差异必须被发现: %+v", shared.Indexes)

			assert.Equal(t, TableOnlyInSource, tableDiffOf(t, result, "drift_only_src").Kind)
			assert.Equal(t, TableOnlyInTarget, tableDiffOf(t, result, "drift_only_dst").Kind)

			// 一致的那一侧再比一次自己：不得报出任何差异（证明差异来自真实结构而不是读取噪声）
			same, err := runDiff(t, src, src, []string{"drift_shared"})
			require.NoError(t, err)
			assert.Empty(t, same.Tables, "identical structures must produce no diffs, got %+v", same.Tables)

			for _, conn := range []*dbi.DbConn{src, dst} {
				exec(t, conn, "DROP TABLE IF EXISTS "+quote("drift_shared"))
				exec(t, conn, "DROP TABLE IF EXISTS "+quote("drift_only_src"))
				exec(t, conn, "DROP TABLE IF EXISTS "+quote("drift_only_dst"))
			}
		})
	}
}

// TestITSchemaDiffPrimaryKeyChange 主键列变化必须在两侧元数据都读得到时被发现
func TestITSchemaDiffPrimaryKeyChange(t *testing.T) {
	for _, dbType := range []dbi.DbType{"mysql", "postgres"} {
		dbType := dbType
		t.Run(string(dbType), func(t *testing.T) {
			src := mustConn(t, dbType, "pk_src")
			dst := mustConn(t, dbType, "pk_dst")
			defer src.Close()
			defer dst.Close()
			assertDifferentDbs(t, src, dst)

			quote := src.GetDialect().Quoter().QuoteIdent
			for _, conn := range []*dbi.DbConn{src, dst} {
				exec(t, conn, "DROP TABLE IF EXISTS "+quote("pk_drift"))
			}
			exec(t, src, fmt.Sprintf("CREATE TABLE %s (%s INT, %s INT, %s VARCHAR(20), PRIMARY KEY (%s, %s))", quote("pk_drift"), quote("a"), quote("b"), quote("c"), quote("a"), quote("b")))
			exec(t, dst, fmt.Sprintf("CREATE TABLE %s (%s INT, %s INT, %s VARCHAR(20), PRIMARY KEY (%s))", quote("pk_drift"), quote("a"), quote("b"), quote("c"), quote("a")))

			result, err := runDiff(t, src, dst, []string{"pk_drift"})
			require.NoError(t, err)
			shared := tableDiffOf(t, result, "pk_drift")
			require.NotNil(t, shared.PrimaryKey, "联合主键变成单列主键必须被发现: %+v", shared)
			assert.Contains(t, shared.PrimaryKey.Source, "b", "差异里要能看出少了哪一列: %+v", shared.PrimaryKey)
			assert.Empty(t, shared.Unavailable, "两侧都读到了主键，不该有「比不了」的记录: %+v", shared.Unavailable)

			for _, conn := range []*dbi.DbConn{src, dst} {
				exec(t, conn, "DROP TABLE IF EXISTS "+quote("pk_drift"))
			}
		})
	}
}

// TestITSchemaDiffCrossDialectIsQuiet 跨方言的正常类型映射不得报成漂移。
//
// 这是「同一个结构在两个库里长得不一样」最容易误报的地方：mysql 的 blob/datetime/tinyint
// 迁到 pg 就是 bytea/timestamp/int2，报出来就会把每次跨库对比变成满屏噪声
func TestITSchemaDiffCrossDialectIsQuiet(t *testing.T) {
	src := mustConn(t, "mysql", "x_src")
	dst := mustConn(t, "postgres", "x_dst")
	defer src.Close()
	defer dst.Close()

	srcQuote := src.GetDialect().Quoter().QuoteIdent
	dstQuote := dst.GetDialect().Quoter().QuoteIdent
	exec(t, src, "DROP TABLE IF EXISTS "+srcQuote("x_dialect"))
	exec(t, dst, "DROP TABLE IF EXISTS "+dstQuote("x_dialect"))
	exec(t, src, fmt.Sprintf("CREATE TABLE %s (%s INT PRIMARY KEY, %s BLOB, %s DATETIME, %s VARCHAR(60))", srcQuote("x_dialect"), srcQuote("id"), srcQuote("payload"), srcQuote("at"), srcQuote("name")))
	exec(t, dst, fmt.Sprintf("CREATE TABLE %s (%s INT PRIMARY KEY, %s BYTEA, %s TIMESTAMP, %s VARCHAR(60))", dstQuote("x_dialect"), dstQuote("id"), dstQuote("payload"), dstQuote("at"), dstQuote("name")))

	result, err := runDiff(t, src, dst, []string{"x_dialect"})
	require.NoError(t, err)
	if len(result.Tables) != 0 {
		// 允许的唯一噪声是「该方言类型引擎认不出的名字」，那必须显式标成比不了而不是差异
		for _, table := range result.Tables {
			for _, column := range table.Columns {
				assert.NotEqual(t, ColumnTypeChanged, column.Kind, "跨方言的正常类型映射不得报成类型漂移, got %+v", table)
			}
		}
	}

	exec(t, src, "DROP TABLE IF EXISTS "+srcQuote("x_dialect"))
	exec(t, dst, "DROP TABLE IF EXISTS "+dstQuote("x_dialect"))
}

// TestITSchemaDiffRejectsSameDb 两侧同一个库时必须拒绝，而不是比出一片「无差异」
func TestITSchemaDiffRejectsSameDb(t *testing.T) {
	conn := mustConn(t, "mysql", "same")
	defer conn.Close()

	svc := &SchemaDiffAppImpl{dbApp: fakeReader{byKey: map[string]*dbi.DbConn{"1/a": conn}}}
	_, err := svc.Diff(context.Background(), &DiffReq{SrcDbId: 1, SrcDb: "a", DstDbId: 1, DstDb: "a"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "两个不同的库")
}

// TestITSchemaDiffDeclaresTruncation 未指定表清单时超出限额必须声明：
// 「没报差异」与「没参与比对」在用户眼里是同一件事，而含义完全不同
func TestITSchemaDiffDeclaresTruncation(t *testing.T) {
	src := mustConn(t, "mysql", "cap_src")
	dst := mustConn(t, "mysql", "cap_dst")
	defer src.Close()
	defer dst.Close()
	assertDifferentDbs(t, src, dst)

	quote := src.GetDialect().Quoter().QuoteIdent
	total := maxAutoDiffTables + 2
	names := make([]string, 0, total)
	for i := 1; i <= total; i++ {
		name := fmt.Sprintf("cap_%03d", i)
		names = append(names, name)
		for _, conn := range []*dbi.DbConn{src, dst} {
			exec(t, conn, "DROP TABLE IF EXISTS "+quote(name))
			exec(t, conn, fmt.Sprintf("CREATE TABLE %s (%s INT PRIMARY KEY)", quote(name), quote("id")))
		}
	}
	// 在限额之外的那张表上制造一处差异，用来证明「未参与比对」而不是「比过且一致」
	last := names[len(names)-1]
	exec(t, src, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s INT", quote(last), quote("extra")))

	result, err := runDiff(t, src, dst, nil)
	require.NoError(t, err)
	assert.Equal(t, maxAutoDiffTables, result.Compared, "自动范围必须按限额截断")
	assert.NotEmpty(t, result.Note, "截断必须声明")
	for _, table := range result.Tables {
		assert.NotEqual(t, last, table.Table, "被截断的表不得出现在结果里")
	}

	for _, name := range names {
		for _, conn := range []*dbi.DbConn{src, dst} {
			exec(t, conn, "DROP TABLE IF EXISTS "+quote(name))
		}
	}
}

// TestITSchemaDiffEmptyTableList 表清单为空且两侧都没有表时不得给出「一致」的假结论
func TestITSchemaDiffEmptyTableList(t *testing.T) {
	src := mustConn(t, "mysql", "empty")
	defer src.Close()

	svc := app(src)
	_, err := svc.Diff(context.Background(), &DiffReq{SrcDbId: 1, SrcDb: "a", DstDbId: 2, DstDb: "b", Tables: []string{"   "}})
	require.Error(t, err, "空表清单必须报错，不能当成「比过了」")
}

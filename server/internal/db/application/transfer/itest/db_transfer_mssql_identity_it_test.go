package itest

// mssql IDENTITY 自增列的 导出→导入→分片并行迁移 真实链路集成测试（本机 SQL Server 容器，未启动自动跳过）。
//
// 为何必须补这组用例：pg 的序列需要导出侧显式校正（见 db_transfer_autoincrement_it_test.go），
// mssql 走的是另一套机制 `set identity_insert <t> on/off`，其正确性有三个只有真实库才能验证的前提：
//  1. IDENTITY_INSERT 是**会话级且同一时刻只允许一张表为 ON**，多表 dump 的 on/off 若交叠或未闭合，
//     第二张表导入即报 "already ON for table"（SET 语句非事务性，回滚也不撤销）；
//  2. 显式插入 id 后 SQL Server 是否自动把标识种子推进到最大值——决定 pg 上那类
//     「分片并行导入后序列被下调 → 首条业务写入撞主键」的缺陷在 mssql 上是否根本不存在；
//  3. 元数据 IDENTITY 属性必须能穿过 dump→DDL 还原，否则目标列静默退化为普通整型，
//     业务插入不带 id 的行直接报 NOT NULL 违反。
//
// 运行：cd server && go test -tags it -count=1 -run TestITMssqlIdentity ./internal/db/application/transfer/

import (
	"mayfly-go/internal/db/application/transfer"
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
)

const (
	itMsTableA    = "it_ms_id_a"
	itMsTableB    = "it_ms_id_b"
	itMsPlainTale = "it_ms_plain"
)

// itMsIdentityRe identity_insert 开关语句（表名可能带方括号引用）
var itMsIdentityRe = regexp.MustCompile(`(?i)^\s*set\s+identity_insert\s+(\[?[\w]+]?)\s+(on|off)\s*;?\s*$`)

// itMssql / itMssqlNode（方言节点与连接工厂）见 it_env_it_test.go

// itMsPrepare 建表（identity 与否可选）并写入 id=1..rowCount 的行
func itMsPrepare(t *testing.T, conn *dbi.DbConn, table string, withIdentity bool, rowCount int) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, err := conn.Exec("DROP TABLE IF EXISTS " + quote(table))
	require.NoError(t, err, "[%s] 清理表失败", table)

	idCol := "INT NOT NULL"
	if withIdentity {
		idCol = "INT IDENTITY(1,1) NOT NULL"
	}
	_, err = conn.Exec(fmt.Sprintf("CREATE TABLE %s (id %s PRIMARY KEY, val NVARCHAR(64) NULL)", quote(table), idCol))
	require.NoError(t, err, "[%s] 建表失败（identity=%v）", table, withIdentity)
	if rowCount == 0 {
		return
	}

	// 写入必须作为一个批次执行（会话级开关的实测语义见 itIdentityBatch）
	inserts := make([]string, 0, (rowCount+499)/500)
	for start := 1; start <= rowCount; start += 500 {
		end := min(start+499, rowCount)
		values := make([]string, 0, end-start+1)
		for i := start; i <= end; i++ {
			values = append(values, fmt.Sprintf("(%d, N'v%03d')", i, i))
		}
		inserts = append(inserts, fmt.Sprintf("INSERT INTO %s (id, val) VALUES %s", quote(table), strings.Join(values, ",")))
	}
	_, err = conn.Exec(itIdentityBatch(t, conn, table, withIdentity, inserts...))
	require.NoError(t, err, "[%s] 写入 id 1..%d 失败", table, rowCount)
}

func itMsDropTable(t *testing.T, conn *dbi.DbConn, table string) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, _ = conn.Exec("DROP TABLE IF EXISTS " + quote(table))
}

// itMsDump 用生产 DumpDbScript 导出指定表清单（同构 mssql→mssql）
func itMsDump(t *testing.T, conn *dbi.DbConn, tables []string, dumpDDL, dumpData bool) string {
	t.Helper()
	var buf strings.Builder
	dump := &dto.DumpDb{
		DbName:       conn.Info.Database,
		Tables:       tables,
		DumpDDL:      dumpDDL,
		DumpData:     dumpData,
		TargetDbType: "mssql",
		Writer:       &buf,
	}
	require.NoError(t, transfer.DumpDbScript(context.Background(), conn, dump), "dump 失败")
	return buf.String()
}

// itMsRowCount 读回表行数
func itMsRowCount(t *testing.T, conn *dbi.DbConn, table string) int64 {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.Query("SELECT COUNT(*) AS cnt FROM " + quote(table))
	require.NoError(t, err, "[%s] 查询行数失败", table)
	require.Len(t, rows, 1)
	cnt, ok := dbi.ValToInt64(rows[0]["cnt"])
	require.True(t, ok, "[%s] 行数取值形态异常: %T", table, rows[0]["cnt"])
	return cnt
}

// itMsIdentityColumn 查表的标识列数量（结构迁移后必须仍为1）
func itMsIdentityColumn(t *testing.T, conn *dbi.DbConn, table string) int64 {
	t.Helper()
	_, rows, err := conn.Query(fmt.Sprintf(
		"SELECT COUNT(*) AS cnt FROM sys.columns WHERE object_id = OBJECT_ID('dbo.%s') AND is_identity = 1", table))
	require.NoError(t, err, "[%s] 查询标识列失败", table)
	require.Len(t, rows, 1)
	cnt, ok := dbi.ValToInt64(rows[0]["cnt"])
	require.True(t, ok, "标识列数取值形态异常: %T", rows[0]["cnt"])
	return cnt
}

// TestITMssqlIdentityDumpProduct 多表 dump 的 identity_insert 开关必须严格不交叠且逐表闭合。
//
// SQL Server 同一会话同时只允许一张表处于 IDENTITY_INSERT ON，交叠会让后一张表导入直接失败；
// 无标识列的表若被输出开关语句，则报 "does not have the identity property"。
// 批量刷新按 100 行切分，故单表的 on 可能出现多次（同表重复 ON 是幂等的），off 必须恰好一次
func TestITMssqlIdentityDumpProduct(t *testing.T) {
	conn := itMssqlNode(t)
	defer conn.Close()

	itMsPrepare(t, conn, itMsTableA, true, 150)
	defer itMsDropTable(t, conn, itMsTableA)
	itMsPrepare(t, conn, itMsTableB, true, 3)
	defer itMsDropTable(t, conn, itMsTableB)
	itMsPrepare(t, conn, itMsPlainTale, false, 3)
	defer itMsDropTable(t, conn, itMsPlainTale)

	script := itMsDump(t, conn, []string{itMsTableA, itMsTableB, itMsPlainTale}, true, true)
	require.Contains(t, strings.ToUpper(script), "INSERT INTO", "产物应含数据，否则本用例失去鉴别力")

	splitter := conn.GetDialect().GetSQLSplitter()
	var onTable string
	onCount, offCount := 0, 0
	for _, stmt := range itSplitStmts(t, conn, script) {
		// 产物的分段注释头会与其后语句切在同一条，必须先按生产口径掩码归一化再判类型
		m := itMsIdentityRe.FindStringSubmatch(sqlparser.NormalizeStmtText(splitter, stmt))
		if m == nil {
			continue
		}
		table, state := strings.Trim(m[1], "[]"), strings.ToLower(m[2])
		assert.NotEqual(t, itMsPlainTale, table, "无标识列的表不应输出 identity_insert: %s", stmt)
		if state == "on" {
			if onTable != "" {
				assert.Equal(t, onTable, table,
					"identity_insert 交叠：[%s] 尚未关闭就对 [%s] 置 ON，导入会报 already ON for table", onTable, table)
			}
			onTable = table
			onCount++
			continue
		}
		require.Equal(t, table, onTable, "[%s] 未处于 ON 状态却输出 off（前一张表的开关未闭合）", table)
		onTable = ""
		offCount++
	}

	assert.Empty(t, onTable, "产物结束时会话仍留有 identity_insert ON，同连接后续导入其他自增表会失败")
	assert.Equal(t, 2, offCount, "两张标识表应各输出一次 off")
	assert.GreaterOrEqual(t, onCount, offCount, "每次刷新批次各输出一条 on")

	// 产物必须能整体导入：导入侧按500条批级提交，多张标识表的开关会落进同一事务，
	// 而本机实测同一事务内交叠置ON必报 8107（IDENTITY_INSERT is already ON for table）
	require.NoError(t, (&transfer.DbTransferAppImpl{}).ImportDumpStream(context.Background(), 0, conn, strings.NewReader(script)),
		"多表dump产物导入失败\n%s", itTruncate(script, 4000))
	assert.Equal(t, int64(150), itMsRowCount(t, conn, itMsTableA), "恢复后表A行数不符")
	assert.Equal(t, int64(3), itMsRowCount(t, conn, itMsTableB), "恢复后表B行数不符")
	assert.Equal(t, int64(3), itMsRowCount(t, conn, itMsPlainTale), "恢复后无标识表行数不符")
	assert.Equal(t, int64(1), itMsIdentityColumn(t, conn, itMsTableA), "恢复后表A标识属性丢失")
}

// TestITMssqlIdentityParallelShardImport mssql 自增表按主键分片并行导入：
// 结构阶段必须保住 IDENTITY 属性，数据阶段并行完成后业务写入要能续接最大 id。
//
// 这是 pg 侧序列校正竞态的对照组：mssql 无独立序列对象，显式插入标识值时由服务端推进种子，
// 因此结论（无需校正语句）在此实测钉死——若某天不再成立（如驱动/版本行为变化）用例会变红
func TestITMssqlIdentityParallelShardImport(t *testing.T) {
	const (
		rows        = 200
		shardRows   = 40 // 200/40 → 5个分片
		cycles      = 2
		concurrency = 8
	)
	conn := itMssqlNode(t)
	defer conn.Close()

	origTargetRows := dbi.ShardTargetRows
	dbi.ShardTargetRows = shardRows
	defer func() { dbi.ShardTargetRows = origTargetRows }()

	app := &transfer.DbTransferAppImpl{}
	for cycle := 0; cycle < cycles; cycle++ {
		round := fmt.Sprintf("第%d轮", cycle+1)
		itMsPrepare(t, conn, itAiTable, true, rows)

		wheres := app.PlanTableShards(context.Background(), 0, conn, itAiTable, rows)
		require.GreaterOrEqual(t, len(wheres), 3, "%s：主键分片未生效（并行导入退化为单任务）", round)

		ddlScript := itDumpTable(t, conn, itAiTable, "mssql", true, false, "")
		shards := make([]string, 0, len(wheres))
		for _, where := range wheres {
			shard := itDumpTable(t, conn, itAiTable, "mssql", false, true, where)
			require.NotContains(t, shard, "CREATE TABLE", "%s：分片段产物不应含DDL", round)
			require.Contains(t, shard, "identity_insert", "%s：标识表的数据产物应含 identity_insert 开关", round)
			shards = append(shards, shard)
		}

		require.NoError(t, app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader(ddlScript)),
			"%s：结构阶段导入失败", round)
		assert.Equal(t, int64(0), itAiRowCount(t, conn), "%s：结构阶段不应携带数据", round)
		assert.Equal(t, int64(1), itMsIdentityColumn(t, conn, itAiTable),
			"%s：结构迁移后标识属性丢失（列退化为普通整型，业务插入不带id将报NOT NULL）", round)

		eg := new(errgroup.Group)
		eg.SetLimit(concurrency)
		for i, shard := range shards {
			i, shard := i, shard
			eg.Go(func() error {
				if err := app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader(shard)); err != nil {
					return fmt.Errorf("分片%d导入失败: %w\n%s", i+1, err, itTruncate(shard, 2000))
				}
				return nil
			})
		}
		require.NoError(t, eg.Wait(), "%s：分片并行导入失败", round)

		assert.Equal(t, int64(rows), itAiRowCount(t, conn), "%s：并行导入后行数不一致", round)
		assert.Equal(t, int64(rows), itAiMaxId(t, conn), "%s：并行导入后最大id不一致", round)
		assert.Equal(t, int64(rows), itAiDistinctIdCount(t, conn), "%s：id存在重叠或空洞", round)

		require.NoError(t, itAiInsertNext(t, conn),
			"%s：并行分片导入后标识种子未续接到全表最大id（首条业务写入即撞主键）", round)
		assert.Equal(t, int64(rows+1), itAiMaxId(t, conn), "%s：并行导入后新行id不一致", round)
		itAiDrop(t, conn)
	}
}

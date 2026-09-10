package itest

// 导出侧「行数+字节双预算」分批策略的真实产物集成测试（驱动生产 DumpDbScript）。
//
// 本文件直接调用生产 DumpDbScript，对真实导出产物做断言（早期版本曾在测试内**复刻**生产分批循环，
// 生产实现变更后复刻版依然绿——实测去掉预算后它无法反映真实产物形态，属无效守卫，已删除）：
//   - 写入块尺寸与次数（分批正确的直接证据：巨型块=内存峰值与单包超限的根源）
//   - 产物中INSERT语句条数（行数预算100/批、字节预算8MB各自的生效结果）
//   - 真实导入回环后的行数与逐值一致性
//
// 运行：cd server && go test -tags it -count=1 -run TestITDumpRealProduct ./internal/db/application/transfer/

import (
	"mayfly-go/internal/db/application/transfer"
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
)

// itChunkProbe 记录生产dump每次写入的块尺寸：分批策略正确时块尺寸有界（单条批量INSERT），
// 退化为整表一次性flush时会出现与数据总量同量级的巨型块
type itChunkProbe struct {
	next   io.Writer
	chunks int
	max    int
	total  int64
}

func (p *itChunkProbe) Write(b []byte) (int, error) {
	if len(b) > p.max {
		p.max = len(b)
	}
	p.chunks++
	p.total += int64(len(b))
	return p.next.Write(b)
}

// itSplitStmts 用真实方言切割器切开导出产物，返回语句列表
func itSplitStmts(t *testing.T, conn *dbi.DbConn, script string) []string {
	t.Helper()
	splitter := conn.GetDialect().GetSQLSplitter()
	var stmts []string
	require.NoError(t, splitter.SplitSQL(strings.NewReader(script), func(s string) error {
		stmts = append(stmts, s)
		return nil
	}))
	return stmts
}

// itInsertStmts 过滤出产物中的INSERT语句（按生产归一化取首关键字，注释前缀不干扰判定）
func itInsertStmts(splitter sqlparser.SQLSplitter, stmts []string) []string {
	inserts := make([]string, 0, len(stmts))
	for _, s := range stmts {
		if strings.HasPrefix(sqlparser.NormalizeStmtText(splitter, s), "INSERT") {
			inserts = append(inserts, s)
		}
	}
	return inserts
}

// itDumpToFile 驱动生产DumpDbScript导出到真实文件，返回写入块探针与产物大小
func itDumpToFile(t *testing.T, conn *dbi.DbConn, tables []string, targetDbType dbi.DbType, dumpDDL bool, path string) (*itChunkProbe, int64) {
	t.Helper()
	f, err := os.Create(path)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, f.Close())
	}()

	probe := &itChunkProbe{next: bufio.NewWriterSize(f, 1<<20)}
	require.NoError(t, transfer.DumpDbScript(context.Background(), conn, &dto.DumpDb{
		DbName:       conn.Info.Database,
		Tables:       tables,
		DumpDDL:      dumpDDL,
		DumpData:     true,
		TargetDbType: targetDbType,
		Writer:       probe,
	}), "导出失败")
	// bufio刷盘，确保文件完整可读回
	if bw, ok := probe.next.(*bufio.Writer); ok {
		require.NoError(t, bw.Flush())
	}
	st, err := os.Stat(path)
	require.NoError(t, err)
	return probe, st.Size()
}

// TestITDumpRealProductRowBudget 2万行×~1KB同构导出：真实产物必须是100行/批的批量INSERT，
// 且无任何巨型写入块（流式分批生效）；再异构导入sqlite验证行数与逐值一致
func TestITDumpRealProductRowBudget(t *testing.T) {
	conn := itMysqlNode(t)
	defer conn.Close()

	const rowCount = 20000
	table := "it_dump_row_budget"
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table)))
	require.NoError(t, err)
	_, err = conn.Exec(fmt.Sprintf("CREATE TABLE %s (id INT AUTO_INCREMENT PRIMARY KEY, v_text VARCHAR(2048), v_dt DATETIME) DEFAULT CHARSET=utf8mb4", quote(table)))
	require.NoError(t, err)

	// 500行/批写入减少往返；值含中文/单双引号/反斜杠，产物含转义膨胀
	for batch := 0; batch < rowCount/500; batch++ {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("INSERT INTO %s (v_text, v_dt) VALUES ", quote(table)))
		for i := 0; i < 500; i++ {
			idx := batch*500 + i + 1
			if i > 0 {
				sb.WriteString(",")
			}
			text := fmt.Sprintf("row-%06d-中文数据'quote\"dquote\\slash %s", idx, strings.Repeat("x", 900))
			sb.WriteString(fmt.Sprintf("('%s', '2026-09-05 12:00:00')", strings.ReplaceAll(text, "'", "''")))
		}
		_, err := conn.Exec(sb.String())
		require.NoError(t, err)
	}

	path := filepath.Join(t.TempDir(), table+".sql")
	probe, size := itDumpToFile(t, conn, []string{table}, "mysql", true, path)

	// 产物体积与源数据同量级：偏小说明静默截断/丢行
	assert.Greater(t, size, int64(15<<20), "2万行×~1KB产物应≥15MB（偏小说明导出丢数据）")
	// 分批生效的直接证据：写入次数≈批次数，且无巨型块
	assert.GreaterOrEqual(t, probe.chunks, rowCount/dbi.DumpInsertBatchRows, "每次flush应产生一个写入块")
	assert.Less(t, probe.max, 4<<20, "最大写入块应<4MB（出现整表级巨型块说明行数/字节预算失效，内存峰值与单包上限都会失控）")

	script := itReadFile(t, path)
	inserts := itInsertStmts(conn.GetDialect().GetSQLSplitter(), itSplitStmts(t, conn, script))
	assert.Equal(t, rowCount/dbi.DumpInsertBatchRows, len(inserts),
		"真实产物必须是%d行/批的批量INSERT", dbi.DumpInsertBatchRows)
	t.Logf("产物 %.1fMB, 写入块 %d 次(最大 %.2fMB), INSERT语句 %d 条",
		float64(size)/(1<<20), probe.chunks, float64(probe.max)/(1<<20), len(inserts))

	// 异构导入sqlite：由生产ImportDumpStream从文件流式恢复，行数与抽样内容必须一致
	sq := itSqliteNode(t)
	defer sq.Close()
	sqPath := filepath.Join(t.TempDir(), table+"_sqlite.sql")
	sqProbe, sqSize := itDumpToFile(t, conn, []string{table}, "sqlite", true, sqPath)
	assert.Less(t, sqProbe.max, 4<<20, "sqlite目标产物同样不应出现巨型写入块")
	_, err = sq.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", sq.GetDialect().Quoter().QuoteIdent(table)))
	require.NoError(t, err)
	require.NoError(t, (&transfer.DbTransferAppImpl{}).ImportDumpStream(context.Background(), 0, sq,
		bufio.NewReaderSize(strings.NewReader(itReadFile(t, sqPath)), 1<<20)), "2万行异构导入sqlite失败")
	assert.Equal(t, int64(rowCount), itCountLeak(t, sq, table), "异构导入sqlite行数不一致")
	assert.Greater(t, sqSize, int64(15<<20), "sqlite产物体积异常偏小")

	_, srcSample, err := conn.Query(fmt.Sprintf("SELECT v_text FROM %s WHERE id = 12345", quote(table)))
	require.NoError(t, err)
	require.Len(t, srcSample, 1)
	_, dstSample, err := sq.Query(fmt.Sprintf("SELECT v_text FROM %s WHERE id = 12345", sq.GetDialect().Quoter().QuoteIdent(table)))
	require.NoError(t, err)
	require.Len(t, dstSample, 1)
	assert.Equal(t, string(itAsBytes(srcSample[0]["v_text"])), string(itAsBytes(dstSample[0]["v_text"])),
		"抽样内容必须逐字节一致")

	_, _ = conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table)))
}

// TestITDumpRealProductLargeValueBudget 单行大值（15行×5MB）必须触发字节预算：
// 每批最多2行（2×5MB≥8MB预算）。实测去掉字节预算后产物退化为1条75MB语句，
// 生产导入链路直接失败（超mysql max_allowed_packet），本用例即变红
func TestITDumpRealProductLargeValueBudget(t *testing.T) {
	conn := itMysqlNode(t)
	defer conn.Close()

	table := "it_dump_big_value"
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table)))
	require.NoError(t, err)
	_, err = conn.Exec(fmt.Sprintf("CREATE TABLE %s (id INT PRIMARY KEY, v_text LONGTEXT)", quote(table)))
	require.NoError(t, err)

	const rowCount = 15
	const valueSize = 5 << 20
	for i := 1; i <= rowCount; i++ {
		text := fmt.Sprintf("large-%02d-", i) + strings.Repeat("abcdefgh", valueSize/8)
		_, err := conn.Exec(fmt.Sprintf("INSERT INTO %s (id, v_text) VALUES (%d, ?)", quote(table), i), text)
		require.NoError(t, err)
	}

	// 导出前留存长度+md5基准，回环导入后逐项比对（大值不得被截断或改写）
	type digest struct{ length, md5 string }
	baseline := make(map[int]digest, rowCount)
	itDigest := func(id int) digest {
		_, rows, err := conn.Query(fmt.Sprintf("SELECT LENGTH(v_text) AS len, MD5(v_text) AS md5 FROM %s WHERE id = %d", quote(table), id))
		require.NoError(t, err)
		return digest{fmt.Sprintf("%v", rows[0]["len"]), fmt.Sprintf("%v", rows[0]["md5"])}
	}
	for i := 1; i <= rowCount; i++ {
		baseline[i] = itDigest(i)
	}

	path := filepath.Join(t.TempDir(), table+".sql")
	probe, size := itDumpToFile(t, conn, []string{table}, "mysql", false, path)
	require.Greater(t, size, int64(70<<20), "15行×5MB产物应≈75MB")

	script := itReadFile(t, path)
	inserts := itInsertStmts(conn.GetDialect().GetSQLSplitter(), itSplitStmts(t, conn, script))
	// 行数预算100远达不到，全靠字节预算提前flush：ceil(15/2)=8批
	assert.Equal(t, rowCount/2+1, len(inserts), "字节预算下真实产物应为每批≤2行")
	assert.Less(t, probe.max, 24<<20, "最大写入块应<24MB（转义膨胀后仍需低于mysql单包上限）")
	t.Logf("产物 %.1fMB, 写入块 %d 次(最大 %.1fMB), INSERT %d 条",
		float64(size)/(1<<20), probe.chunks, float64(probe.max)/(1<<20), len(inserts))

	// 清空后由生产导入链路恢复
	_, err = conn.Exec(fmt.Sprintf("TRUNCATE TABLE %s", quote(table)))
	require.NoError(t, err)
	require.NoError(t, (&transfer.DbTransferAppImpl{}).ImportDumpStream(context.Background(), 0, conn,
		bufio.NewReaderSize(strings.NewReader(script), 1<<20)), "大值dump回环导入失败")
	for i := 1; i <= rowCount; i++ {
		assert.Equal(t, baseline[i], itDigest(i), "id=%d 大值恢复后长度/md5不一致", i)
	}

	_, _ = conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table)))
}

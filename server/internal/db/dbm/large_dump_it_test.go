//go:build it

package dbm

// 大数据量导出/导入链路集成测试：
//   - 验证DumpDb的行数+字节双预算分批策略（dumpTableScriptBatched与DumpDb实现一致）：
//     大批量行场景内存峰值与数据总量解耦（流式），单行大值场景字节预算提前flush避免单条语句超长
//   - 运行方式：cd server && go test -tags it -count=1 -v ./internal/db/dbm/

import (
	"fmt"
	"io"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

// heapAlloc GC后返回当前堆内存
func heapAlloc() uint64 {
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapAlloc
}

// heapDelta 计算堆内存增量（MB），GC后堆可能小于采样前，下溢按0处理
func heapDelta(before, after uint64) int64 {
	d := int64(after) - int64(before)
	if d < 0 {
		return 0
	}
	return d
}

type dumpStats struct {
	rowCount     int   // 导出总行数
	totalBytes   int64 // 导出产物总字节
	flushCount   int   // 分批flush次数
	maxBatchRows int   // 单批最大行数
	maxStmtBytes int   // 单条语句最大字节
}

// dumpTableScriptBatched 与DumpDb一致的流式分批导出：WalkTableRows游标遍历，
// 行数+字节双预算分批生成INSERT写入writer（DDL部分与dumpTableScript一致，这里仅数据）
func dumpTableScriptBatched(t *testing.T, conn *dbi.DbConn, table string, writer io.Writer) (*dumpStats, error) {
	t.Helper()
	srcMeta := conn.GetMetadata()
	srcDialect := conn.GetDialect()
	gen := srcDialect.GetSQLGenerator()

	srcCols, err := srcMeta.GetColumns(table)
	require.NoError(t, err)
	cols := make([]dbi.Column, 0, len(srcCols))
	for _, col := range srcCols {
		if col.TableName != table {
			continue
		}
		require.NoError(t, dbi.ConvToTargetDbColumn(conn.Info.Type, conn.Info.Type, srcDialect, &col))
		cols = append(cols, col)
	}
	require.NotEmpty(t, cols)

	stats := &dumpStats{}
	dataCount := 0
	rows := make([][]any, 0)
	pendingBytes := 0

	flushRows := func() error {
		if len(rows) == 0 {
			return nil
		}
		insertSql := gen.GenInsert(table, cols, rows, dbi.DuplicateStrategyNone, nil)
		stmt := strings.Join(insertSql, ";\n") + ";\n"
		if len(stmt) > stats.maxStmtBytes {
			stats.maxStmtBytes = len(stmt)
		}
		if _, err := io.WriteString(writer, stmt); err != nil {
			return err
		}
		stats.totalBytes += int64(len(stmt))
		stats.flushCount++
		if len(rows) > stats.maxBatchRows {
			stats.maxBatchRows = len(rows)
		}
		rows = make([][]any, 0)
		pendingBytes = 0
		return nil
	}

	_, err = conn.WalkTableRows(itCtx(), srcDialect.Quoter().Quote(table), func(row map[string]any, _ []*dbi.QueryColumn) error {
		rowValues := make([]any, len(cols))
		rowBytes := 0
		for i, col := range cols {
			rowValues[i] = row[col.ColumnName]
			switch v := rowValues[i].(type) {
			case string:
				rowBytes += len(v)
			case []byte:
				rowBytes += len(v)
			default:
				rowBytes += 16
			}
		}
		rows = append(rows, rowValues)
		dataCount++
		pendingBytes += rowBytes
		if dataCount%dbi.DumpInsertBatchRows != 0 && pendingBytes < dbi.DumpInsertBatchBytes {
			return nil
		}
		if err := flushRows(); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return stats, err
	}
	stats.rowCount = dataCount
	return stats, flushRows()
}

// TestITMysqlLargeRowsDumpImport 大批量行导出导入：2万行×~1KB，
// 验证流式导出内存峰值与总量解耦 + 导入sqlite后行数与内容一致
func TestITMysqlLargeRowsDumpImport(t *testing.T) {
	conn := mysqlConn(t)
	defer conn.Close()

	table := "it_large_rows"
	mustExec(t, conn, "DROP TABLE IF EXISTS `"+table+"`")
	mustExec(t, conn, "CREATE TABLE `"+table+"` (id INT AUTO_INCREMENT PRIMARY KEY, v_text VARCHAR(2048), v_dt DATETIME)")

	const rowCount = 20000
	// 500行/批插入，减少往返
	for batch := 0; batch < rowCount/500; batch++ {
		var sb strings.Builder
		sb.WriteString("INSERT INTO `" + table + "` (v_text, v_dt) VALUES ")
		for i := 0; i < 500; i++ {
			idx := batch*500 + i + 1
			if i > 0 {
				sb.WriteString(",")
			}
			// ~1KB文本，含中文与特殊字符
			text := fmt.Sprintf("row-%06d-中文数据'quote\"dquote\\slash %s", idx, strings.Repeat("x", 900))
			sb.WriteString(fmt.Sprintf("('%s', '2026-09-05 12:00:00')", strings.ReplaceAll(text, "'", "''")))
		}
		mustExec(t, conn, sb.String())
	}

	// 导出并观测内存峰值：流式实现下内存增量不应随数据总量（~20MB）线性放大数倍
	before := heapAlloc()
	stats, err := dumpTableScriptBatched(t, conn, table, io.Discard)
	require.NoError(t, err)
	delta := heapDelta(before, heapAlloc())

	assert.Equal(t, rowCount, stats.rowCount)
	assert.Equal(t, rowCount/100, stats.flushCount, "100行/批")
	assert.Equal(t, 100, stats.maxBatchRows)
	assert.Less(t, stats.maxStmtBytes, 4<<20, "单条INSERT应远小于字节预算")
	assert.Less(t, delta, int64(128<<20), "导出内存峰值(增量%dMB)不应暴涨", delta>>20)
	t.Logf("dump %d rows (~%.1fMB data): product %.1fMB, mem delta %dMB, max stmt %.1fMB",
		rowCount, float64(rowCount*930)/(1<<20), float64(stats.totalBytes)/(1<<20), delta>>20, float64(stats.maxStmtBytes)/(1<<20))

	// 导入sqlite验证行数与内容
	sqConn := sqliteConn(t)
	defer sqConn.Close()
	mustExec(t, sqConn, "CREATE TABLE `"+table+"` (id INTEGER PRIMARY KEY AUTOINCREMENT, v_text TEXT, v_dt DATETIME)")
	stmtCount := importScript(t, sqConn, strings.NewReader(func() string {
		sb := &strings.Builder{}
		_, _ = dumpTableScriptBatched(t, conn, table, sb)
		return sb.String()
	}()))
	assert.Equal(t, rowCount/100, stmtCount, "200条批量INSERT")

	_, row, err := sqConn.Query("SELECT COUNT(*) AS cnt FROM `" + table + "`")
	require.NoError(t, err)
	require.NotEmpty(t, row)
	// sqlite驱动可能返回string类型的数字（如"20000"），统一转int64比较
	cntNum, parseErr := strconv.ParseInt(fmt.Sprint(normalizeDbValue(row[0]["cnt"])), 10, 64)
	require.NoError(t, parseErr)
	assert.Equal(t, int64(rowCount), cntNum)

	// 抽样校验内容一致
	_, srcRow, err := conn.Query(fmt.Sprintf("SELECT v_text FROM `%s` WHERE id = 12345", table))
	require.NoError(t, err)
	_, dstRow, err := sqConn.Query(fmt.Sprintf("SELECT v_text FROM `%s` WHERE id = 12345", table))
	require.NoError(t, err)
	assert.Equal(t, srcRow[0]["v_text"], dstRow[0]["v_text"])
}

// TestITMysqlLargeValueBatchBudget 单行大值（15行×5MB）触发字节预算：
// 每批最多2行（2×5MB≥8MB预算），单条语句不超长（旧固定100行策略会生成~75MB单条语句，
// 内存峰值暴涨且导入超mysql max_allowed_packet默认64MB直接失败）
func TestITMysqlLargeValueBatchBudget(t *testing.T) {
	conn := mysqlConn(t)
	defer conn.Close()

	table := "it_large_value"
	mustExec(t, conn, "DROP TABLE IF EXISTS `"+table+"`")
	mustExec(t, conn, "CREATE TABLE `"+table+"` (id INT PRIMARY KEY, v_text LONGTEXT)")

	const rowCount = 15
	const valueSize = 5 << 20 // 5MB/行
	// 逐行插入大值（文本长度恰为valueSize字节）
	for i := 1; i <= rowCount; i++ {
		text := fmt.Sprintf("large-%02d-", i) + strings.Repeat("abcdefgh", valueSize/8)
		mustExec(t, conn, fmt.Sprintf("INSERT INTO `%s` (id, v_text) VALUES (%d, '%s')", table, i, text))
	}

	before := heapAlloc()
	product := &strings.Builder{}
	stats, err := dumpTableScriptBatched(t, conn, table, product)
	require.NoError(t, err)
	delta := heapDelta(before, heapAlloc())

	assert.Equal(t, rowCount, stats.rowCount)
	assert.LessOrEqual(t, stats.maxBatchRows, 2, "字节预算下单批最多2行")
	assert.Equal(t, rowCount/2+1, stats.flushCount)
	assert.Less(t, stats.maxStmtBytes, 24<<20, "单条INSERT(hex/转义膨胀后)仍应远低于64MB单包上限")
	assert.Less(t, delta, int64(192<<20), "导出内存峰值(增量%dMB)不应暴涨，旧策略会超200MB", delta>>20)
	t.Logf("dump %d rows x 5MB: product %.1fMB, mem delta %dMB, max stmt %.1fMB, flushes %d",
		rowCount, float64(stats.totalBytes)/(1<<20), delta>>20, float64(stats.maxStmtBytes)/(1<<20), stats.flushCount)

	// 导入回mysql并校验字节长度与md5一致
	importTable := table + "_import"
	mustExec(t, conn, "DROP TABLE IF EXISTS `"+importTable+"`")
	mustExec(t, conn, "CREATE TABLE `"+importTable+"` LIKE `"+table+"`")
	script := strings.ReplaceAll(product.String(), table, importTable)
	stmtCount := importScript(t, conn, strings.NewReader(script))
	assert.Equal(t, stats.flushCount, stmtCount)

	for _, id := range []int{1, 8, 15} {
		_, srcRow, err := conn.Query(fmt.Sprintf("SELECT LENGTH(v_text) AS len, MD5(v_text) AS md5 FROM `%s` WHERE id = %d", table, id))
		require.NoError(t, err)
		_, dstRow, err := conn.Query(fmt.Sprintf("SELECT LENGTH(v_text) AS len, MD5(v_text) AS md5 FROM `%s` WHERE id = %d", importTable, id))
		require.NoError(t, err)
		assert.Equal(t, srcRow[0]["len"], dstRow[0]["len"], "id=%d 字节长度一致", id)
		assert.Equal(t, srcRow[0]["md5"], dstRow[0]["md5"], "id=%d md5一致", id)
	}

	mustExec(t, conn, "DROP TABLE IF EXISTS `"+importTable+"`")
}

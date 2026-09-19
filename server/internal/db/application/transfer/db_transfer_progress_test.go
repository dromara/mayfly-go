package transfer

// 多表 Progress 回调行数统计逻辑的单元测试。
// 验证 DumpDbScript 每表 dataCount 从 0 重新计数时，
// 上层 errGroup 并发 goroutine 中的 lastStmtCount 差值法能正确跨表累加总行数。
//
// 运行：cd server && go test -count=1 -run TestMultiTableRowCounting ./internal/db/application/transfer/

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/stretchr/testify/assert"
)

// simulateTableDump 模拟 DumpDbScript 对单表的 Progress 回调行为：
// dataCount 从 0 开始，每行递增，按批次和表结束各回调一次
func simulateTableDump(progress func(string, dbi.StmtType, int, bool), tableName string, rowCount int, batchSize int) {
	dataCount := 0
	for i := 1; i <= rowCount; i++ {
		dataCount++
		if dataCount%batchSize == 0 {
			progress(tableName, dbi.StmtTypeInsert, dataCount, false)
		}
	}
	// 表结束时的最终回调
	progress(tableName, dbi.StmtTypeInsert, dataCount, true)
}

// TestMultiTableRowCounting 验证多表并发迁移场景下的行数统计正确性。
// 模拟 transfer2Db 中 errGroup goroutine 的 lastStmtCount 差值法
func TestMultiTableRowCounting(t *testing.T) {
	// 模拟72个表，每表不同行数
	type tableSpec struct {
		name     string
		rowCount int
	}
	tables := make([]tableSpec, 72)
	expectedTotal := int64(0)
	for i := range tables {
		tables[i] = tableSpec{
			name:     fmt.Sprintf("table_%03d", i+1),
			rowCount: (i%10 + 1) * 7, // 7, 14, 21, ..., 70 循环
		}
		expectedTotal += int64(tables[i].rowCount)
	}

	var totalRows int64
	var wg sync.WaitGroup
	// 模拟并发度为4
	sem := make(chan struct{}, 4)

	for _, tbl := range tables {
		wg.Add(1)
		sem <- struct{}{}
		go func(tableName string, rowCount int) {
			defer wg.Done()
			defer func() { <-sem }()

			// 这正是 transfer2Db 中每个 goroutine 的逻辑
			lastStmtCount := 0
			progress := func(currentTable string, stmtType dbi.StmtType, stmtCount int, currentStmtTypeEnd bool) {
				if stmtType == dbi.StmtTypeInsert {
					if stmtCount < lastStmtCount {
						lastStmtCount = 0
					}
					if stmtCount > lastStmtCount {
						atomic.AddInt64(&totalRows, int64(stmtCount-lastStmtCount))
						lastStmtCount = stmtCount
					}
				}
			}

			// 模拟 DumpDbScript 的回调行为（batch=100）
			simulateTableDump(progress, tableName, rowCount, 100)
		}(tbl.name, tbl.rowCount)
	}
	wg.Wait()

	assert.Equal(t, expectedTotal, atomic.LoadInt64(&totalRows),
		"72表并发迁移的总行数应精确匹配预期")
}

// TestSingleTableRowCounting 验证单表场景（基准对照）
func TestSingleTableRowCounting(t *testing.T) {
	var totalRows int64
	lastStmtCount := 0

	progress := func(currentTable string, stmtType dbi.StmtType, stmtCount int, currentStmtTypeEnd bool) {
		if stmtType == dbi.StmtTypeInsert {
			if stmtCount < lastStmtCount {
				lastStmtCount = 0
			}
			if stmtCount > lastStmtCount {
				atomic.AddInt64(&totalRows, int64(stmtCount-lastStmtCount))
				lastStmtCount = stmtCount
			}
		}
	}

	// 3行，batch=100（不触发批次回调，仅表结束回调）
	simulateTableDump(progress, "single_table", 3, 100)
	assert.Equal(t, int64(3), atomic.LoadInt64(&totalRows))
}

// TestMultiTableWithBatching 验证大批量场景（触发批次中间回调）
func TestMultiTableWithBatching(t *testing.T) {
	type tableSpec struct {
		name     string
		rowCount int
	}
	// 5个表，每表250行（batch=100时会触发2次中间回调+1次结束回调）
	tables := []tableSpec{
		{"t1", 250},
		{"t2", 100},
		{"t3", 50},
		{"t4", 300},
		{"t5", 1},
	}
	expectedTotal := int64(0)
	for _, tbl := range tables {
		expectedTotal += int64(tbl.rowCount)
	}

	var totalRows int64
	var wg sync.WaitGroup

	for _, tbl := range tables {
		wg.Add(1)
		go func(tableName string, rowCount int) {
			defer wg.Done()
			lastStmtCount := 0
			progress := func(currentTable string, stmtType dbi.StmtType, stmtCount int, currentStmtTypeEnd bool) {
				if stmtType == dbi.StmtTypeInsert {
					if stmtCount < lastStmtCount {
						lastStmtCount = 0
					}
					if stmtCount > lastStmtCount {
						atomic.AddInt64(&totalRows, int64(stmtCount-lastStmtCount))
						lastStmtCount = stmtCount
					}
				}
			}
			simulateTableDump(progress, tableName, rowCount, 100)
		}(tbl.name, tbl.rowCount)
	}
	wg.Wait()

	assert.Equal(t, expectedTotal, atomic.LoadInt64(&totalRows),
		"5表含批次回调的并发迁移总行数应精确匹配")
}

// TestEmptyTablesRowCounting 验证空表（0行）不影响总计数
func TestEmptyTablesRowCounting(t *testing.T) {
	var totalRows int64
	lastStmtCount := 0

	// 模拟3个空表 + 1个有数据的表
	for _, table := range []struct {
		name     string
		rowCount int
	}{
		{"empty1", 0},
		{"empty2", 0},
		{"empty3", 0},
		{"has_data", 42},
	} {
		progress := func(currentTable string, stmtType dbi.StmtType, stmtCount int, currentStmtTypeEnd bool) {
			if stmtType == dbi.StmtTypeInsert {
				if stmtCount < lastStmtCount {
					lastStmtCount = 0
				}
				if stmtCount > lastStmtCount {
					atomic.AddInt64(&totalRows, int64(stmtCount-lastStmtCount))
					lastStmtCount = stmtCount
				}
			}
		}
		simulateTableDump(progress, table.name, table.rowCount, 100)
	}

	assert.Equal(t, int64(42), atomic.LoadInt64(&totalRows),
		"空表不应影响总行数统计")
}

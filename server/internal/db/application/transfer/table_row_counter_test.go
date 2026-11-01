package transfer

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTableRowCounterPerTable 按表计数必须独立且幂等。
//
// 旧实现用共享的 lastStmtCount 跨表做差值：表 A 报到 20 后，表 B 从 5 开始会被判为
// "回退"而重置，随后的增量累加全错 —— 迁移行数漏计/重计。
func TestTableRowCounterPerTable(t *testing.T) {
	c := newTableRowCounter()
	require.EqualValues(t, 10, c.observe("a", 10))
	require.EqualValues(t, 20, c.observe("a", 20))
	// 表 B 独立计数（旧差值法在这里会漏 5 行）
	require.EqualValues(t, 25, c.observe("b", 5))
	// 幂等：同表重复上报同一值不重复累加
	require.EqualValues(t, 25, c.observe("b", 5))
	// 乱序上报较小的旧值不把总数拉低
	require.EqualValues(t, 25, c.observe("b", 3))
	require.EqualValues(t, 25, c.Total())
}

// TestTableRowCounterConcurrent 并发上报不得丢计数（库→文件迁移是多表并发）。
func TestTableRowCounterConcurrent(t *testing.T) {
	c := newTableRowCounter()
	var wg sync.WaitGroup
	for i := 1; i <= 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c.observe("t"+string(rune('a'+i%5)), i)
		}(i)
	}
	wg.Wait()
	// 5 张表各取历史最大值：t_a..t_e 分别对应 i%5 相同的一组，最大值是 100/99/98/97/96 中的最大值
	require.EqualValues(t, 100+99+98+97+96, c.Total())
}

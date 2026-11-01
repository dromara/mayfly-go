package transfer

import "sync"

// tableRowCounter 按表记录最新累计行数并维护全局合计（并发安全、幂等）。
//
// 为什么不能用「末批 stmtCount 差值」跨表累加：那依赖 dump 侧"每张表从 0 重新计数"的
// 隐含约定。库→文件迁移是多表并发（errGroup），共享的 lastStmtCount 会被其他表推进，
// 甚至因 stmtCount 变小而回退重置 —— 日志里的"迁移行数"会漏计或重计，而这个数字正是
// 用户判断迁移完整性的主要依据。
type tableRowCounter struct {
	mu    sync.Mutex
	rows  map[string]int64
	total int64
}

func newTableRowCounter() *tableRowCounter {
	return &tableRowCounter{rows: make(map[string]int64)}
}

// observe 记录某表的最新累计行数，返回全局合计。
//
// 幂等且单调：同一张表重复上报同一值不重复累加；乱序上报较小的旧值不会把总数拉低
// （只取该表历史最大值）。count <= 0 视为无信息，不改变计数。
func (c *tableRowCounter) observe(table string, count int) int64 {
	if count <= 0 {
		return c.Total()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if v := int64(count); v > c.rows[table] {
		c.total += v - c.rows[table]
		c.rows[table] = v
	}
	return c.total
}

// Total 返回当前全局合计
func (c *tableRowCounter) Total() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.total
}

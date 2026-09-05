package dbi

// DumpDb 导出数据时的批量INSERT分批策略：行数与字节数双预算，达到任一预算即生成一批
const (
	// DumpInsertBatchRows 单条批量INSERT的最大行数
	DumpInsertBatchRows = 100

	// DumpInsertBatchBytes 单批累计字节预算（8MB）：避免单行大值（如大blob/text）
	// 导致单条语句超长——内存峰值暴涨，且导入时超mysql max_allowed_packet等单包上限直接失败
	DumpInsertBatchBytes = 8 << 20
)

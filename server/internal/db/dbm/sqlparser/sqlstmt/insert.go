package sqlstmt

// InsertStmt INSERT 语句
type InsertStmt struct {
	Base
	Table   TableRef
	Columns []string
	// ValueRows VALUES 的逐行字面量：仅当每一格都是可确定求值的纯字面量（字符串/数字/NULL/布尔）时才非 nil；
	// 出现函数、表达式、子查询、占位符等任何形态即整体为 nil——这些形态的最终写入值只有执行后才知道
	ValueRows []InsertValueRow
	// HasConflictClause 标记 INSERT 带冲突处理子句：MySQL 的 REPLACE / INSERT IGNORE /
	// ON DUPLICATE KEY UPDATE、PostgreSQL/SQLite 的 ON CONFLICT DO NOTHING|UPDATE。
	// 这类语句在目标行已存在时并不新增行（忽略/改为更新/先删后插），而 INSERT 的回滚是
	// 「删掉刚插入的行」——按 VALUES 字面量生成的 DELETE 会命中并删除那条既有行（数据丢失）。
	// 故采集层见此标志即 fail-closed 跳过插入快照，不提供回滚而非给出会删既有行的回滚
	HasConflictClause bool
}

func (*InsertStmt) StmtKind() Kind { return KindInsert }

// InsertValueRow 一行 VALUES 的逐列字面量
type InsertValueRow []InsertValue

// InsertValue VALUES 单元格字面量
type InsertValue struct {
	// Text 字面量原文（字符串含引号与转义、数字含符号小数指数、布尔为原词）：
	// 消费方按原文回放而不是「解码为值再编码」——与执行时同一份文本、同一方言语义，
	// 不引入 \n、''、E''、q'[]' 等形态的二次转义改写
	Text string
	// Null 是否为 NULL 关键字（条件渲染须写 IS NULL，= NULL 恒不成立）
	Null bool
}

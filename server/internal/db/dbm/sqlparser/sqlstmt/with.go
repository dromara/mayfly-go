package sqlstmt

// WithStmt WITH 语句
type WithStmt struct {
	Base
	// MainType WITH 主语句或数据修改型 CTE 解析出的实际语句类型（Select/Insert/Update/Delete）。
	// 由 base 解析器 ParseWithStmt 扫描回填，供 sqlparser.Classify 判定「WITH 开头但实为写」的语句
	// （如 `WITH c AS (SELECT..) DELETE FROM t` 或 `WITH c AS (DELETE..) SELECT`）。
	// 空表示未解析（手工构造等），Classify 兜底按查询，与历史一致。
	// StmtKind 仍返回 KindWith（结构类别），MainType 只表达主语句语义，两者正交。
	MainType StmtType
}

func (*WithStmt) StmtKind() Kind { return KindWith }

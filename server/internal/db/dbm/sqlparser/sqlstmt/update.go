package sqlstmt

// UpdateStmt UPDATE 语句
type UpdateStmt struct {
	Base
	Tables []TableRef   // 更新的表（支持多表）
	Set    []Assignment // SET 子句
	Where  *Expr
	// Limit 尾随行限制子句（MySQL/SQLite 的 UPDATE … LIMIT n）。非 nil 表示实际只影响 WHERE
	// 匹配集的一个非确定子集：旧值采集的回查 SELECT 按 WHERE 捞回全部匹配行，与该子集不一致，
	// 消费方（采集层）据此 fail-closed 跳过采集，而不是生成覆盖未变更行的错误回滚
	Limit *Limit
}

func (*UpdateStmt) StmtKind() Kind { return KindUpdate }

package sqlstmt

// DeleteStmt DELETE 语句
type DeleteStmt struct {
	Base
	Tables []TableRef // 删除的表
	Where  *Expr
	// Limit 尾随行限制子句（MySQL/SQLite 的 DELETE … LIMIT n）。非 nil 表示实际只删除 WHERE
	// 匹配集的一个非确定子集：旧值采集按 WHERE 捞回全部匹配行，回滚会为未被删除的行生成
	// INSERT（主键冲突或重复行），故消费方（采集层）据此 fail-closed 跳过采集
	Limit *Limit
}

func (*DeleteStmt) StmtKind() Kind { return KindDelete }

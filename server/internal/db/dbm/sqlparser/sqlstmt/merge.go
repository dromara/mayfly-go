package sqlstmt

// MergeStmt MERGE（upsert）语句：`MERGE INTO target USING source ON (cond) WHEN [NOT] MATCHED THEN ...`
//
// 它是可插入/更新/删除的写语句，但结构不建模（各方言 MERGE 子句差异大，且不纳入回滚旧值快照），
// 仅需语句边界与原文。分类为 StmtTypeMerge，使其走写执行通道并受 DML 审批策略约束，
// 而非被当作 OtherStmt（只读通道）绕过治理。
type MergeStmt struct {
	Base
}

func (*MergeStmt) StmtKind() Kind { return KindMerge }

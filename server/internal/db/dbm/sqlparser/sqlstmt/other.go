package sqlstmt

// OtherStmt 未被结构化建模的语句（SHOW/SET/事务控制/CALL/LOAD DATA…）。
//
// ReadOnly 是「这条语句是否确定不写业务数据」的一等输出，由词法层的首词白名单判定：
// 白名单之外一律 false（可能写）。下游据此选择只读通道或写通道，是 fail-closed 的落点——
// 之前只输出「other」，执行层只能按只读处理，导致 LOAD DATA / COPY FROM / CALL 绕过审批与审计。
type OtherStmt struct {
	Base
	ReadOnly bool
}

// NewOtherStmt 构造未建模语句节点；readOnly 必须由词法白名单判定后传入。
func NewOtherStmt(text string, readOnly bool) *OtherStmt {
	return &OtherStmt{Base: Base{Text: text}, ReadOnly: readOnly}
}

func (*OtherStmt) StmtKind() Kind { return KindOther }

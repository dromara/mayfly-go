package sqlstmt

// StmtType 语句类型
type StmtType string

const (
	StmtTypeSelect StmtType = "select"
	StmtTypeInsert StmtType = "insert"
	StmtTypeUpdate StmtType = "update"
	StmtTypeDelete StmtType = "delete"
	StmtTypeMerge  StmtType = "merge"
	StmtTypeDDL    StmtType = "ddl"
	// OTHER 拆两类：只读（导航/事务/维护）与可能写（CALL/LOAD DATA/COPY FROM/未知）。
	// 拆分的意义在安全侧：写语句不能再落进只读通道。
	StmtTypeOtherRead  StmtType = "other-read"
	StmtTypeOtherWrite StmtType = "other-write"
)

// ClassifyByParse 供各方言 StatementClassifier 复用的通用实现：解析 → 按 AST Kind 判定语句类型。
// 把「解析出错回传 / nil AST（纯可执行注释 /*!…*/、空串等无可分类语句）返回空类型交调用方兜底」
// 这一 nil 守卫收敛到单点——方言侧只需 `return sqlstmt.ClassifyByParse(p.Parse, sql)`，
// 不会遗漏 nil 判定而对 nil AST 解引用 panic。
func ClassifyByParse(parse func(string) (Stmt, error), sql string) (StmtType, error) {
	stmt, err := parse(sql)
	if err != nil {
		return "", err
	}
	if stmt == nil {
		return "", nil
	}
	return DefaultClassifyStmt(stmt.StmtKind()), nil
}

// DefaultClassifyStmt 基于 AST 的默认语句分类：把 Kind 映射为 StmtType。
//
// 这是唯一实现（原先 stmt_type.go 与 pagination.go 各有一份，已收敛到此）。
// Kind 层无法区分 OTHER 的读写，兜底按可能写处理（fail-closed）——需要精确区分时
// 走 sqlparser.Classify 的 AST 分支（*OtherStmt.ReadOnly）。
func DefaultClassifyStmt(kind Kind) StmtType {
	switch kind {
	case KindSelect:
		return StmtTypeSelect
	case KindInsert:
		return StmtTypeInsert
	case KindUpdate:
		return StmtTypeUpdate
	case KindDelete:
		return StmtTypeDelete
	case KindMerge:
		return StmtTypeMerge
	case KindDdl:
		return StmtTypeDDL
	case KindShow:
		return StmtTypeOtherRead
	default:
		return StmtTypeOtherWrite
	}
}

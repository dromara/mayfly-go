package sqlstmt

// StmtType 语句类型
type StmtType string

const (
	StmtTypeSelect StmtType = "select"
	StmtTypeInsert StmtType = "insert"
	StmtTypeUpdate StmtType = "update"
	StmtTypeDelete StmtType = "delete"
	StmtTypeDDL    StmtType = "ddl"
	StmtTypeOther  StmtType = "other"
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

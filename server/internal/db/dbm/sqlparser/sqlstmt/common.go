package sqlstmt

// TableRef 表示表引用
type TableRef struct {
	Schema string // 数据库/模式名
	Name   string // 表名
	Alias  string // 别名
}

// FullName 返回完整表名（含 schema）
func (t TableRef) FullName() string {
	if t.Schema != "" {
		return t.Schema + "." + t.Name
	}
	return t.Name
}

// Expr 表示表达式（WHERE、ON、HAVING 等）
type Expr struct {
	Text string   // 原始表达式文本
	Root ExprNode // 结构化AST（Pratt解析成功时非nil）；nil表示降级为纯文本
}

// Limit LIMIT/OFFSET
type Limit struct {
	Count  int
	Offset int
	Text   string
}

// OrderByItem ORDER BY 项
type OrderByItem struct {
	Text string
	Desc bool
}

// Assignment SET 赋值
type Assignment struct {
	Column string
	Value  *Expr
	Text   string
}

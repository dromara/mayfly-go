package sqlstmt

// SelectItemKind SELECT 元素类型
type SelectItemKind int

const (
	SelectItemStar SelectItemKind = iota
	SelectItemColumn
	SelectItemExpr
	SelectItemFunction
)

// SelectItem 表示 SELECT 列表中的一个元素
type SelectItem struct {
	Kind       SelectItemKind
	Text       string // 原始文本
	Alias      string
	TableAlias string // 对于 t.* 或 t.col，这是表别名
	ColumnName string // 对于列引用
}

// IsStar 判断是否为 *
func (s SelectItem) IsStar() bool {
	return s.Kind == SelectItemStar
}

// JoinKind JOIN 类型
type JoinKind int

const (
	JoinKindInner JoinKind = iota
	JoinKindLeft
	JoinKindRight
	JoinKindFull
	JoinKindCross
	JoinKindNatural
)

// JoinClause JOIN 子句
type JoinClause struct {
	Kind  JoinKind
	Table TableRef
	On    *Expr
	Text  string
}

// UnionClause 集合操作子句（UNION/INTERSECT/EXCEPT/MINUS）
type UnionClause struct {
	Select *SelectStmt
	Op     string // 集合操作符：UNION/INTERSECT/EXCEPT/MINUS
	All    bool
	Text   string
}

// SelectStmt SELECT 语句（含 UNION、子查询等）
type SelectStmt struct {
	Base
	Distinct bool
	Items    []SelectItem
	From     []TableRef
	Joins    []JoinClause
	Where    *Expr
	GroupBy  []string
	Having   *Expr
	OrderBy  []OrderByItem
	Limit    *Limit
	Unions   []UnionClause
	// Into 标记 SELECT ... INTO（mssql/pg 建表、mysql INTO OUTFILE/DUMPFILE、变量赋值）：
	// 它写出数据或文件，不是纯读。解析器消费 INTO 目标但不建模，仅置此标志供 Classify 判为写。
	Into bool
}

func (*SelectStmt) StmtKind() Kind { return KindSelect }

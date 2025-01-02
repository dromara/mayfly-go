package sqlstmt

type (
	IExpr interface {
		INode

		isExpr()
	}

	Expr struct {
		*Node
	}

	ExprLogical struct {
		Expr

		Operator string
		Exprs    []IExpr
	}

	ExprPredicate struct {
		Expr

		Predicate IPredicate
	}
)

func (*Expr) isExpr() {}

type (
	IPredicate interface {
		INode

		isPredicate()
	}

	Predicate struct {
		*Node
	}

	PredicateBinaryComparison struct {
		Predicate

		Left               IPredicate
		Right              IPredicate
		ComparisonOperator string
	}

	PredicateIn struct {
		Predicate

		InPredicate IPredicate
		Exprs       []IExpr
		SelectStmt  ISelectStmt
	}

	PredicateExprAtom struct {
		Predicate

		ExprAtom IExprAtom
	}

	PredicateLike struct {
		Predicate

		InPredicate IPredicate
		Exprs       []IExpr
		SelectStmt  ISelectStmt
	}
)

func (*Predicate) isPredicate() {}

type (
	IExprAtom interface {
		INode

		isExprAtom()
	}

	ExprAtom struct {
		*Node
	}

	ExprAtomFunctionCall struct {
		*Node
	}

	ExprAtomConstant struct {
		ExprAtom

		Constant *Constant
	}

	ExprAtomColumnName struct {
		ExprAtom

		ColumnName *ColumnName
	}
)

func (*ExprAtom) isExprAtom() {}

type (
	ITableSource interface {
		INode

		isTableSource()
	}

	TableSource struct {
		*Node
	}

	TableSources struct {
		*Node

		TableSources []ITableSource
	}

	TableSourceBase struct {
		TableSource

		TableSourceItem ITableSourceItem
		JoinParts       []IJoinPart
	}

	ITableSourceItem interface {
		INode
	}

	TableSourceItem struct {
		*Node
	}

	AtomTableItem struct {
		TableSourceItem

		TableName *TableName // 表名
		Alias     string     // 别名
	}
)

func (*TableSource) isTableSource() {}

type (
	Constant struct {
		*Node

		Value string
	}

	FullId struct {
		*Node

		Uids []string
	}
)

type ColumnName struct {
	*Node

	Owner             string
	Identifier        *IdentifierValue
	NestedObjectAttrs []string
}

type TableName struct {
	*Node

	Owner      string
	Identifier *IdentifierValue
}

type (
	FuncCall interface {
		INode
	}
)

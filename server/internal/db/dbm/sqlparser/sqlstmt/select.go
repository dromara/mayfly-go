package sqlstmt

type (
	ISelectStmt interface {
		INode

		isSelect()
	}

	SelectStmt struct {
		*Node
	}

	QuerySpecification struct {
		*Node

		SelectElements *SelectElements
		From           *TableSources
		Where          IExpr
		Limit          *Limit
	}

	SimpleSelectStmt struct {
		SelectStmt

		QuerySpecification *QuerySpecification
	}

	UnionSelectStmt struct {
		SelectStmt

		UnionType          string
		QuerySpecification *QuerySpecification
		QueryExpr          *QueryExpr
		UnionStmts         []*UnionStmt
		Limit              *Limit
	}

	// 圆括号查询
	ParenthesisSelect struct {
		SelectStmt

		QueryExpr *QueryExpr
	}

	QueryExpr struct {
		*Node

		QuerySpecification *QuerySpecification
		QueryExpr          *QueryExpr
	}
)

func (*SelectStmt) isSelect() {}

// var _ (ISelectStmt) = (*SimpleSelectStmt)(nil)
// var _ (ISelectStmt) = (*SelectStmt)(nil)
// var _ (ISelectStmt) = (*ParenthesisSelect)(nil)

type (
	SelectElements struct {
		*Node

		Star     string // 查询所有
		Elements []ISelectElement
	}

	ISelectElement interface {
		INode
	}

	SelectStarElement struct {
		*Node

		FullId string
	}

	SelectColumnElement struct {
		*Node

		FullColumnName *ColumnName
		Alias          string
	}

	SelectFunctionElement struct {
		*Node

		Alias string
	}
)

type From struct {
	TableSource *ITableSource
}

type Limit struct {
	*Node

	RowCount int
	Offset   int
}

type (
	IJoinPart interface {
		INode
	}

	JoinPart struct {
		*Node

		TableSourceItem ITableSourceItem
	}

	InnerJoin struct {
		JoinPart
	}

	OuterJoin struct {
		JoinPart
	}

	NaturalJoin struct {
		JoinPart
	}
)

type (
	UnionStmt struct {
		*Node

		UnionType          string
		QuerySpecification *QuerySpecification
		QueryExpr          *QueryExpr
	}
)

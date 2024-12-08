package sqlstmt

type (
	IDeleteStmt interface {
		isDelete()
	}

	DeleteStmt struct {
		*Node

		TableSources *TableSources
		Where        IExpr
	}
)

func (*DeleteStmt) isDelete() {}

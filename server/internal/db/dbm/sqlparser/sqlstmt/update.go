package sqlstmt

type (
	IUpdateStmt interface {
		INode

		isUpdate()
	}

	UpdateStmt struct {
		*Node

		TableSources    *TableSources
		UpdatedElements []*UpdatedElement
		Where           IExpr
	}
)

func (*UpdateStmt) isUpdate() {}

type UpdatedElement struct {
	*Node

	ColumnName *ColumnName
	Value      IExpr
}

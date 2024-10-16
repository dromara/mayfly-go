package sqlstmt

type (
	IInsertStmt interface {
		isInsert()
	}

	InsertStmt struct {
		*Node

		TableName *TableName
	}
)

func (*InsertStmt) isInsert() {}

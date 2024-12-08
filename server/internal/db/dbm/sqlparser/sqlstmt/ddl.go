package sqlstmt

type (
	IDdlStmt interface {
		isDdl()
	}

	DdlStmt struct {
		*Node
	}

	CreateDatabase struct {
		DdlStmt
	}

	CreateTable struct {
		DdlStmt
	}

	CreateIndex struct {
		DdlStmt
	}

	AlterTable struct {
		DdlStmt
	}

	AlterDatabase struct {
		DdlStmt
	}

	DropDatabase struct {
		DdlStmt
	}

	DropIndex struct {
		DdlStmt
	}

	DropTable struct {
		DdlStmt
	}

	DropView struct {
		DdlStmt
	}
)

func (d *DdlStmt) isDdl() {}

func IsDDL(node INode) bool {
	return true
}

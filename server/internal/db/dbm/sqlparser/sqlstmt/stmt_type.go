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

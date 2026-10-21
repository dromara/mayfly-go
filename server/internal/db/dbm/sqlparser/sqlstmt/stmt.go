package sqlstmt

// Kind 表示 SQL 语句种类
type Kind string

const (
	KindSelect Kind = "SELECT"
	KindInsert Kind = "INSERT"
	KindUpdate Kind = "UPDATE"
	KindDelete Kind = "DELETE"
	KindMerge  Kind = "MERGE"
	KindDdl    Kind = "DDL"
	KindWith   Kind = "WITH"
	KindOther  Kind = "OTHER"
)

// Stmt 是所有 SQL 语句的接口
type Stmt interface {
	GetText() string
	StmtKind() Kind
}

// Base 提供基础实现
type Base struct {
	Text string // 原始 SQL 文本
}

func (b *Base) GetText() string {
	if b == nil {
		return ""
	}
	return b.Text
}

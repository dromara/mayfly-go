package sqlstmt

// DdlStmt DDL 语句。
//
// Target 是语句作用的目标**表**名（CREATE/ALTER/DROP TABLE、CREATE INDEX ... ON t、
// DROP INDEX ... ON t、CREATE/ALTER/DROP VIEW/TRIGGER 的对象名）；取不到时为空串。
// 表级访问控制与「指定表需审批」策略据此把 DDL 也纳入覆盖——此前 DDL 只有类型、没有表，
// 表级规则对改结构语句完全失效。
type DdlStmt struct {
	Base
	DdlKind string // CREATE_TABLE, DROP_TABLE, ALTER_TABLE 等
	Target  string // 目标表名（可能带 schema 限定，不含引号）；未知为空
	// Columns CREATE TABLE 的列定义（名/类型原文/主键/显式可空/DEFAULT 原文）。
	// 非 CREATE TABLE 或用 `AS SELECT`/`LIKE` 建表时为空（不猜）。
	Columns []ColumnDef
	// Renames 改名对：`RENAME TABLE a TO b[, c TO d]` 与 `ALTER TABLE t RENAME TO t2`。
	// 只收表级改名（`RENAME COLUMN a TO b` 的列改名不收，否则策略会在列名上误命中表规则）。
	Renames []Rename
}

// Rename 一次表改名（From 旧名、To 新名，均不含引号）。
type Rename struct {
	From string
	To   string
}

func (*DdlStmt) StmtKind() Kind { return KindDdl }

// ColumnDef CREATE TABLE 的列定义（只建被消费的字段，约束细节仍留在原文）。
type ColumnDef struct {
	Name     string
	Type     string // 类型原文，如 varchar(20)、decimal(10,2)、int unsigned
	Primary  bool   // 行内 PRIMARY KEY 或表级 PRIMARY KEY(...) 命中
	Unique   bool   // 行内 UNIQUE
	Nullable *bool  // nil=未声明；true/false=显式 NULL / NOT NULL
	Default  string // DEFAULT 之后的原文（保留引号，便于回放）
}

package dbi

// DumpKind 导出/迁移进度回调中，标识「当前正在处理的语句类别」的粗粒度分类：结构语句(DDL) 或 数据行(INSERT)。
//
// 仅服务于进度上报（如「表 X 的 DDL 已 dump N 条 / 数据已 dump N 行」），
// 与 sqlparser 用于**执行路由**的语句类型（sqlstmt.StmtType：select/update/delete/…）是两回事，勿混用。
type DumpKind string

const (
	DumpKindDDL    DumpKind = "ddl"    // 建表/建索引等结构语句
	DumpKindInsert DumpKind = "insert" // 数据行插入语句
)

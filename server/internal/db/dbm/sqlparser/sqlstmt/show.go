package sqlstmt

// ShowStmt SHOW 类语句（MySQL 系）。
//
// 历史实现把它伪装成 SelectStmt 以复用结果集渲染，副作用是类型失真：
// 分类输出 select、下游按「查询」对待，而 SHOW 本质是元信息导航语句（属 other-read）。
// 独立建模后，分类由 AST 直接给出 other-read，不再需要按方言特判。
type ShowStmt struct {
	Base
}

func (*ShowStmt) StmtKind() Kind { return KindShow }

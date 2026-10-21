package sqlstmt

// ExplainStmt EXPLAIN 语句（由 base 解析器按「语句头值 EXPLAIN」识别）。
//
// 语义分两种，必须区分对待：
//   - `EXPLAIN <stmt>`：只输出执行计划、不执行语句 → Inner 为空，分类为 Other（只读，走查询通道回计划）。
//   - `EXPLAIN ANALYZE <stmt>`（pg 亦支持 `EXPLAIN (ANALYZE, ...) <stmt>`）：会真正执行内层语句并回计划+耗时。
//     若内层是写语句（INSERT/UPDATE/DELETE/MERGE），整条就是写——Inner 记录该写类型，使 Classify 判为写、
//     走写通道并受 DML 审批约束，避免「EXPLAIN ANALYZE DELETE」被当只读放行而绕过审批（pg/mysql 均会真的删改）。
//
// StmtKind 仍为 KindOther（EXPLAIN 属辅助语句），Inner 只承载「ANALYZE 命中的内层写类型」，与 Kind 正交。
type ExplainStmt struct {
	Base
	Inner StmtType
}

// IsWrite 是否为会执行内层写的形态（EXPLAIN ANALYZE <写>）。
func (s *ExplainStmt) IsWrite() bool { return s.Inner != "" }

func (*ExplainStmt) StmtKind() Kind { return KindOther }

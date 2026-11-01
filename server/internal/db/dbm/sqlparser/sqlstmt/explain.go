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
	// Analyze 是否带 ANALYZE/ANALYSE：带则语句会**真正执行**内层语句。
	// 即便内层是 SELECT（Inner 为空），也可能调用有副作用的函数（lo_export/dblink_exec 等），
	// 故分类侧对 Analyze=true 一律按「可能写」处理（fail-closed）。
	Analyze bool
}

// IsWrite 是否为会执行内层**写**的形态（EXPLAIN ANALYZE <写>）。
func (s *ExplainStmt) IsWrite() bool { return s.Inner != "" }

// Executes 是否真的执行了内层语句（带 ANALYZE），此时即便内层是读也不能保证无副作用。
func (s *ExplainStmt) Executes() bool { return s.Analyze }

func (*ExplainStmt) StmtKind() Kind { return KindOther }

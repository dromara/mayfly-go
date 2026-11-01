package sqlparser

import (
	"strings"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// Classify 判定 SQL 语句类型，是语句分类的**单一入口**。
//
// 供执行路由、数据迁移、脱敏、导入等共同消费——分类是所有上层共用的基础语义，
// 必须单点实现、集中演进，而非各处用 strings.Contains 各自判定而漂移。
//
// 判定优先级：
//  1. 解析成功（有 AST）→ 按 AST 节点类型判定，最准确；
//  2. 无 AST / 解析失败 → 若方言具备 StatementClassifier 能力，用其 ClassifyStmt；
//  3. 仍无法判定 → 按方言语义取首个整词关键字（LeadingKeyword）映射（跳过前导空白与注释）。
//
// 返回空串 StmtType("") 表示未识别，交由调用方决定兜底执行（通常按通用非查询语句处理）。
func Classify(parser SQLParser, splitter SQLSplitter, sql string, stmt sqlstmt.Stmt, parseErr error) sqlstmt.StmtType {
	// 多语句保护：Parse 只解析到首个分号（各方言单语句语义），若整串含多条语句，
	// 按首句分类会让 "SELECT 1; DELETE FROM t" 被当成只读。调用方约定传单条，
	// 这里对违约输入直接判可能写（fail-closed），不再依赖调用方自觉拆句。
	if splitter != nil && strings.Contains(sql, ";") && !isRoutineDefinition(sql) &&
		countStatements(splitter, sql) > 1 {
		return sqlstmt.StmtTypeOtherWrite
	}
	// 1. AST 优先
	if parseErr == nil && stmt != nil {
		if t, ok := astStmtType(stmt); ok {
			return t
		}
	}
	// 2. 方言分类能力：仅在「解析出错」或「已产出 AST 但通用映射不识别」时二次判定。
	//    stmt==nil && parseErr==nil 表示无可分类语句（如纯可执行注释 /*!... */、空串），
	//    此时直接走关键字兜底，避免方言 ClassifyStmt 对 nil AST 解引用。
	if parser != nil && (stmt != nil || parseErr != nil) {
		// 分类是 SQLParser 的必需能力（不再是"可选能力 + nil 回退"）
		if t, err := parser.ClassifyStmt(sql); err == nil && t != "" {
			return t
		}
	}
	// 3. 首词关键字兜底（整词匹配，非子串包含）
	return StatementTypeByKeyword(splitter, sql)
}

// StatementTypeByKeyword 仅按语句首个整词关键字（已跳过前导空白与注释）判定语句类型。
//
// 与 Classify 的第 3 级兜底同源：供「无 AST、只需按语句性质做轻量分流」的调用方使用
// （如执行成功后判断是否 DDL 以失效元数据缓存），避免各处用前缀字符/Contains 各自判定而漂移。
// splitter 为 nil 或关键字未识别时返回空串 StmtType，由调用方决定兜底行为。
func StatementTypeByKeyword(splitter SQLSplitter, sql string) sqlstmt.StmtType {
	if splitter == nil {
		return ""
	}
	return stmtTypeFromKeyword(splitter.LeadingKeyword(sql))
}

// astStmtType 将解析出的 AST 节点类型映射为 StmtType；未知节点返回 ok=false 以触发后续兜底
func astStmtType(stmt sqlstmt.Stmt) (sqlstmt.StmtType, bool) {
	switch s := stmt.(type) {
	case *sqlstmt.SelectStmt:
		// SELECT ... INTO（建表/导出/变量）会写出数据或文件，不是纯读：判为写（insert），
		// 走写通道并受 DML 审批约束，避免「SELECT INTO 建表」绕过审批（与前端 isQueryStatement 同口径）
		if s.Into {
			return sqlstmt.StmtTypeInsert, true
		}
		return sqlstmt.StmtTypeSelect, true
	case *sqlstmt.WithStmt:
		// WITH 主语句动词由解析器回填（数据修改型 CTE 亦是写）；未回填按查询兜底，与历史一致
		if s.MainType != "" {
			return s.MainType, true
		}
		return sqlstmt.StmtTypeSelect, true
	case *sqlstmt.UpdateStmt:
		return sqlstmt.StmtTypeUpdate, true
	case *sqlstmt.DeleteStmt:
		return sqlstmt.StmtTypeDelete, true
	case *sqlstmt.InsertStmt:
		return sqlstmt.StmtTypeInsert, true
	case *sqlstmt.MergeStmt:
		return sqlstmt.StmtTypeMerge, true
	case *sqlstmt.ExplainStmt:
		// EXPLAIN ANALYZE <写语句> 会真执行内层写 → 判为该写类型（s.Inner）；
		// 纯 EXPLAIN <计划> / EXPLAIN ANALYZE <SELECT> 的 Inner 为空 → 只读
		if s.Inner != "" {
			return s.Inner, true
		}
		if s.Analyze {
			// EXPLAIN ANALYZE <读/未知>：内层会被真正执行，可能调用有副作用函数，
			// 不能当纯只读放行（fail-closed）
			return sqlstmt.StmtTypeOtherWrite, true
		}
		return sqlstmt.StmtTypeOtherRead, true
	case *sqlstmt.DdlStmt:
		return sqlstmt.StmtTypeDDL, true
	case *sqlstmt.ShowStmt:
		// SHOW 是元信息导航语句：只读，但不是 SELECT（历史实现伪装成 SelectStmt 致类型失真）
		return sqlstmt.StmtTypeOtherRead, true
	case *sqlstmt.OtherStmt:
		// 只读与否由解析期按首词白名单填好的 ReadOnly 决定（单一真源在 tokenizer 白名单）
		if s.ReadOnly {
			return sqlstmt.StmtTypeOtherRead, true
		}
		return sqlstmt.StmtTypeOtherWrite, true
	}
	// 通用映射：不带语义特例的语句类型在此表登记（新增语句只需补一行 Kind→StmtType）
	if t, ok := kindStmtTypes[stmt.StmtKind()]; ok {
		return t, true
	}
	return "", false
}

// kindStmtTypes 无特例语句的 Kind → StmtType 映射（表驱动）。
// 需要额外判断的语句（SELECT INTO / WITH 主语句 / EXPLAIN 内外层 / OTHER 读写）在上面的
// type switch 里优先处理，不进入本表。
var kindStmtTypes = map[sqlstmt.Kind]sqlstmt.StmtType{
	sqlstmt.KindSelect: sqlstmt.StmtTypeSelect,
	sqlstmt.KindInsert: sqlstmt.StmtTypeInsert,
	sqlstmt.KindUpdate: sqlstmt.StmtTypeUpdate,
	sqlstmt.KindDelete: sqlstmt.StmtTypeDelete,
	sqlstmt.KindMerge:  sqlstmt.StmtTypeMerge,
	sqlstmt.KindDdl:    sqlstmt.StmtTypeDDL,
	// SHOW 是只读导航语句（不是 SELECT）
	sqlstmt.KindShow: sqlstmt.StmtTypeOtherRead,
}

// stmtTypeFromKeyword 按语句首个整词关键字映射 StmtType。
//
// with 归入查询类（CTE，与 AST 路径对 with 的判定保持一致）；只读动词归 other-read、
// 已知写动词归 other-write、完全未知返回空串（由执行层按可能写处理）。
func stmtTypeFromKeyword(kw string) sqlstmt.StmtType {
	switch kw {
	case "select", "with", "table":
		return sqlstmt.StmtTypeSelect
	case "insert":
		return sqlstmt.StmtTypeInsert
	case "update":
		return sqlstmt.StmtTypeUpdate
	case "delete":
		return sqlstmt.StmtTypeDelete
	case "merge":
		return sqlstmt.StmtTypeMerge
	case "create", "alter", "drop", "truncate", "rename", "comment", "grant", "revoke":
		return sqlstmt.StmtTypeDDL
	}
	// 无 AST 时的首词兜底：与解析期填 ReadOnly 用的是同一份白名单（见 tokenizer.readOnlyLeadingKeywords）
	if tokenizer.IsReadOnlyLeadingKeyword(strings.ToUpper(kw)) {
		return sqlstmt.StmtTypeOtherRead
	}
	if tokenizer.IsWriteLeadingKeyword(strings.ToUpper(kw)) {
		return sqlstmt.StmtTypeOtherWrite
	}
	// 完全未知的首词：返回空串，由执行层按「未知即可能写」处理（doOtherWrite）
	return ""
}

// countStatements 用方言切割器数语句条数（仅用于多语句保护，不计错误：切不动按 1 条算）。
func countStatements(splitter SQLSplitter, sql string) int {
	count := 0
	_ = splitter.SplitSQL(strings.NewReader(sql), func(string) error {
		count++
		return nil
	})
	return count
}

// isRoutineDefinition 判断是否为「体内合法包含分号」的定义语句（触发器/存储过程/函数）。
//
// 这些语句的 BEGIN...END 体里必然有分号，按「多条语句」处理会把整条 DDL 降级成 other-write，
// 使审计类型与策略通道失准（它本身仍走写通道，降级不改变安全性，但会污染分类）。
func isRoutineDefinition(sql string) bool {
	fields := strings.Fields(strings.ToUpper(sql))
	if len(fields) < 2 || fields[0] != "CREATE" {
		return false
	}
	for i := 1; i < len(fields) && i < 4; i++ {
		switch fields[i] {
		case "TRIGGER", "PROCEDURE", "PROC", "FUNCTION", "ROUTINE":
			return true
		case "OR", "REPLACE", "DEFINER", "ALGORITHM", "DEFINER=", "TEMPORARY":
			continue
		default:
			// CREATE TABLE/VIEW/... 不是过程体定义
			if i == 1 && fields[1] != "OR" {
				return false
			}
		}
	}
	return false
}

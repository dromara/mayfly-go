package sqlparser

import (
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
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
	// 1. AST 优先
	if parseErr == nil && stmt != nil {
		if t, ok := astStmtType(stmt); ok {
			return t
		}
	}
	// 2. 方言分类能力：仅在「解析出错」或「已产出 AST 但通用映射不识别」时二次判定。
	//    stmt==nil && parseErr==nil 表示无可分类语句（如纯可执行注释 /*!... */、空串），
	//    此时直接走关键字兜底，避免方言 ClassifyStmt 对 nil AST 解引用。
	if stmt != nil || parseErr != nil {
		if sc := GetStatementClassifier(parser); sc != nil {
			if t, err := sc.ClassifyStmt(sql); err == nil && t != "" {
				return t
			}
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
	switch stmt.(type) {
	case *sqlstmt.SelectStmt, *sqlstmt.WithStmt:
		return sqlstmt.StmtTypeSelect, true
	case *sqlstmt.UpdateStmt:
		return sqlstmt.StmtTypeUpdate, true
	case *sqlstmt.DeleteStmt:
		return sqlstmt.StmtTypeDelete, true
	case *sqlstmt.InsertStmt:
		return sqlstmt.StmtTypeInsert, true
	case *sqlstmt.DdlStmt:
		return sqlstmt.StmtTypeDDL, true
	case *sqlstmt.OtherStmt:
		return sqlstmt.StmtTypeOther, true
	}
	return "", false
}

// stmtTypeFromKeyword 按语句首个整词关键字映射 StmtType。
//
// with 归入查询类（CTE，与 AST 路径对 with 的判定保持一致）；
// show/explain/desc 等只读元信息语句归 Other；未识别动词返回空串交调用方兜底。
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
	case "create", "alter", "drop", "truncate", "rename", "comment", "grant", "revoke":
		return sqlstmt.StmtTypeDDL
	case "show", "explain", "describe", "desc":
		return sqlstmt.StmtTypeOther
	default:
		return ""
	}
}

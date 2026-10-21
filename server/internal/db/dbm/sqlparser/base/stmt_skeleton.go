package base

// 通用语句骨架解析：CREATE/ALTER/DROP/WITH 等语句各方言的宽容解析行为一致
// （跳至语句结束并包装为对应Stmt类型），统一下沉至此供方言复用；
// 方言如需方言特有语义（如mysql的REPLACE INTO、pg的ON CONFLICT）在各自
// parseStatement 分发或专用方法中覆写即可

import (
	"strings"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// ParseCreateStmt CREATE语句骨架：跳至语句结束，包装为DdlStmt(CREATE)
func (l *Lexer) ParseCreateStmt() sqlstmt.Stmt {
	start := l.Pos
	l.SkipToNextStatement()
	return &sqlstmt.DdlStmt{
		Base:    sqlstmt.Base{Text: l.TextFrom(start)},
		DdlKind: "CREATE",
	}
}

// ParseAlterStmt ALTER语句骨架：跳至语句结束，包装为DdlStmt(ALTER)
func (l *Lexer) ParseAlterStmt() sqlstmt.Stmt {
	start := l.Pos
	l.SkipToNextStatement()
	return &sqlstmt.DdlStmt{
		Base:    sqlstmt.Base{Text: l.TextFrom(start)},
		DdlKind: "ALTER",
	}
}

// ParseDropStmt DROP语句骨架：跳至语句结束，包装为DdlStmt(DROP)
func (l *Lexer) ParseDropStmt() sqlstmt.Stmt {
	start := l.Pos
	l.SkipToNextStatement()
	return &sqlstmt.DdlStmt{
		Base:    sqlstmt.Base{Text: l.TextFrom(start)},
		DdlKind: "DROP",
	}
}

// ParseSharedStatement 分发各方言语义完全一致的语句头：SELECT（含外层括号形态）、
// UPDATE/DELETE/MERGE（多表写法与尾部子句已由骨架统一登记）、CREATE/DROP/ALTER/WITH，
// 以及只需语句边界的非结构化 DDL。
// 返回 ok=false 表示该语句头存在真正的方言差异（mysql 的 REPLACE/SHOW 等），
// 必须由方言自己的分发处理，base 不做猜测；
// INSERT 走通用形态，有冲突处理子句的方言必须在调用本分发之前先接管 INSERT
func (l *Lexer) ParseSharedStatement() (sqlstmt.Stmt, bool) {
	tok := l.Current()
	switch {
	case tok.IsKeyword("SELECT") || tok.Value == "(":
		// 分发条件已保证骨架能解析出主体（括号内非 SELECT 时按查询形态降级保留原文）
		sel, _ := l.parseSelectStatement()
		return sel, true
	case tok.IsKeyword("INSERT"):
		// 方言若有冲突处理子句，需在调用本分发之前自行处理 INSERT
		return l.parseInsert(), true
	case tok.IsKeyword("UPDATE"):
		return l.ParseUpdate(), true
	case tok.IsKeyword("DELETE"):
		return l.ParseDelete(), true
	case isMergeAt(l.Tokens, l.Pos):
		// MERGE INTO（upsert）：按「值 MERGE + 后随 INTO」判定，不依赖关键字登记（见 isMergeAt）
		return l.parseMergeStmt(), true
	case strings.EqualFold(tok.Value, "EXPLAIN"):
		// EXPLAIN（语句头值匹配，EXPLAIN 未登记为关键字；语句不可能以裸标识符开头，故头位值匹配安全）：
		// 区分「仅出计划」与「EXPLAIN ANALYZE 会真执行内层写」
		return l.parseExplainStmt(), true
	case tok.IsKeyword("CREATE"):
		return l.ParseCreateStmt(), true
	case tok.IsKeyword("DROP"):
		return l.ParseDropStmt(), true
	case tok.IsKeyword("ALTER"):
		return l.ParseAlterStmt(), true
	case tok.IsKeyword("WITH"):
		return l.ParseWithStmt(), true
	case tok.IsKeyword("TRUNCATE", "COMMENT", "GRANT", "REVOKE", "RENAME", "ANALYZE"):
		// 非查询语句统一按 DDL 执行
		return l.parseDdlStmt(), true
	}
	return nil, false
}

// parseDdlStmt 非结构化 DDL 骨架：跳至语句结束并包装为 DdlStmt(DDL)。
// TRUNCATE / COMMENT ON / GRANT / REVOKE / RENAME / ANALYZE 等只需语句边界与文本，无需结构化建模
func (l *Lexer) parseDdlStmt() sqlstmt.Stmt {
	start := l.Pos
	l.SkipToNextStatement()
	return &sqlstmt.DdlStmt{
		Base:    sqlstmt.Base{Text: l.TextFrom(start)},
		DdlKind: "DDL",
	}
}

// ParseOtherStmt 未知语句骨架：跳至语句结束，包装为 OtherStmt
func (l *Lexer) ParseOtherStmt() sqlstmt.Stmt {
	start := l.Pos
	l.SkipToNextStatement()
	return &sqlstmt.OtherStmt{
		Base: sqlstmt.Base{Text: l.TextFrom(start)},
	}
}

// ParseWithStmt WITH语句骨架：先扫描出主语句/数据修改 CTE 的实际类型回填 MainType，再跳至语句结束。
// （CTE 主体不做结构化建模，仅需语句边界、文本与「这条 WITH 到底是读还是写」的判定）
func (l *Lexer) ParseWithStmt() sqlstmt.Stmt {
	start := l.Pos
	mainType := l.scanWithMainType()
	l.SkipToNextStatement()
	return &sqlstmt.WithStmt{
		Base:     sqlstmt.Base{Text: l.TextFrom(start)},
		MainType: mainType,
	}
}

// parseMergeStmt MERGE 语句骨架：跳至语句结束，包装为 MergeStmt。
// MERGE 是写语句（upsert），结构不建模（各方言子句差异大、不纳入回滚快照），仅需边界与文本。
func (l *Lexer) parseMergeStmt() sqlstmt.Stmt {
	start := l.Pos
	l.SkipToNextStatement()
	return &sqlstmt.MergeStmt{
		Base: sqlstmt.Base{Text: l.TextFrom(start)},
	}
}

// parseExplainStmt EXPLAIN 语句：先扫描内层是否为「会被 ANALYZE 真执行的写语句」（见 ExplainStmt），
// 随后跳至语句结束（EXPLAIN 不建模内部结构）。
func (l *Lexer) parseExplainStmt() sqlstmt.Stmt {
	start := l.Pos
	inner := l.scanExplainInnerWrite()
	l.SkipToNextStatement()
	return &sqlstmt.ExplainStmt{
		Base:  sqlstmt.Base{Text: l.TextFrom(start)},
		Inner: inner,
	}
}

// scanExplainInnerWrite 从 EXPLAIN 之后扫描内层语句：
//   - 遇 ANALYZE/ANALYSE（裸词或 pg 的括号选项形态）记 analyze=true；
//   - 遇到的第一个语句动词（SELECT/INSERT/UPDATE/DELETE/MERGE INTO）即被解释的内层语句头。
//
// 仅当 analyze 且内层为写动词（非 SELECT）时返回该写类型（EXPLAIN ANALYZE <写> 会真的写）；
// 纯 EXPLAIN <任意>（只出计划）或 EXPLAIN ANALYZE <SELECT>（读）均返回空串，按 Other 只读处理。
// tokenizer 已丢弃注释、把字符串/引用标识符归为非关键字整体 token，故字符串里的 'delete' 不会被当成内层动词。
func (l *Lexer) scanExplainInnerWrite() sqlstmt.StmtType {
	analyze := false
	for i := l.Pos + 1; i < l.Length; i++ {
		tok := l.Tokens[i]
		if tok.Type == tokenizer.TokenOperator && tok.Value == ";" {
			break
		}
		if tok.IsKeyword("ANALYZE") || strings.EqualFold(tok.Value, "ANALYSE") {
			analyze = true
			continue
		}
		if vt := l.stmtVerbTypeAt(i); vt != "" {
			if analyze && vt != sqlstmt.StmtTypeSelect {
				return vt
			}
			return ""
		}
	}
	return ""
}

// scanWithMainType 扫描 WITH 语句 token，解析其有效语句类型（只读 l.Tokens，不移动 l.Pos）：
//   - 主语句动词：CTE 列表之后、括号深度 0 上第一个 SELECT/INSERT/UPDATE/DELETE/MERGE INTO；
//   - 数据修改型 CTE：深度 0 的 "(" 体首 token 是 INSERT/UPDATE/DELETE/MERGE（`WITH x AS (DELETE ...) SELECT ...`）。
//
// 任一为 DML 即判为写（返回对应 StmtType）；纯查询（含只读 CTE）返回 StmtTypeSelect。
// tokenizer 丢弃注释、并把字符串/引用标识符归为非关键字的整体 token，故体内出现的 'delete' 字面量、
// 或注释里的 DELETE 都不会被误判为 DML（SELECT/INSERT/UPDATE/DELETE 是全局保留字，MERGE 按值+INTO 判，各方言一致）。
func (l *Lexer) scanWithMainType() sqlstmt.StmtType {
	mainType := sqlstmt.StmtType("")
	cteWrite := sqlstmt.StmtType("")
	depth := 0
	for i := l.Pos; i < l.Length; i++ {
		tok := l.Tokens[i]
		if tok.IsEOF() || tok.Value == ";" {
			break
		}
		switch tok.Value {
		case "(":
			// 深度 0 的 "(" 是 CTE 体（列清单 (a,b) 体首是标识符，不会命中 DML）
			if depth == 0 && cteWrite == "" {
				cteWrite = l.dmlTypeAt(i + 1)
			}
			depth++
		case ")":
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 && mainType == "" {
				mainType = l.stmtVerbTypeAt(i)
			}
		}
	}
	switch mainType {
	case sqlstmt.StmtTypeInsert, sqlstmt.StmtTypeUpdate, sqlstmt.StmtTypeDelete, sqlstmt.StmtTypeMerge:
		return mainType
	}
	if cteWrite != "" {
		return cteWrite
	}
	return sqlstmt.StmtTypeSelect
}

// stmtVerbTypeAt：索引 i 处若为语句头动词（SELECT/INSERT/UPDATE/DELETE/MERGE INTO）→ StmtType；否则 ""
func (l *Lexer) stmtVerbTypeAt(i int) sqlstmt.StmtType {
	if i < 0 || i >= l.Length {
		return ""
	}
	tok := l.Tokens[i]
	switch {
	case tok.IsKeyword("SELECT"):
		return sqlstmt.StmtTypeSelect
	case tok.IsKeyword("INSERT"):
		return sqlstmt.StmtTypeInsert
	case tok.IsKeyword("UPDATE"):
		return sqlstmt.StmtTypeUpdate
	case tok.IsKeyword("DELETE"):
		return sqlstmt.StmtTypeDelete
	case isMergeAt(l.Tokens, i):
		return sqlstmt.StmtTypeMerge
	}
	return ""
}

// dmlTypeAt：索引 i 处若为 DML 写动词（INSERT/UPDATE/DELETE/MERGE INTO）→ StmtType；SELECT/其他 ""
func (l *Lexer) dmlTypeAt(i int) sqlstmt.StmtType {
	if i < 0 || i >= l.Length {
		return ""
	}
	tok := l.Tokens[i]
	switch {
	case tok.IsKeyword("INSERT"):
		return sqlstmt.StmtTypeInsert
	case tok.IsKeyword("UPDATE"):
		return sqlstmt.StmtTypeUpdate
	case tok.IsKeyword("DELETE"):
		return sqlstmt.StmtTypeDelete
	case isMergeAt(l.Tokens, i):
		return sqlstmt.StmtTypeMerge
	}
	return ""
}

// isMergeAt：tokens[i] 值为 MERGE 且其后一个为 INTO（大小写不敏感）。
// 按「值 + 后随 INTO」判定而非登记关键字：MERGE 仅部分方言支持、保留字状态不一，
// 值匹配既方言统一，又不影响 merge 作裸标识符（语句头不可能是裸标识符，误判面为零）。
func isMergeAt(tokens []tokenizer.Token, i int) bool {
	if i < 0 || i+1 >= len(tokens) {
		return false
	}
	return strings.EqualFold(tokens[i].Value, "MERGE") && strings.EqualFold(tokens[i+1].Value, "INTO")
}

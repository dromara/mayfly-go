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

// parseObjectNameAt 从 token 下标 idx 起读一个对象名（跳过 IF [NOT] EXISTS / ONLY 等修饰词）。
//
// 只读 token 流，不推进语句解析位置（保存并恢复 l.Pos），因此可安全用于「先取名字再跳到底」的 DDL 骨架。
func (l *Lexer) parseObjectNameAt(idx int) string {
	for idx < l.Length {
		switch strings.ToUpper(l.Tokens[idx].Value) {
		case "IF", "NOT", "EXISTS", "ONLY":
			idx++
			continue
		}
		break
	}
	save := l.Pos
	l.Pos = idx
	ref := l.parseTableRef()
	l.Pos = save
	return ref.Name
}

// ddlTargetObject 从 DDL 语句头之后定位目标对象名：
//   - TABLE / VIEW / TRIGGER：紧随其后的对象名（支持 IF [NOT] EXISTS 与 schema 限定）；
//   - INDEX：取 ON 之后的**表**名（DROP INDEX idx ON t / CREATE INDEX idx ON t）；
//   - 其他对象类型（SCHEMA/DATABASE/FUNCTION/PROCEDURE…）返回空串——它们不是表，
//     塞进 Target 会让「指定表」策略误命中。
func (l *Lexer) ddlTargetObject() string {
	for i := l.Pos; i < l.Length; i++ {
		tok := l.Tokens[i]
		if tok.Type == tokenizer.TokenOperator && tok.Value == ";" {
			return ""
		}
		switch strings.ToUpper(tok.Value) {
		case "TABLE", "VIEW", "TRIGGER":
			return l.parseObjectNameAt(i + 1)
		case "INDEX":
			for k := i + 1; k < l.Length; k++ {
				if l.Tokens[k].Type == tokenizer.TokenOperator && l.Tokens[k].Value == ";" {
					return ""
				}
				if strings.EqualFold(l.Tokens[k].Value, "ON") {
					return l.parseObjectNameAt(k + 1)
				}
			}
			return ""
		}
	}
	return ""
}

// ddlRenamePairs 提取表级改名对（`RENAME TABLE a TO b, c TO d` / `ALTER TABLE t RENAME TO t2`）。
//
// 只认 TO 形态且 FROM 必须是表名位置：`ALTER TABLE t RENAME COLUMN a TO b` 里的 a/b 是列，
// 不能进 Renames（否则表级策略会在列名上误命中）。取不到 FROM 就跳过（不猜）。
func (l *Lexer) ddlRenamePairs() []sqlstmt.Rename {
	var pairs []sqlstmt.Rename
	renameSeen := false
	for i := l.Pos; i < l.Length; i++ {
		tok := l.Tokens[i]
		if tok.Type == tokenizer.TokenOperator && tok.Value == ";" {
			break
		}
		if strings.EqualFold(tok.Value, "RENAME") {
			renameSeen = true
			continue
		}
		if !renameSeen || !strings.EqualFold(tok.Value, "TO") {
			continue
		}
		// RENAME COLUMN a TO b：列改名，整条跳过
		if i >= 2 && strings.EqualFold(l.Tokens[i-2].Value, "COLUMN") {
			continue
		}
		to := l.parseObjectNameAt(i + 1)
		if to == "" {
			continue
		}
		fromIdx := -1
		for j := i - 1; j >= l.Pos; j-- {
			t := l.Tokens[j]
			if t.Type == tokenizer.TokenIdentifier {
				fromIdx = j
				break
			}
			if t.Type == tokenizer.TokenKeyword && !strings.EqualFold(t.Value, "RENAME") && !strings.EqualFold(t.Value, "TABLE") {
				fromIdx = j
				break
			}
		}
		if fromIdx < 0 {
			continue
		}
		if from := l.parseObjectNameAt(fromIdx); from != "" && !strings.EqualFold(from, to) {
			pairs = append(pairs, sqlstmt.Rename{From: from, To: to})
		}
	}
	return pairs
}

// ParseCreateStmt CREATE语句骨架：跳至语句结束，包装为DdlStmt(CREATE)
func (l *Lexer) ParseCreateStmt() sqlstmt.Stmt {
	start := l.Pos
	target := l.ddlTargetObject()
	// 列定义同样必须在 Skip 之前提取（Skip 后 Pos 已到语句末尾）
	columns := l.ddlColumnDefs()
	l.SkipToNextStatement()
	return &sqlstmt.DdlStmt{
		Base:    sqlstmt.Base{Text: l.TextFrom(start)},
		DdlKind: "CREATE",
		Target:  target,
		Columns: columns,
	}
}

// ParseAlterStmt ALTER语句骨架：跳至语句结束，包装为DdlStmt(ALTER)
func (l *Lexer) ParseAlterStmt() sqlstmt.Stmt {
	start := l.Pos
	// 同样先提取再 Skip（Skip 之后 Pos 已到末尾，ddlRenamePairs 会扫不到 TO）
	target := l.ddlTargetObject()
	renames := l.ddlRenamePairs()
	l.SkipToNextStatement()
	return &sqlstmt.DdlStmt{
		Base:    sqlstmt.Base{Text: l.TextFrom(start)},
		DdlKind: "ALTER",
		Target:  target,
		Renames: renames,
	}
}

// ParseDropStmt DROP语句骨架：跳至语句结束，包装为DdlStmt(DROP)
func (l *Lexer) ParseDropStmt() sqlstmt.Stmt {
	start := l.Pos
	target := l.ddlTargetObject()
	l.SkipToNextStatement()
	return &sqlstmt.DdlStmt{
		Base:    sqlstmt.Base{Text: l.TextFrom(start)},
		DdlKind: "DROP",
		Target:  target,
	}
}

// parseDdlStmt 非结构化 DDL 骨架：跳至语句结束并包装为 DdlStmt(DDL)。
// TRUNCATE / COMMENT ON / GRANT / REVOKE / RENAME / ANALYZE 等只需语句边界与文本，无需结构化建模
func (l *Lexer) parseDdlStmt() sqlstmt.Stmt {
	start := l.Pos
	// 必须在 Skip 之前提取：SkipToNextStatement 会把 Pos 推到语句末尾，之后再扫就是空的
	target := l.ddlTargetObject()
	renames := l.ddlRenamePairs()
	l.SkipToNextStatement()
	return &sqlstmt.DdlStmt{
		Base:    sqlstmt.Base{Text: l.TextFrom(start)},
		DdlKind: "DDL",
		Target:  target,
		Renames: renames,
	}
}

// ParseOtherStmt 未知语句骨架：跳至语句结束，包装为 OtherStmt
func (l *Lexer) ParseOtherStmt() sqlstmt.Stmt {
	start := l.Pos
	l.SkipToNextStatement()
	text := l.TextFrom(start)
	// 只读与否按整条语句的首词白名单判定（含方言注释/可执行注释语义），
	// 白名单外一律按可能写处理：执行层据此选写通道，避免写语句从只读通道溜过。
	readOnly := tokenizer.IsReadOnlyLeadingKeyword(tokenizer.LeadingKeyword(text, l.config))
	return sqlstmt.NewOtherStmt(text, readOnly)
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
	// MERGE INTO <target> USING ...：INTO 之后即目标表；取不到留空（不猜）
	target := ""
	for i := l.Pos; i < l.Length; i++ {
		tok := l.Tokens[i]
		if tok.Type == tokenizer.TokenOperator && tok.Value == ";" {
			break
		}
		if strings.EqualFold(tok.Value, "INTO") {
			target = l.parseObjectNameAt(i + 1)
			break
		}
	}
	l.SkipToNextStatement()
	return &sqlstmt.MergeStmt{
		Base:   sqlstmt.Base{Text: l.TextFrom(start)},
		Target: target,
	}
}

// parseExplainStmt EXPLAIN 语句：先扫描内层是否为「会被 ANALYZE 真执行的写语句」（见 ExplainStmt），
// 随后跳至语句结束（EXPLAIN 不建模内部结构）。
func (l *Lexer) parseExplainStmt() sqlstmt.Stmt {
	start := l.Pos
	inner, analyze := l.scanExplainInnerWrite()
	l.SkipToNextStatement()
	return &sqlstmt.ExplainStmt{
		Base:    sqlstmt.Base{Text: l.TextFrom(start)},
		Inner:   inner,
		Analyze: analyze,
	}
}

// scanExplainInnerWrite 从 EXPLAIN 之后扫描内层语句：
//   - 遇 ANALYZE/ANALYSE（裸词或 pg 的括号选项形态）记 analyze=true；
//   - 遇到的第一个语句动词（SELECT/INSERT/UPDATE/DELETE/MERGE INTO）即被解释的内层语句头。
//
// 仅当 analyze 且内层为写动词（非 SELECT）时返回该写类型（EXPLAIN ANALYZE <写> 会真的写）；
// 纯 EXPLAIN <任意>（只出计划）或 EXPLAIN ANALYZE <SELECT>（读）均返回空串，按 Other 只读处理。
// tokenizer 已丢弃注释、把字符串/引用标识符归为非关键字整体 token，故字符串里的 'delete' 不会被当成内层动词。
func (l *Lexer) scanExplainInnerWrite() (sqlstmt.StmtType, bool) {
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
				return vt, analyze
			}
			return "", analyze
		}
	}
	return "", analyze
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

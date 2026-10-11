package base

// 通用语句骨架解析：CREATE/ALTER/DROP/WITH 等语句各方言的宽容解析行为一致
// （跳至语句结束并包装为对应Stmt类型），统一下沉至此供方言复用；
// 方言如需方言特有语义（如mysql的REPLACE INTO、pg的ON CONFLICT）在各自
// parseStatement 分发或专用方法中覆写即可

import "mayfly-go/internal/db/dbm/sqlparser/sqlstmt"

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
// UPDATE/DELETE（多表写法与尾部子句已由骨架统一登记）、CREATE/DROP/ALTER/WITH，
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

// ParseWithStmt WITH语句骨架：跳至语句结束，包装为WithStmt
// （CTE主体结构化建模成本高，当前场景仅需判定语句边界与文本回填）
func (l *Lexer) ParseWithStmt() sqlstmt.Stmt {
	start := l.Pos
	l.SkipToNextStatement()
	return &sqlstmt.WithStmt{
		Base: sqlstmt.Base{Text: l.TextFrom(start)},
	}
}

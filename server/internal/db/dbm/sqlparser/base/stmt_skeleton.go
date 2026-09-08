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

// ParseWithStmt WITH语句骨架：跳至语句结束，包装为WithStmt
// （CTE主体结构化建模成本高，当前场景仅需判定语句边界与文本回填）
func (l *Lexer) ParseWithStmt() sqlstmt.Stmt {
	start := l.Pos
	l.SkipToNextStatement()
	return &sqlstmt.WithStmt{
		Base: sqlstmt.Base{Text: l.TextFrom(start)},
	}
}

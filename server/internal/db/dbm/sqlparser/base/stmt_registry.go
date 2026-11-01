package base

import (
	"strings"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// stmtSpec 一种「各方言语义一致」语句的注册项。
//
// 这是新增语句类型的**唯一注册点**（另需一个 AST 节点）：语句头判定、解析函数、
// 产物 Kind、在触发策略里的归属类型都集中在此。分发（ParseSharedStatement）与
// 策略选项（PolicyStatementTypes）都从这张表读，不再散在 switch/数组两处维护。
//
// 方言特有形态（MySQL 的 REPLACE/SHOW、PG 的 ON CONFLICT 等）仍由方言自己的
// Parse 入口先接管，再回落到本注册表——注册表只承载「共享骨架能一次实现」的部分。
type stmtSpec struct {
	// Name 稳定标识（用于自测与错误信息），与 StmtType 一致
	Name string
	// Kind 解析产物的 AST Kind
	Kind sqlstmt.Kind
	// PolicyType 该语句在触发策略「语句类型」里的归属（空表示不进入策略选项）。
	// 例如 EXPLAIN 归 other-read、CREATE/ALTER/DROP/TRUNCATE 归 ddl、WITH 归 select。
	PolicyType sqlstmt.StmtType
	// Match 语句头判定：关键字、词值或结构判据（如 MERGE 需「值 MERGE + 后随 INTO」）
	Match func(l *Lexer) bool
	// Parse 解析函数（必须消费到语句结束）
	Parse func(l *Lexer) sqlstmt.Stmt
}

// sharedStatementSpecs 共享语句注册表：**顺序即分发优先级**。
//
// 新增一种共享语句 = 在此加一项 + 在 sqlstmt 加一个 AST 节点 + 在 classify 的
// Kind 映射表里加一行（若不走特例）。其余（分发、策略选项、路由键）自动生效。
var sharedStatementSpecs = []stmtSpec{
	{
		Name:       "select",
		Kind:       sqlstmt.KindSelect,
		PolicyType: sqlstmt.StmtTypeSelect,
		Match: func(l *Lexer) bool {
			// 外层括号形态按查询降级保留原文（括号内非 SELECT 时骨架自行兜底）
			return l.Current().IsKeyword("SELECT") || l.Current().Value == "("
		},
		Parse: func(l *Lexer) sqlstmt.Stmt { s, _ := l.parseSelectStatement(); return s },
	},
	{
		Name:       "insert",
		Kind:       sqlstmt.KindInsert,
		PolicyType: sqlstmt.StmtTypeInsert,
		Match:      func(l *Lexer) bool { return l.Current().IsKeyword("INSERT") },
		Parse:      func(l *Lexer) sqlstmt.Stmt { return l.parseInsert() },
	},
	{
		Name:       "update",
		Kind:       sqlstmt.KindUpdate,
		PolicyType: sqlstmt.StmtTypeUpdate,
		Match:      func(l *Lexer) bool { return l.Current().IsKeyword("UPDATE") },
		Parse:      func(l *Lexer) sqlstmt.Stmt { return l.ParseUpdate() },
	},
	{
		Name:       "delete",
		Kind:       sqlstmt.KindDelete,
		PolicyType: sqlstmt.StmtTypeDelete,
		Match:      func(l *Lexer) bool { return l.Current().IsKeyword("DELETE") },
		Parse:      func(l *Lexer) sqlstmt.Stmt { return l.ParseDelete() },
	},
	{
		Name:       "merge",
		Kind:       sqlstmt.KindMerge,
		PolicyType: sqlstmt.StmtTypeMerge,
		// MERGE 未登记为关键字（避免污染名称位），按「值 MERGE + 后随 INTO」结构判据
		Match: func(l *Lexer) bool { return isMergeAt(l.Tokens, l.Pos) },
		Parse: func(l *Lexer) sqlstmt.Stmt { return l.parseMergeStmt() },
	},
	{
		Name:       "explain",
		Kind:       sqlstmt.KindOther,
		PolicyType: sqlstmt.StmtTypeOtherRead,
		// EXPLAIN 未登记关键字，语句头位置做值匹配（语句不可能以裸标识符开头）
		Match: func(l *Lexer) bool { return strings.EqualFold(l.Current().Value, "EXPLAIN") },
		Parse: func(l *Lexer) sqlstmt.Stmt { return l.parseExplainStmt() },
	},
	{
		Name:       "create",
		Kind:       sqlstmt.KindDdl,
		PolicyType: sqlstmt.StmtTypeDDL,
		Match:      func(l *Lexer) bool { return l.Current().IsKeyword("CREATE") },
		Parse:      func(l *Lexer) sqlstmt.Stmt { return l.ParseCreateStmt() },
	},
	{
		Name:       "drop",
		Kind:       sqlstmt.KindDdl,
		PolicyType: sqlstmt.StmtTypeDDL,
		Match:      func(l *Lexer) bool { return l.Current().IsKeyword("DROP") },
		Parse:      func(l *Lexer) sqlstmt.Stmt { return l.ParseDropStmt() },
	},
	{
		Name:       "alter",
		Kind:       sqlstmt.KindDdl,
		PolicyType: sqlstmt.StmtTypeDDL,
		Match:      func(l *Lexer) bool { return l.Current().IsKeyword("ALTER") },
		Parse:      func(l *Lexer) sqlstmt.Stmt { return l.ParseAlterStmt() },
	},
	{
		Name:       "with",
		Kind:       sqlstmt.KindWith,
		PolicyType: sqlstmt.StmtTypeSelect,
		Match:      func(l *Lexer) bool { return l.Current().IsKeyword("WITH") },
		Parse:      func(l *Lexer) sqlstmt.Stmt { return l.ParseWithStmt() },
	},
	{
		Name:       "ddl",
		Kind:       sqlstmt.KindDdl,
		PolicyType: sqlstmt.StmtTypeDDL,
		// 只需语句边界的非结构化 DDL（TRUNCATE/COMMENT ON/GRANT/REVOKE/RENAME/ANALYZE）
		// 注意不含 ANALYZE：它是只读维护语句（收集统计），应落 OtherStmt → other-read；
		// 归到 ddl 会与 tokenizer 的只读白名单冲突（同一语句两种分类）。
		// OPTIMIZE（ClickHouse 表维护）归 ddl：它会重写数据文件，属破坏性维护。
		Match: func(l *Lexer) bool {
			return l.Current().IsKeyword("TRUNCATE", "COMMENT", "GRANT", "REVOKE", "RENAME") || l.wordIs("OPTIMIZE")
		},
		Parse: func(l *Lexer) sqlstmt.Stmt { return l.parseDdlStmt() },
	},
}

// ParseSharedStatement 按注册表分发共享语句；返回 ok=false 表示该语句头属方言特有
// （如 MySQL 的 REPLACE/SHOW），须由方言自己的分发处理——base 不做猜测。
//
// INSERT 走通用形态；有冲突处理子句的方言必须在调用本分发之前先接管 INSERT。
func (l *Lexer) ParseSharedStatement() (sqlstmt.Stmt, bool) {
	for _, spec := range sharedStatementSpecs {
		if spec.Match(l) {
			return spec.Parse(l), true
		}
	}
	return nil, false
}

// SharedStatementNames 注册表里的语句名（顺序即分发优先级），供自测与文档使用。
func SharedStatementNames() []string {
	names := make([]string, 0, len(sharedStatementSpecs))
	for _, spec := range sharedStatementSpecs {
		names = append(names, spec.Name)
	}
	return names
}

// PolicyStatementTypes 触发策略「语句类型」可选值的稳定集合（去重保序）。
//
// 共享语句按注册表的 PolicyType 归属汇总（EXPLAIN→other-read、WITH→select、
// CREATE/ALTER/DROP/TRUNCATE→ddl），再补上执行层才会产出的语义类型
// （read = 只读导航语句走 doOtherRead；other-read/other-write = 未建模语句的读写分流）。
//
// 新增语句类型只要在注册表填好 PolicyType 就自动出现在策略选项里，无需再改数组。
func PolicyStatementTypes() []string {
	seen := make(map[string]bool, len(sharedStatementSpecs)+3)
	types := make([]string, 0, len(sharedStatementSpecs)+3)
	add := func(t string) {
		if t == "" || seen[t] {
			return
		}
		seen[t] = true
		types = append(types, t)
	}
	for _, spec := range sharedStatementSpecs {
		add(string(spec.PolicyType))
	}
	// 执行层专有语义类型（不是语句头，而是「通道归属」）
	for _, t := range []string{"read", string(sqlstmt.StmtTypeOtherRead), string(sqlstmt.StmtTypeOtherWrite)} {
		add(t)
	}
	return types
}

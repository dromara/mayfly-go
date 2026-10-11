package base

// 表引用、JOIN 与 FROM 子句的统一解析实现。
//
// 四方言曾各写一份近似代码，漂移会以「静默少解析一段」暴露：
//   - 派生表分支曾把别名一并写进 TableRef.Name（四方言全中），使下游按文本再解析内层 SQL 时多带一段；
//   - 括号内子查询在部分方言走「递归建树」而树实际被丢弃（TableRef 无子查询字段，
//     脱敏血缘是按 Name 文本重新解析的），且遇 `(SELECT ... UNION SELECT ...) x` 会在 UNION 处停下，
//     把别名与其后的 WHERE 整段丢掉；
//   - IsJoinStart 含 STRAIGHT_JOIN 但各方言的 parseJoinClause 无对应分支，且该词从未登记为关键字。
//
// 因此这里统一实现：结构差异交由词法能力表（tokenizer.DialectConfig）表达，语法骨架只留一份。

import (
	"strings"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// parseTableRef 解析单个表引用：schema.table、带/不带引用符、别名（AS 或隐式）、派生表与括号表。
//
// 派生表只消费括号内容而不递归建子查询树：TableRef 不持有子查询节点，递归产物无人消费；
// 且 SkipParentheses 按括号配对消费，天然能吃下 UNION 等任何内层形态。
// Name 只保留括号表达式本身，别名交给 Alias（下游据此重新解析内层 SQL）。
func (l *Lexer) parseTableRef() sqlstmt.TableRef {
	// 括号项：派生表或括号表名列表
	if l.Current().Value == "(" {
		start := l.Pos
		l.SkipParentheses()
		return sqlstmt.TableRef{
			Name:  l.TextFromExclusive(start),
			Alias: l.parseOptionalTableAlias(),
		}
	}

	if !l.IsUsableName() {
		return sqlstmt.TableRef{}
	}

	// 逐段吃掉 a.b.c...：段数不限（mysql 的 db.table、T-SQL 的 server.db.schema.table）。
	// 最后一段是表名，之前的段合成限定前缀写入 Schema——
	// 消费方（回滚采集层）只看 Schema 是否非空来判定「跨 schema 限定，无法按默认库定位」，
	// 因此 n 段与 2 段同样被 fail-closed，不会因为段数变化而漏判
	segments := []string{l.Consume().Value}
	for l.Current().Value == "." && l.isUsableNameAt(1) {
		l.Consume()
		segments = append(segments, l.Consume().Value)
	}

	ref := sqlstmt.TableRef{Name: l.Unquote(segments[len(segments)-1])}
	if len(segments) > 1 {
		qualifiers := make([]string, 0, len(segments)-1)
		for _, seg := range segments[:len(segments)-1] {
			qualifiers = append(qualifiers, l.Unquote(seg))
		}
		ref.Schema = strings.Join(qualifiers, ".")
	}

	ref.Alias = l.parseOptionalTableAlias()
	return ref
}

// parseOptionalTableAlias 解析可选的表别名（AS 显式，或隐式的裸名称）
//
// 名称位一律走 IsUsableName：它已排除所有子句边界词与（声明了层次能力的方言的）
// `START WITH` / `CONNECT BY` 头部词对，因此 `FROM t START WITH ...` 里的 start
// 不会被当成别名而把整个层次子句吞掉，`FROM quick`（quick 是 mysql 登记的修饰词）
// 也不会解析成空表
func (l *Lexer) parseOptionalTableAlias() string {
	if l.Current().IsKeyword("AS") {
		l.Consume()
	}
	if !l.IsUsableName() {
		return ""
	}
	return l.Unquote(l.Consume().Value)
}

// parseJoinClause 解析一个 JOIN 子句（含 CROSS/NATURAL/STRAIGHT_JOIN 与 ON/USING 条件）。
// 起始 token 解析不出合法 JOIN 时恢复位置并返回 nil，由调用方退出循环（防止 Pos 不前进导致死循环）。
func (l *Lexer) parseJoinClause() *sqlstmt.JoinClause {
	start := l.Pos
	joinType := sqlstmt.JoinKindInner
	joined := false

	switch {
	case l.Current().IsKeyword("STRAIGHT_JOIN"):
		// mysql 特有：等价于 INNER JOIN，仅强制左表先行；该 token 自身即连接词，后不再跟 JOIN
		l.Consume()
		joined = true
	case l.Current().IsKeyword("LEFT"):
		l.Consume()
		l.ExpectValue("OUTER")
		joinType = sqlstmt.JoinKindLeft
	case l.Current().IsKeyword("RIGHT"):
		l.Consume()
		l.ExpectValue("OUTER")
		joinType = sqlstmt.JoinKindRight
	case l.Current().IsKeyword("FULL"):
		l.Consume()
		l.ExpectValue("OUTER")
		joinType = sqlstmt.JoinKindFull
	case l.Current().IsKeyword("CROSS"):
		l.Consume()
		joinType = sqlstmt.JoinKindCross
	case l.Current().IsKeyword("NATURAL"):
		// NATURAL JOIN 无 ON/USING 条件（按同名列自动连接），仅标记连接类型；
		// IsJoinStart 已含 NATURAL（跨方言共用），此处必须消费，否则解析器停在 NATURAL
		// 使整个 JOIN 连同其后的 WHERE/ORDER 一起被丢弃
		l.Consume()
		joinType = sqlstmt.JoinKindNatural
	case l.Current().IsKeyword("INNER"):
		l.Consume()
	}

	if !joined {
		if !l.Current().IsKeyword("JOIN") {
			l.Pos = start
			return nil
		}
		l.Consume()
	}

	tableRef := l.parseTableRef()
	if tableRef.Name == "" {
		l.Pos = start
		return nil
	}

	var onExpr *sqlstmt.Expr
	if l.Current().IsKeyword("ON") {
		l.Consume()
		onExpr = l.ParseCondExpr(true)
	} else if l.Current().IsKeyword("USING") {
		// USING (c1, c2) 按同名列连接：条件列不建模（无 ON 表达式），仅需消费括号，防止其泄漏为 FROM 表
		l.Consume()
		if l.Current().Value == "(" {
			l.SkipParentheses()
		}
	}

	return &sqlstmt.JoinClause{
		Kind:  joinType,
		Table: tableRef,
		On:    onExpr,
		Text:  l.TextFromExclusive(start),
	}
}

// parseFromTables 解析 FROM 后的表引用列表（逗号分隔），遇 JOIN 起始或子句边界结束。
// JOIN 由 parseJoinClauses 单独解析，使「逗号交叉表」与「显式 JOIN」在任意先后顺序下都能被吃下
func (l *Lexer) parseFromTables() []sqlstmt.TableRef {
	var tables []sqlstmt.TableRef
	for !l.Current().IsEOF() && !l.IsFromClauseEnd() {
		if l.Current().Value == "," {
			l.Consume()
			continue
		}
		if l.IsJoinStart() || l.Current().IsKeyword("JOIN") {
			break
		}
		// 表提示：T-SQL 的 WITH (NOLOCK|UPDLOCK)、mysql 的 FORCE/IGNORE USE INDEX(...)。
		// 提示不建模（不改变被作用的行列集合），但必须消费，否则它会被当成 FROM 的第二张表
		if l.Current().IsKeyword("WITH", "USE", "FORCE", "IGNORE") && l.Peek(1).Value == "(" {
			l.Consume()
			l.SkipParentheses()
			continue
		}
		// 防御：解析无进展时退出，避免异常输入导致死循环
		before := l.Pos
		ref := l.parseTableRef()
		if l.Pos == before {
			break
		}
		if ref.Name != "" {
			tables = append(tables, ref)
		}
	}
	return tables
}

// parseJoinClauses 解析连续的 JOIN 子句链
func (l *Lexer) parseJoinClauses() []sqlstmt.JoinClause {
	var joins []sqlstmt.JoinClause
	for l.IsJoinStart() || l.Current().IsKeyword("JOIN") {
		join := l.parseJoinClause()
		if join == nil {
			break
		}
		joins = append(joins, *join)
	}
	return joins
}

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
	// PG 的 `FROM ONLY t`：ONLY 是修饰词（只查父表不含分区/子表），不是表名。
	// 不消费它会把 ONLY 当成表、真正的 t 反而丢失。
	if l.wordIs("ONLY") {
		l.Consume()
	}
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
	// 别名之后仍可能跟索引提示（JOIN t2 USE INDEX (idx) ON ...），一并消费
	l.skipTableHints()
	return ref
}

// skipTableHints 消费表级提示（不建模）：T-SQL 的 WITH (NOLOCK|UPDLOCK)、
// MySQL 的 USE|FORCE|IGNORE INDEX|KEY [FOR JOIN|ORDER BY|GROUP BY] (列清单)。
//
// 返回是否消费过提示。MySQL 的提示是 `USE INDEX (...)` 而非 `USE (...)`：
// 只认「关键字后紧跟 (」时，INDEX 一词会被当作下一张表收进 FROM。
func (l *Lexer) skipTableHints() bool {
	consumed := false
	for {
		if l.wordIs("TABLESAMPLE") {
			// PG：TABLESAMPLE <method> ( <args> ) [REPEATABLE ( <seed> )]，不改变被作用的表
			l.Consume()
			l.Consume() // 采样方法名
			if l.Current().Value == "(" {
				l.SkipParentheses()
			}
			if l.wordIs("REPEATABLE") {
				l.Consume()
				if l.Current().Value == "(" {
					l.SkipParentheses()
				}
			}
			consumed = true
			continue
		}
		if l.wordIs("FINAL") {
			// ClickHouse FINAL：强制合并去重，不改变被作用的表
			l.Consume()
			consumed = true
			continue
		}
		if l.wordIs("WITH") && l.Peek(1).Value == "(" {
			l.Consume()
			l.SkipParentheses()
			consumed = true
			continue
		}
		if l.wordIs("USE", "FORCE", "IGNORE") && l.wordIsAt(1, "INDEX", "KEY") {
			l.Consume()
			l.Consume()
			if l.Current().IsKeyword("FOR") {
				l.Consume()
				l.Consume()
				if l.Current().IsKeyword("BY") {
					l.Consume()
				}
			}
			if l.Current().Value == "(" {
				l.SkipParentheses()
			}
			consumed = true
			continue
		}
		return consumed
	}
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
	// 这些位置性词只在特定形态下不是别名（按结构判据，不能无条件排除：
	// `INSERT ... IGNORE INTO` 里的 IGNORE 在别处仍可能是别名位置）：
	//   - MySQL 索引提示 `USE|FORCE|IGNORE INDEX|KEY (...)`：若当别名，INDEX 会变成下一张表；
	//   - T-SQL 表提示 `WITH (...)`；
	//   - ClickHouse 的 `FINAL` 修饰。
	if (l.wordIs("USE", "FORCE", "IGNORE") && l.wordIsAt(1, "INDEX", "KEY")) ||
		(l.wordIs("WITH") && l.Peek(1).Value == "(") ||
		l.wordIs("FINAL") ||
		// PG TABLESAMPLE <method>(...)：采样修饰；不排除会把 SYSTEM 当第二张表
		l.wordIs("TABLESAMPLE") ||
		// ClickHouse SETTINGS / ARRAY JOIN：查询级修饰与数组连接子句
		l.wordIs("SETTINGS") ||
		(l.wordIs("ARRAY") && l.wordIsAt(1, "JOIN")) {
		return ""
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
		// T-SQL CROSS APPLY：右侧是表值函数/相关子查询，同样是被连接的数据源
		if l.wordIs("APPLY") {
			l.Consume()
			joined = true
			// CROSS APPLY fn(x)：右侧可能是表值函数而非表——函数名不能进表清单
			// （表级 ACL 会在函数名上误命中）。括号内是参数，整段消费即可。
			if l.IsUsableName() && l.Peek(1).Value == "(" {
				l.Consume()
				l.SkipParentheses()
				l.parseOptionalTableAlias()
				return &sqlstmt.JoinClause{Kind: joinType, Table: sqlstmt.TableRef{}}
			}
		}
	case l.isArrayJoin():
		// ClickHouse ARRAY JOIN：右侧是数组列/表达式，不是表——整段消费但不登记表引用
		l.skipArrayJoinClause()
		return nil
	case l.wordIs("OUTER") && l.wordIsAt(1, "APPLY"):
		// T-SQL OUTER APPLY：右侧数据源保留（表要进清单），语义同 LEFT JOIN
		l.Consume()
		l.Consume()
		joinType = sqlstmt.JoinKindLeft
		joined = true
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
		if l.IsJoinStart() || l.Current().IsKeyword("JOIN") || l.isArrayJoin() {
			break
		}
		// 表提示：T-SQL 的 WITH (NOLOCK|UPDLOCK)、MySQL 的 USE|FORCE|IGNORE INDEX|KEY (...)。
		// 提示不建模（不改变被作用的行列集合），但必须消费，否则 INDEX 一词会被当成 FROM 的第二张表，
		// 让表级 ACL/策略在错误的表上命中。
		if l.skipTableHints() {
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
	for l.IsJoinStart() || l.Current().IsKeyword("JOIN") || l.isArrayJoin() {
		join := l.parseJoinClause()
		if join == nil {
			break
		}
		joins = append(joins, *join)
	}
	return joins
}

// skipArrayJoinClause 消费 ClickHouse 的 `ARRAY JOIN <expr>[, <expr>]`：
// 右侧是数组列/表达式而非表，只消费不建模，避免数组列名进入表清单。
func (l *Lexer) skipArrayJoinClause() {
	l.Consume() // ARRAY
	l.Consume() // JOIN
	for !l.Current().IsEOF() && !l.IsFromClauseEnd() && l.Current().Value != ";" {
		if l.Current().Value == "(" {
			l.SkipParentheses()
			continue
		}
		l.Consume()
	}
}

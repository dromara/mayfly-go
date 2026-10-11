package base

// SELECT 语句骨架：投影、FROM/JOIN、WHERE、GROUP BY/HAVING、层次查询、ORDER BY、
// 集合操作、行限制、锁与 RETURNING 的统一解析实现。
//
// 与 base/stmt_skeleton.go 同一分层取向：语句级共享骨架住在 base、挂在 *Lexer 上，
// 方言只做分发与真正方言特有的子句（INSERT 冲突子句、mysql 多表 UPDATE/DELETE）。
//
// 扩展方式（开闭原则）：
//   - 词法差异（某方言认得哪些引用符、哪些词是它的关键字）→ 登记进 tokenizer.DialectConfig；
//   - 语法差异（某方言多一种子句）→ 在本骨架的对应位置新增一个消费函数，或在方言侧包一层
//     （先调骨架再消费自己的尾部）；两种都不需要改动既有方言的任何代码。
//   - 新增方言 → 只需 config + 分发，不拷贝骨架。

import (
	"strings"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// parseSelectStatement 解析一条完整 SELECT：外层括号、集合操作、语句级 ORDER BY 与全部尾部子句。
//
// 第二个返回值表示是否解析出了 SELECT 主体：括号内不是 SELECT 起始（如 `(WITH ...)`、`(CALL ...)`）
// 时为 false，调用方可据此改走自己的通用语句解析。
func (l *Lexer) parseSelectStatement() (*sqlstmt.SelectStmt, bool) {
	start := l.Pos
	var sel *sqlstmt.SelectStmt

	if l.Current().Value == "(" {
		l.Consume()
		if l.Current().IsKeyword("SELECT") {
			sel = l.parseSelectBody()
		} else {
			// 括号内非 SELECT：整体跳过并按「是一条查询语句」的形态回填文本
			// （(WITH ...) 之类的合法查询主体仍由消费方按原文处理，骨架不做猜测式建模）
			l.SkipParentheses()
			sel = &sqlstmt.SelectStmt{}
		}
		l.ExpectValue(")")
	} else if l.Current().IsKeyword("SELECT") {
		sel = l.parseSelectBody()
	} else {
		return nil, false
	}

	// 集合操作（UNION/INTERSECT/EXCEPT/MINUS，具体接受哪些由该方言的关键字表决定）
	l.parseSetOperations(sel)

	// 集合操作之后的全局 ORDER BY（子句体内已解析出排序时不覆盖）
	if len(sel.OrderBy) == 0 && l.Current().IsKeyword("ORDER") {
		sel.OrderBy = l.parseOrderByClause()
	}

	// 全局行限制（覆盖子句体内的同形态解析结果，仅在确实解析到新子句时覆盖）
	if limit := l.parseRowLimitTail(); limit != nil {
		sel.Limit = limit
	}

	// mysql 允许 INTO 出现在 FROM/WHERE/ORDER/LIMIT 之后（SELECT ... INTO OUTFILE），
	// 与子句体位置各调用一次，两处都无需消费时是空操作
	l.parseIntoClause()

	l.parseLocking()
	l.ParseReturning()

	sel.Base = sqlstmt.Base{Text: l.TextFrom(start)}
	return sel, true
}

// parseSelectBody 解析 SELECT 子句体（不含外层括号与集合操作）
func (l *Lexer) parseSelectBody() *sqlstmt.SelectStmt {
	start := l.Pos
	l.Consume() // SELECT（调用方已判定）

	sel := &sqlstmt.SelectStmt{}
	if l.Current().IsKeyword("DISTINCT") {
		l.Consume()
		sel.Distinct = true
	} else if l.Current().IsKeyword("ALL") {
		// SELECT ALL 是默认形态（Oracle/达梦/mysql），消费掉即可，不改变 Distinct
		l.Consume()
	}
	// TOP n 只可能出现在 SELECT [ALL|DISTINCT] 之后，此处解析不影响任何裸列名
	sel.Limit = l.parseTopLimit()

	sel.Items = l.parseSelectItems()

	l.parseIntoClause()

	if l.Current().IsKeyword("FROM") {
		l.Consume()
		sel.From = l.parseFromTables()
		sel.Joins = l.parseJoinClauses()
	}

	if l.Current().IsKeyword("WHERE") {
		l.Consume()
		sel.Where = l.ParseCondExpr(false)
		// 层次子句在 WHERE 之后（Oracle 文档语序）
		l.parseHierarchic()
	}

	if l.Current().IsKeyword("GROUP") {
		l.Consume()
		l.ExpectValue("BY")
		// 分组表达式不建模，回填原始文本标记其存在，供调用方判定语句形态
		if text := l.CaptureSkip(l.SkipGroupByExpr); text != "" {
			sel.GroupBy = append(sel.GroupBy, text)
		}
	}

	if l.Current().IsKeyword("HAVING") {
		l.Consume()
		if having := l.ParseCondExpr(false); having.Text != "" {
			sel.Having = having
		}
	}

	// 层次子句也可能出现在 HAVING 之后（两种语序都存在，出现一次即消费）
	l.parseHierarchic()

	if l.Current().IsKeyword("ORDER") {
		sel.OrderBy = l.parseOrderByClause()
	}

	// 子句体内的行限制（派生表/集合操作分支自带分页时需要；语句级会再次覆盖）
	if limit := l.parseRowLimitTail(); limit != nil {
		sel.Limit = limit
	}

	sel.Base = sqlstmt.Base{Text: l.TextFrom(start)}
	return sel
}

// parseSelectItems 解析投影列表。
// 项类型与表限定符统一由 ClassifySelectItem 判定，保证跨方言 Kind 语义一致；
// 星号项不回填 ColumnName（Kind 已表达「全部列」，`*` 当列名传给下游只会成为噪声）。
// 项文本保留原文（含引用符）：Text 的契约就是原始文本，去引用后的名字在 ColumnName/Alias。
func (l *Lexer) parseSelectItems() []sqlstmt.SelectItem {
	var items []sqlstmt.SelectItem
	for !l.Current().IsEOF() && !l.IsSelectClauseEnd() {
		if l.Current().Value == "," {
			l.Consume()
			continue
		}
		itemStart := l.Pos
		l.SkipSelectItem()
		if l.Pos == itemStart {
			// 无进展：当前 token 是子句边界关键字（畸形 SQL），退出防死循环
			break
		}
		text := TrimTrailingComma(l.TextFromExclusive(itemStart))
		// 别名提取与项分类都用原文：整体 Unquote 会把 `u`.`phone` 破坏为 u`.`phone，污染限定符分段
		col, alias := l.ExtractColumnAndAlias(text)
		// 分类必须用原文别名：ClassifySelectItem 靠「文本是否以该别名结尾」剥离别名
		kind, tableAlias := l.ClassifySelectItem(text, alias)
		if kind == sqlstmt.SelectItemStar {
			col = ""
		}
		// 存储去引用符后的名字，与 ColumnName 对称：结果集列名是服务端返回的裸名，
		// 别名带引号（`cnt` / "CNT"）会让按别名匹配的下游（脱敏血缘兜底映射）对不上
		alias = l.Unquote(alias)
		items = append(items, sqlstmt.SelectItem{
			Kind:       kind,
			Text:       text,
			TableAlias: tableAlias,
			ColumnName: col,
			Alias:      alias,
		})
	}
	return items
}

// parseOrderByClause 消费 ORDER 子句头部（含 Oracle 层次查询的 ORDER SIBLINGS BY）并解析排序项
func (l *Lexer) parseOrderByClause() []sqlstmt.OrderByItem {
	l.Consume() // ORDER
	if l.IsWord("SIBLINGS") {
		l.Consume()
	}
	l.ExpectValue("BY")
	return l.parseOrderBy()
}

// parseOrderBy 解析排序项列表（ASC/DESC 归一为 Desc 标志，其余保留原文）
func (l *Lexer) parseOrderBy() []sqlstmt.OrderByItem {
	var items []sqlstmt.OrderByItem
	for !l.Current().IsEOF() && !l.IsExprEnd() {
		if l.Current().Value == "," {
			l.Consume()
			continue
		}
		start := l.Pos
		l.SkipExpr()
		text := TrimTrailingComma(l.TextFromExclusive(start))
		if text == "" {
			break // 无进展防御
		}

		desc := false
		upper := strings.ToUpper(text)
		if strings.HasSuffix(upper, " DESC") {
			desc = true
			text = strings.TrimSpace(text[:len(text)-5])
		} else if strings.HasSuffix(upper, " ASC") {
			text = strings.TrimSpace(text[:len(text)-4])
		}
		items = append(items, sqlstmt.OrderByItem{Text: text, Desc: desc})
	}
	return items
}

// parseSetOperations 解析集合操作链，并把最后一个分支的全局 ORDER BY / 行限制上提到整条语句
// （UNION 之后的排序与分页作用于合并结果，属于整条语句而非某个分支）。
func (l *Lexer) parseSetOperations(sel *sqlstmt.SelectStmt) {
	for l.Current().IsKeyword(setOpKeywords...) {
		op := strings.ToUpper(l.Current().Value)
		start := l.Pos
		l.Consume()

		all := false
		if l.Current().IsKeyword("ALL") {
			l.Consume()
			all = true
		} else if l.Current().IsKeyword("DISTINCT") {
			l.Consume()
		}

		var next *sqlstmt.SelectStmt
		if l.Current().Value == "(" {
			parenStart := l.Pos
			l.Consume()
			if l.Current().IsKeyword("SELECT") {
				next = l.parseSelectBody()
			} else {
				l.SkipParentheses()
			}
			l.ExpectValue(")")
			if next != nil {
				// 括号形态的操作数：文本按括号span回填，避免子句体在 `)` 处结束时多带闭合括号
				next.Base = sqlstmt.Base{Text: l.TextFromExclusive(parenStart)}
			}
		} else if l.Current().IsKeyword("SELECT") {
			next = l.parseSelectBody()
		}
		if next == nil {
			continue // 操作数不是 SELECT（畸形 SQL）：已消费的部分保留，不挂一个空分支
		}

		if len(next.OrderBy) > 0 {
			sel.OrderBy = next.OrderBy
			next.OrderBy = nil
		}
		if next.Limit != nil {
			sel.Limit = next.Limit
			next.Limit = nil
		}
		sel.Unions = append(sel.Unions, sqlstmt.UnionClause{
			Op:     op,
			Select: next,
			All:    all,
			Text:   l.TextFromExclusive(start),
		})
	}
}

// isHierarchicHead 是否处于层次查询子句头部（START WITH / CONNECT BY）。
//
// 单一判定点，三处共用：骨架消费层次子句、名称位拒绝吞掉子句头部词、条件扫描在此收口。
// 只在声明了层次查询能力的方言（Oracle 系）生效：START 不是保留字，
// 若无条件地按词值配对，T-SQL 的 `FROM start WITH (NOLOCK)`（start 作表名）会被当成层次子句头而截断 FROM
func (l *Lexer) isHierarchicHead() bool {
	return l.isHierarchicHeadAt(0)
}

// isHierarchicHeadAt 判断相对当前位置偏移 offset 起是否为层次子句头部词对
func (l *Lexer) isHierarchicHeadAt(offset int) bool {
	if !l.config.Hierarchic {
		return false
	}
	return l.isWordPairAt(offset, "START", "WITH") || l.isWordPairAt(offset, "CONNECT", "BY")
}

// parseHierarchic 解析层次查询子句（Oracle/达梦的 START WITH 与 CONNECT BY，两种先后顺序皆可，
// 且 Oracle 允许同一语句多次交替出现）。条件表达式本身不建模——消费方只需知道「有层次条件」这一
// 形态并保留原文；未消费会导致其后的 WHERE/ORDER 整段被丢弃。
func (l *Lexer) parseHierarchic() {
	for {
		switch {
		case l.isWordPair("START", "WITH"):
			l.Consume()
			l.Consume()
			l.skipHierarchicCond()
		case l.isWordPair("CONNECT", "BY"):
			l.Consume()
			l.Consume()
			// CONNECT BY NOCYCLE 形态
			if l.IsWord("NOCYCLE") {
				l.Consume()
			}
			l.skipHierarchicCond()
		default:
			return
		}
	}
}

// skipHierarchicCond 跳过层次查询条件：在下一个 START WITH / CONNECT BY 头部
// 或常规子句边界处结束（IsExprEnd 不含这两个词，故单独判定）
func (l *Lexer) skipHierarchicCond() {
	for !l.Current().IsEOF() {
		if l.Current().Value == "(" {
			l.SkipParentheses()
			continue
		}
		if l.IsExprEnd() || l.isHierarchicHead() {
			return
		}
		l.Consume()
	}
}

// parseIntoClause 消费 SELECT ... INTO 目标（SQL Server 的 SELECT INTO 建表、
// PostgreSQL 的 SELECT INTO、mysql 的 INTO OUTFILE/DUMPFILE/INTO @var）。
//
// 目标不建模（它是写出目的地，不参与行列集合判定），但必须消费：
// 子句体位置不消费会让紧随的 FROM/WHERE 整段丢失，尾部位置不消费则整个导出子句失联
func (l *Lexer) parseIntoClause() {
	if !l.Current().IsKeyword("INTO") {
		return
	}
	l.Consume()
	l.SkipExpr()
}

// parseLocking 解析行锁子句：FOR UPDATE / FOR SHARE 及其修饰（OF 引用、NOWAIT、WAIT n、SKIP LOCKED）。
//
// 锁修饰词在 Oracle 系不是保留字，故只在「FOR UPDATE/SHARE 之后」这一无歧义位置按单词值匹配。
// 锁模式不建模（消费方不区分），但必须整段消费：残留的 NOWAIT/SKIP LOCKED 会让其后的
// RETURNING 等子句被当成未解析文本，也会让语句原文截断使下游按文本再解析时失败。
func (l *Lexer) parseLocking() {
	if !l.Current().IsKeyword("FOR") {
		return
	}
	if !wordAt(l, 1, "UPDATE") && !wordAt(l, 1, "SHARE") {
		return
	}
	l.Consume() // FOR
	l.Consume() // UPDATE | SHARE

	if l.IsWord("OF") {
		l.Consume()
		l.SkipExpr() // OF 后是被锁定的表/列引用列表
	}
	for {
		switch {
		case l.IsWord("NOWAIT"):
			l.Consume()
		case l.IsWord("WAIT"):
			l.Consume()
			if isNumberToken(l.Current()) {
				l.Consume()
			}
		case l.IsWord("SKIP") && wordAt(l, 1, "LOCKED"):
			l.Consume()
			l.Consume()
		default:
			return
		}
	}
}

// ParseReturning 解析 RETURNING 子句（pg / oracle / SQLite 的 DML 返回值形态）。
// 返回列不建模：它是执行侧的输出契约，与采集/准入决策无关，但必须消费到语句末尾。
func (l *Lexer) ParseReturning() {
	if !l.Current().IsKeyword("RETURNING") {
		return
	}
	l.Consume()
	l.SkipToStatementEnd()
}

// SkipToStatementEnd 消费到本条语句结束（分号前或 EOF），分号本身留给调用方/下一条语句
func (l *Lexer) SkipToStatementEnd() {
	for !l.Current().IsEOF() && l.Current().Value != ";" {
		l.Consume()
	}
}

package oracle

import (
	"strings"

	"mayfly-go/internal/db/dbm/sqlparser/base"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// Parser Oracle 方言 SQL 解析器
type Parser struct {
	*base.Lexer
}

// NewParser 创建 Oracle 解析器
func NewParser(sql string) *Parser {
	p := &Parser{Lexer: base.NewLexer(sql, tokenizer.DialectConfig{
		DoubleQuoteAsIdentifier: true,
	})}
	// 注入子查询解析回调：条件表达式中的(SELECT...)可建SelectStmt树
	p.SelectParser = func() (*sqlstmt.SelectStmt, bool) {
		st := p.parseSelect()
		sel, ok := st.(*sqlstmt.SelectStmt)
		return sel, ok
	}
	return p
}

// Parse 解析单条 SQL 语句
func (p *Parser) Parse() (sqlstmt.Stmt, error) {
	p.SkipSemicolons()
	if p.Current().IsEOF() {
		return nil, nil
	}
	stmt := p.parseStatement()
	return stmt, nil
}

func (p *Parser) parseStatement() sqlstmt.Stmt {
	tok := p.Current()
	switch {
	case tok.IsKeyword("SELECT") || tok.Value == "(":
		return p.parseSelect()
	case tok.IsKeyword("INSERT"):
		return p.parseInsert()
	case tok.IsKeyword("UPDATE"):
		return p.parseUpdate()
	case tok.IsKeyword("DELETE"):
		return p.parseDelete()
	case tok.IsKeyword("CREATE"):
		return p.ParseCreateStmt()
	case tok.IsKeyword("DROP"):
		return p.ParseDropStmt()
	case tok.IsKeyword("ALTER"):
		return p.ParseAlterStmt()
	case tok.IsKeyword("WITH"):
		return p.ParseWithStmt()
	case tok.IsKeyword("TRUNCATE"):
		return p.parseGenericDdl()
	case tok.IsKeyword("COMMENT", "GRANT", "REVOKE", "RENAME", "ANALYZE"):
		// Oracle 的 COMMENT ON、GRANT、REVOKE 等均为非查询语句，统一按 DDL 执行
		return p.parseGenericDdl()
	default:
		return p.parseGenericStmt()
	}
}

// ---------- SELECT 解析 ----------

func (p *Parser) parseSelect() sqlstmt.Stmt {
	start := p.Pos
	var selectStmt *sqlstmt.SelectStmt

	if p.Current().Value == "(" {
		p.Consume()
		if p.Current().IsKeyword("SELECT") {
			selectStmt = p.parseSelectBody()
		} else {
			p.SkipParentheses()
			selectStmt = &sqlstmt.SelectStmt{}
		}
		p.ExpectValue(")")
	} else {
		selectStmt = p.parseSelectBody()
	}

	// UNION 解析
	selectStmt = p.parseUnions(selectStmt)

	// ORDER BY（UNION 之后的 ORDER BY）
	if len(selectStmt.OrderBy) == 0 && p.Current().IsKeyword("ORDER") {
		p.Consume()
		if p.Current().IsKeyword("BY") {
			p.Consume()
		}
		selectStmt.OrderBy = p.parseOrderBy()
	}

	// Oracle 12c+ 的 OFFSET n [ROW|ROWS] 分页偏移（可单独使用或与 FETCH FIRST/NEXT 连用）
	offsetText := ""
	if p.Current().IsKeyword("OFFSET") {
		offsetStart := p.Pos
		p.Consume()
		if p.Current().Type == tokenizer.TokenNumber {
			p.Consume()
		}
		if p.Current().IsKeyword("ROW", "ROWS") {
			p.Consume()
		}
		offsetText = p.TextFrom(offsetStart)
		// 单独使用 OFFSET 时先生成 LIMIT，若后续有 FETCH 子句则由 parseFetchFirst 合并覆盖
		selectStmt.Limit = &sqlstmt.Limit{Text: strings.TrimSpace(offsetText)}
	}

	// Oracle 特有的 FETCH FIRST（12c+）
	if p.Current().IsKeyword("FETCH") {
		p.parseFetchFirst(selectStmt, offsetText)
	}

	// FOR UPDATE
	if p.Current().IsKeyword("FOR") {
		p.Consume()
		if p.Current().IsKeyword("UPDATE") {
			p.Consume()
			// OF column
			if p.Current().IsKeyword("OF") {
				p.Consume()
				p.SkipExpr()
			}
			// NOWAIT / WAIT n / SKIP LOCKED
			if p.Current().IsKeyword("NOWAIT") {
				p.Consume()
			} else if p.Current().IsKeyword("WAIT") {
				p.Consume()
				if p.Current().Type == tokenizer.TokenNumber {
					p.Consume()
				}
			} else if p.Current().IsKeyword("SKIP") {
				p.Consume()
				if p.Current().IsKeyword("LOCKED") {
					p.Consume()
				}
			}
		}
	}

	// 更新完整文本
	selectStmt.Base = sqlstmt.Base{Text: p.TextFrom(start)}
	return selectStmt
}

func (p *Parser) parseSelectBody() *sqlstmt.SelectStmt {
	start := p.Pos

	distinct := false
	if p.Current().IsKeyword("SELECT") {
		p.Consume()
		if p.Current().IsKeyword("DISTINCT") {
			p.Consume()
			distinct = true
		} else if p.Current().IsKeyword("ALL") {
			p.Consume()
		}
	}

	// SELECT 项
	var items []sqlstmt.SelectItem
	for !p.Current().IsEOF() && !p.IsSelectClauseEnd() {
		if p.Current().Value == "," {
			p.Consume()
			continue
		}
		itemStart := p.Pos
		p.SkipSelectItem()
		if p.Pos == itemStart {
			// SkipExpr无进展：当前token为子句边界关键字（畸形SQL），退出防死循环
			break
		}
		text := base.TrimTrailingComma(p.TextFromExclusive(itemStart))
		col, alias := p.ExtractColumnAndAlias(text)
		// 项类型与表限定符统一由base分类（跨方言Kind语义一致）
		kind, tableAlias := p.ClassifySelectItem(text, alias)
		items = append(items, sqlstmt.SelectItem{
			Kind:       kind,
			Text:       text,
			TableAlias: tableAlias,
			ColumnName: col,
			Alias:      alias,
		})
	}

	selectStmt := &sqlstmt.SelectStmt{
		Base:     sqlstmt.Base{Text: p.TextFrom(start)},
		Distinct: distinct,
		Items:    items,
	}

	// FROM
	if p.Current().IsKeyword("FROM") {
		p.Consume()
		selectStmt.From = p.parseFromClause()
	}

	// JOIN（起始token解析不出合法JOIN时break，防止Pos重置导致死循环）
	for p.IsJoinStart() || p.Current().IsKeyword("JOIN") {
		join := p.parseJoinClause()
		if join == nil {
			break
		}
		selectStmt.Joins = append(selectStmt.Joins, *join)
	}

	// WHERE（Oracle 可能包含 ROWNUM 条件）
	if p.Current().IsKeyword("WHERE") {
		p.Consume()
		selectStmt.Where = p.ParseCondExpr(false)
	}

	// GROUP BY
	if p.Current().IsKeyword("GROUP") {
		p.Consume()
		if p.Current().IsKeyword("BY") {
			p.Consume()
		}
		// 表达式不建模，回填原始文本标记其存在，供调用方判定语句形态
		if text := p.CaptureSkip(p.SkipGroupByExpr); text != "" {
			selectStmt.GroupBy = append(selectStmt.GroupBy, text)
		}
	}

	// HAVING
	if p.Current().IsKeyword("HAVING") {
		p.Consume()
		if having := p.ParseCondExpr(false); having.Text != "" {
			selectStmt.Having = having
		}
	}

	// CONNECT BY（Oracle 层次查询）
	if p.Current().IsKeyword("CONNECT") {
		p.Consume()
		if p.Current().IsKeyword("BY") {
			p.Consume()
		}
		// 跳过 CONNECT BY 条件
		p.SkipExpr()

		// START WITH（可选）
		if p.Current().IsKeyword("START") {
			p.Consume()
			if p.Current().IsKeyword("WITH") {
				p.Consume()
				p.SkipExpr()
			}
		}
	}

	// ORDER BY
	if p.Current().IsKeyword("ORDER") {
		p.Consume()
		if p.Current().IsKeyword("BY") {
			p.Consume()
		}
		selectStmt.OrderBy = p.parseOrderBy()
	}

	return selectStmt
}

func (p *Parser) parseUnions(selectStmt *sqlstmt.SelectStmt) *sqlstmt.SelectStmt {
	for p.Current().IsKeyword("UNION", "INTERSECT", "EXCEPT", "MINUS") {
		op := strings.ToUpper(p.Current().Value)
		p.Consume()
		all := false
		if p.Current().IsKeyword("ALL") {
			p.Consume()
			all = true
		} else if p.Current().IsKeyword("DISTINCT") {
			p.Consume()
		}

		var nextSelect *sqlstmt.SelectStmt
		if p.Current().Value == "(" {
			p.Consume()
			if p.Current().IsKeyword("SELECT") {
				nextSelect = p.parseSelectBody()
			}
			p.ExpectValue(")")
		} else if p.Current().IsKeyword("SELECT") {
			nextSelect = p.parseSelectBody()
		}

		if nextSelect != nil {
			// 提取最后一个 unionSelect 的 ORDER BY 和 LIMIT 到外层
			if len(nextSelect.OrderBy) > 0 {
				selectStmt.OrderBy = nextSelect.OrderBy
				nextSelect.OrderBy = nil
			}
			selectStmt.Unions = append(selectStmt.Unions, sqlstmt.UnionClause{
				Op:     op,
				Select: nextSelect,
				All:    all,
			})
		}
	}
	return selectStmt
}

// parseFetchFirst 解析 Oracle 12c+ 的 FETCH FIRST n ROWS ONLY
// offsetText 为前置的 OFFSET 子句文本（可为空），非空时会合并到 LIMIT 文本中
func (p *Parser) parseFetchFirst(stmt *sqlstmt.SelectStmt, offsetText string) {
	if !p.Current().IsKeyword("FETCH") {
		return
	}

	start := p.Pos
	p.Consume() // FETCH

	// FIRST 或 NEXT
	if p.Current().IsKeyword("FIRST") || p.Current().IsKeyword("NEXT") {
		p.Consume()
	}

	// 数量
	if p.Current().Type == tokenizer.TokenNumber {
		p.Consume()
	}

	// ROW 或 ROWS
	if p.Current().IsKeyword("ROW") || p.Current().IsKeyword("ROWS") {
		p.Consume()
	}

	// ONLY 或 WITH TIES
	if p.Current().IsKeyword("ONLY") {
		p.Consume()
	} else if p.Current().IsKeyword("WITH") {
		p.Consume()
		if p.Current().IsKeyword("TIES") {
			p.Consume()
		}
	}

	// 创建伪 LIMIT（若存在前置 OFFSET 子句，则合并文本）
	text := p.TextFrom(start)
	if offsetText != "" {
		text = strings.TrimSpace(offsetText + " " + text)
	}
	stmt.Limit = &sqlstmt.Limit{
		Text: text,
	}
}

func (p *Parser) parseOrderBy() []sqlstmt.OrderByItem {
	var items []sqlstmt.OrderByItem
	for !p.Current().IsEOF() && !p.IsExprEnd() {
		if p.Current().Value == "," {
			p.Consume()
			continue
		}
		start := p.Pos
		p.SkipExpr()
		text := p.TextFromExclusive(start)

		desc := false
		upper := strings.ToUpper(text)
		if strings.HasSuffix(upper, " DESC") {
			desc = true
			text = strings.TrimSpace(text[:len(text)-5])
		} else if strings.HasSuffix(upper, " ASC") {
			text = strings.TrimSpace(text[:len(text)-4])
		}

		items = append(items, sqlstmt.OrderByItem{
			Text: text,
			Desc: desc,
		})
	}
	return items
}

// ---------- FROM 解析 ----------

func (p *Parser) parseFromClause() []sqlstmt.TableRef {
	var tables []sqlstmt.TableRef
	for !p.Current().IsEOF() && !p.IsFromClauseEnd() {
		if p.Current().Value == "," {
			p.Consume()
			continue
		}
		if p.IsJoinStart() || p.Current().IsKeyword("JOIN") {
			break
		}
		// 防御：解析无进展时退出，避免异常输入导致死循环
		before := p.Pos
		ref := p.parseTableRef()
		if p.Pos == before {
			break
		}
		if ref.Name != "" {
			tables = append(tables, ref)
		}
	}
	return tables
}

func (p *Parser) parseTableRef() sqlstmt.TableRef {
	start := p.Pos

	// 子查询
	if p.Current().Value == "(" {
		p.Consume()
		if p.Current().IsKeyword("SELECT") {
			p.parseSelectBody()
		} else {
			p.SkipParentheses()
		}
		p.ExpectValue(")")

		var alias string
		if p.Current().IsKeyword("AS") {
			p.Consume()
		}
		if p.Current().Type == tokenizer.TokenIdentifier {
			alias = p.Unquote(p.Consume().Value)
		}
		return sqlstmt.TableRef{
			Name:  p.TextFrom(start),
			Alias: alias,
		}
	}

	if p.Current().Type != tokenizer.TokenIdentifier && p.Current().Type != tokenizer.TokenString {
		return sqlstmt.TableRef{}
	}

	ref := sqlstmt.TableRef{}
	part1 := p.Consume().Value

	if p.Current().Value == "." {
		p.Consume()
		if p.Current().Type == tokenizer.TokenIdentifier || p.Current().Type == tokenizer.TokenString {
			part2 := p.Consume().Value
			ref.Schema = p.Unquote(part1)
			ref.Name = p.Unquote(part2)
		} else {
			ref.Name = p.Unquote(part1)
		}
	} else {
		ref.Name = p.Unquote(part1)
	}

	if p.Current().IsKeyword("AS") {
		p.Consume()
	}
	if p.Current().Type == tokenizer.TokenIdentifier {
		ref.Alias = p.Unquote(p.Consume().Value)
	}

	return ref
}

func (p *Parser) parseJoinClause() *sqlstmt.JoinClause {
	start := p.Pos
	joinType := sqlstmt.JoinKindInner
	if p.Current().IsKeyword("LEFT") {
		p.Consume()
		if p.Current().IsKeyword("OUTER") {
			p.Consume()
		}
		joinType = sqlstmt.JoinKindLeft
	} else if p.Current().IsKeyword("RIGHT") {
		p.Consume()
		if p.Current().IsKeyword("OUTER") {
			p.Consume()
		}
		joinType = sqlstmt.JoinKindRight
	} else if p.Current().IsKeyword("FULL") {
		p.Consume()
		if p.Current().IsKeyword("OUTER") {
			p.Consume()
		}
		joinType = sqlstmt.JoinKindFull
	} else if p.Current().IsKeyword("CROSS") {
		p.Consume()
		joinType = sqlstmt.JoinKindCross
	} else if p.Current().IsKeyword("NATURAL") {
		p.Consume()
		joinType = sqlstmt.JoinKindNatural
	} else if p.Current().IsKeyword("INNER") {
		p.Consume()
	}

	if !p.Current().IsKeyword("JOIN") {
		p.Pos = start
		return nil
	}
	p.Consume()

	tableRef := p.parseTableRef()
	if tableRef.Name == "" {
		p.Pos = start
		return nil
	}

	var onExpr *sqlstmt.Expr
	if p.Current().IsKeyword("ON") {
		p.Consume()
		onExpr = p.ParseCondExpr(true)
	} else if p.Current().IsKeyword("USING") {
		p.Consume()
		if p.Current().Value == "(" {
			p.SkipParentheses()
		}
	}

	return &sqlstmt.JoinClause{
		Kind:  joinType,
		Table: tableRef,
		On:    onExpr,
		Text:  p.TextFromExclusive(start),
	}
}

// ---------- INSERT 解析 ----------

func (p *Parser) parseInsert() sqlstmt.Stmt {
	start := p.Pos
	p.Consume() // INSERT

	// INTO（可选）
	if p.Current().IsKeyword("INTO") {
		p.Consume()
	}

	// 使用 parseTableRef 正确解析表名
	tableRef := p.parseTableRef()

	// 解析列名列表
	columns := []string{}
	if p.Current().Value == "(" {
		p.Consume()
		for !p.Current().IsEOF() && p.Current().Value != ")" {
			if p.Current().Value == "," {
				p.Consume()
				continue
			}
			if p.Current().Type == tokenizer.TokenIdentifier {
				columns = append(columns, p.Unquote(p.Consume().Value))
			} else {
				p.Consume()
			}
		}
		if p.Current().Value == ")" {
			p.Consume()
		}
	}

	// VALUES 或 SELECT
	if p.Current().IsKeyword("VALUES") {
		p.Consume()
		for p.Current().Value == "(" {
			p.SkipParentheses()
			if p.Current().Value == "," {
				p.Consume()
			}
		}
	} else if p.Current().IsKeyword("SELECT") {
		p.parseSelect()
	}

	// RETURNING（Oracle 支持）
	if p.Current().IsKeyword("RETURNING") {
		p.Consume()
		for !p.Current().IsEOF() && p.Current().Value != ";" {
			p.Consume()
		}
	}

	return &sqlstmt.InsertStmt{
		Base:    sqlstmt.Base{Text: p.TextFrom(start)},
		Table:   tableRef,
		Columns: columns,
	}
}

// ---------- UPDATE 解析 ----------

func (p *Parser) parseUpdate() sqlstmt.Stmt {
	start := p.Pos
	p.Consume() // UPDATE

	// 使用 parseTableRef 正确解析表名
	tableRef := p.parseTableRef()
	tables := []sqlstmt.TableRef{tableRef}

	// SET - 解析字段赋值
	assignments := make([]sqlstmt.Assignment, 0)
	if p.Current().IsKeyword("SET") {
		p.Consume()
		for !p.Current().IsEOF() {
			if p.Current().IsKeyword("WHERE") || p.Current().Value == ";" {
				break
			}
			assign := p.parseAssignment()
			if assign != nil {
				assignments = append(assignments, *assign)
			}
			if p.Current().Value == "," {
				p.Consume()
				continue
			}
			if p.Current().IsKeyword("WHERE") || p.Current().Value == ";" {
				break
			}
		}
	}

	// WHERE
	var where *sqlstmt.Expr
	if p.Current().IsKeyword("WHERE") {
		p.Consume()
		where = p.ParseCondExpr(false)
	}

	// RETURNING（Oracle 支持）
	if p.Current().IsKeyword("RETURNING") {
		p.Consume()
		for !p.Current().IsEOF() && p.Current().Value != ";" {
			p.Consume()
		}
	}

	return &sqlstmt.UpdateStmt{
		Base:   sqlstmt.Base{Text: p.TextFrom(start)},
		Tables: tables,
		Set:    assignments,
		Where:  where,
	}
}

// ---------- DELETE 解析 ----------

func (p *Parser) parseDelete() sqlstmt.Stmt {
	start := p.Pos
	p.Consume() // DELETE

	if p.Current().IsKeyword("FROM") {
		p.Consume()
	}

	// 使用 parseTableRef 正确解析表名
	tableRef := p.parseTableRef()
	tables := []sqlstmt.TableRef{tableRef}

	// WHERE
	var where *sqlstmt.Expr
	if p.Current().IsKeyword("WHERE") {
		p.Consume()
		where = p.ParseCondExpr(false)
	}

	// RETURNING（Oracle 支持）
	if p.Current().IsKeyword("RETURNING") {
		p.Consume()
		for !p.Current().IsEOF() && p.Current().Value != ";" {
			p.Consume()
		}
	}

	return &sqlstmt.DeleteStmt{
		Base:   sqlstmt.Base{Text: p.TextFrom(start)},
		Tables: tables,
		Where:  where,
	}
}

// ---------- DDL 解析 ----------

func (p *Parser) parseGenericDdl() sqlstmt.Stmt {
	start := p.Pos
	p.SkipToNextStatement()
	return &sqlstmt.DdlStmt{
		Base:    sqlstmt.Base{Text: p.TextFrom(start)},
		DdlKind: "DDL",
	}
}

// ---------- WITH 解析 ----------

// ---------- 通用语句解析 ----------

func (p *Parser) parseGenericStmt() sqlstmt.Stmt {
	start := p.Pos
	p.SkipToNextStatement()
	return &sqlstmt.OtherStmt{
		Base: sqlstmt.Base{Text: p.TextFrom(start)},
	}
}

// parseAssignment 解析 SET 字段赋值：column = value
func (p *Parser) parseAssignment() *sqlstmt.Assignment {
	start := p.Pos
	colText := ""
	for !p.Current().IsEOF() && p.Current().Value != "=" && p.Current().Value != "," &&
		!p.Current().IsKeyword("WHERE") && p.Current().Value != ";" {
		colText += p.Consume().Value
	}
	if p.Current().Value != "=" {
		p.Pos = start
		return nil
	}
	p.Consume() // =

	valStart := p.Pos
	for !p.Current().IsEOF() && p.Current().Value != "," &&
		!p.Current().IsKeyword("WHERE") && p.Current().Value != ";" {
		if p.Current().Value == "(" {
			p.SkipParentheses()
			continue
		}
		p.Consume()
	}

	return &sqlstmt.Assignment{
		Column: p.Unquote(strings.TrimSpace(colText)),
		Value:  &sqlstmt.Expr{Text: p.TextFromExclusive(valStart)},
		Text:   p.TextFromExclusive(start),
	}
}

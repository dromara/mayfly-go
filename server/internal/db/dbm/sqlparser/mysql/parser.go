package mysql

import (
	"strings"

	"mayfly-go/internal/db/dbm/sqlparser/base"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// Parser MySQL 方言 SQL 解析器
type Parser struct {
	*base.Lexer
}

// NewParser 创建 MySQL 解析器
func NewParser(sql string) *Parser {
	return &Parser{
		Lexer: base.NewLexer(sql, tokenizer.DialectConfig{
			BacktickAsIdentifier: true,
			HashLineComment:      true,
		}),
	}
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
		return p.parseCreate()
	case tok.IsKeyword("DROP"):
		return p.parseDrop()
	case tok.IsKeyword("ALTER"):
		return p.parseAlter()
	case tok.IsKeyword("WITH"):
		return p.parseWith()
	case tok.IsKeyword("SHOW"):
		return p.parseShow()
	case tok.IsKeyword("TRUNCATE"):
		return p.parseGenericDdl()
	case tok.IsKeyword("GRANT", "REVOKE", "RENAME", "ANALYZE"):
		// MySQL 的 GRANT、REVOKE 等均为非查询语句，统一按 DDL 执行
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
			p.SkipToNextStatement()
			return &sqlstmt.OtherStmt{Base: sqlstmt.Base{Text: p.TextFrom(start)}}
		}
		p.ExpectValue(")")
	} else if p.Current().IsKeyword("SELECT") {
		selectStmt = p.parseSelectBody()
	} else {
		return &sqlstmt.OtherStmt{Base: sqlstmt.Base{Text: p.TextFrom(start)}}
	}

	if selectStmt == nil {
		return &sqlstmt.OtherStmt{Base: sqlstmt.Base{Text: p.TextFrom(start)}}
	}

	// UNION
	for p.Current().IsKeyword("UNION") {
		p.Consume()
		isAll := false
		if p.Current().IsKeyword("ALL") {
			p.Consume()
			isAll = true
		} else if p.Current().IsKeyword("DISTINCT") {
			p.Consume()
		}
		unionStart := p.Pos
		var unionSelect *sqlstmt.SelectStmt
		if p.Current().Value == "(" {
			p.Consume()
			if p.Current().IsKeyword("SELECT") {
				unionSelect = p.parseSelectBody()
			}
			p.ExpectValue(")")
		} else if p.Current().IsKeyword("SELECT") {
			unionSelect = p.parseSelectBody()
		}
		if unionSelect != nil {
			// 如果 unionSelect 有 LIMIT 或 ORDER BY，移动到外层 selectStmt（UNION 的 LIMIT/ORDER BY 属于整个语句）
			if unionSelect.Limit != nil {
				selectStmt.Limit = unionSelect.Limit
				unionSelect.Limit = nil
			}
			if len(unionSelect.OrderBy) > 0 {
				selectStmt.OrderBy = unionSelect.OrderBy
				unionSelect.OrderBy = nil
			}
			unionSelect.Text = p.TextFrom(unionStart)
		}
		selectStmt.Unions = append(selectStmt.Unions, sqlstmt.UnionClause{
			Select: unionSelect,
			All:    isAll,
			Text:   p.TextFrom(unionStart),
		})
	}

	// UNION 之后可能有 ORDER BY
	if p.Current().IsKeyword("ORDER") {
		p.Consume()
		if p.Current().IsKeyword("BY") {
			p.Consume()
		}
		p.parseOrderBy(selectStmt)
	}

	// LIMIT (UNION 之后的 LIMIT 覆盖子查询内部的 LIMIT)
	// 注意：只在确实解析到新 LIMIT 时才覆盖，避免 nil 覆盖有效值
	if p.Current().IsKeyword("LIMIT") {
		if limit := p.parseLimit(); limit != nil {
			selectStmt.Limit = limit
		}
	}

	selectStmt.Text = p.TextFrom(start)
	return selectStmt
}

func (p *Parser) parseSelectBody() *sqlstmt.SelectStmt {
	if !p.Current().IsKeyword("SELECT") {
		return nil
	}
	start := p.Pos
	p.Consume()

	distinct := false
	if p.Current().IsKeyword("DISTINCT") {
		p.Consume()
		distinct = true
	} else if p.Current().IsKeyword("ALL") {
		p.Consume()
	}

	items := p.parseSelectItems()

	stmt := &sqlstmt.SelectStmt{
		Distinct: distinct,
		Items:    items,
	}

	if p.Current().IsKeyword("FROM") {
		p.Consume()
		p.parseFromClause(stmt)
	}

	if p.Current().IsKeyword("WHERE") {
		p.Consume()
		whereStart := p.Pos
		p.SkipExpr()
		stmt.Where = &sqlstmt.Expr{Text: p.TextFromExclusive(whereStart)}
	}

	if p.Current().IsKeyword("GROUP") {
		p.Consume()
		if p.Current().IsKeyword("BY") {
			p.Consume()
		}
		p.SkipGroupByExpr()
	}

	if p.Current().IsKeyword("HAVING") {
		p.Consume()
		p.SkipExpr()
	}

	if p.Current().IsKeyword("ORDER") {
		p.Consume()
		if p.Current().IsKeyword("BY") {
			p.Consume()
		}
		p.parseOrderBy(stmt)
	}

	// LIMIT (子查询或简单 SELECT 的 LIMIT)
	if p.Current().IsKeyword("LIMIT") {
		stmt.Limit = p.parseLimit()
	}

	stmt.Text = p.TextFrom(start)
	return stmt
}

func (p *Parser) parseSelectItems() []sqlstmt.SelectItem {
	items := make([]sqlstmt.SelectItem, 0)
	for !p.Current().IsEOF() {
		tok := p.Current()
		if tok.Value == "," {
			p.Consume()
			continue
		}
		if p.IsSelectClauseEnd() {
			break
		}
		if tok.Value == "*" {
			p.Consume()
			items = append(items, sqlstmt.SelectItem{
				Kind: sqlstmt.SelectItemStar,
				Text: "*",
			})
		} else if tok.Type == tokenizer.TokenIdentifier && p.Peek(1).Value == "." && p.Peek(2).Value == "*" {
			tableAlias := p.Unquote(p.Consume().Value)
			p.Consume()
			p.Consume()
			items = append(items, sqlstmt.SelectItem{
				Kind:       sqlstmt.SelectItemStar,
				Text:       tableAlias + ".*",
				TableAlias: tableAlias,
			})
		} else {
			colStart := p.Pos
			p.skipSelectElement()
			colText := base.TrimTrailingComma(p.TextFromExclusive(colStart))
			// 去除标识符引用符，如 mysql 的 `id` -> id
			colText = p.Unquote(colText)
			colName, alias := p.ExtractColumnAndAlias(colText)
			kind := sqlstmt.SelectItemColumn
			if strings.Contains(colText, "(") {
				kind = sqlstmt.SelectItemExpr
			}
			items = append(items, sqlstmt.SelectItem{
				Kind:       kind,
				Text:       colText,
				Alias:      alias,
				ColumnName: colName,
			})
		}
	}
	return items
}

func (p *Parser) skipSelectElement() {
	for !p.Current().IsEOF() {
		tok := p.Current()
		if tok.Value == "," || p.IsSelectClauseEnd() {
			break
		}
		if tok.Value == "(" {
			p.SkipParentheses()
			continue
		}
		p.Consume()
	}
}

// ---------- FROM / TableRef 解析 ----------

func (p *Parser) parseFromClause(stmt *sqlstmt.SelectStmt) {
	for !p.Current().IsEOF() {
		if p.IsFromClauseEnd() {
			break
		}
		if p.Current().Value == "," {
			p.Consume()
			continue
		}
		if p.Current().IsKeyword("JOIN") || p.IsJoinStart() {
			join := p.parseJoinClause()
			if join != nil {
				stmt.Joins = append(stmt.Joins, *join)
			}
			continue
		}
		tableRef := p.parseTableRef()
		if tableRef.Name != "" {
			stmt.From = append(stmt.From, tableRef)
		} else {
			break
		}
	}
}

func (p *Parser) parseTableRef() sqlstmt.TableRef {
	if p.Current().Value == "(" {
		start := p.Pos
		p.SkipParentheses()
		alias := ""
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
		onStart := p.Pos
		p.SkipExpr()
		onExpr = &sqlstmt.Expr{Text: p.TextFromExclusive(onStart)}
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
		Text:  p.TextFrom(start),
	}
}

func (p *Parser) parseOrderBy(stmt *sqlstmt.SelectStmt) {
	for !p.Current().IsEOF() {
		if p.Current().Value == "," {
			p.Consume()
			continue
		}
		if p.IsExprEnd() || p.Current().Value == ";" {
			break
		}
		itemStart := p.Pos
		p.skipOrderByItem()
		itemText := base.TrimTrailingComma(p.TextFromExclusive(itemStart))
		desc := false
		upper := strings.ToUpper(itemText)
		if strings.HasSuffix(upper, " DESC") {
			desc = true
			itemText = strings.TrimSpace(itemText[:len(itemText)-5])
		} else if strings.HasSuffix(upper, " ASC") {
			itemText = strings.TrimSpace(itemText[:len(itemText)-4])
		}
		stmt.OrderBy = append(stmt.OrderBy, sqlstmt.OrderByItem{
			Text: itemText,
			Desc: desc,
		})
	}
}

func (p *Parser) skipOrderByItem() {
	for !p.Current().IsEOF() {
		if p.Current().Value == "," {
			break
		}
		if p.IsExprEnd() || p.Current().Value == ";" {
			break
		}
		if p.Current().Value == "(" {
			p.SkipParentheses()
			continue
		}
		p.Consume()
	}
}

// ---------- LIMIT 解析 ----------

func (p *Parser) parseLimit() *sqlstmt.Limit {
	if !p.Current().IsKeyword("LIMIT") {
		return nil
	}
	start := p.Pos
	p.Consume()
	rowCount := 0
	offset := 0
	if p.Current().Type == tokenizer.TokenNumber {
		rowCount = p.ParseInt(p.Consume().Value)
	}
	if p.Current().Value == "," {
		p.Consume()
		offset = rowCount
		if p.Current().Type == tokenizer.TokenNumber {
			rowCount = p.ParseInt(p.Consume().Value)
		}
	}
	if p.Current().IsKeyword("OFFSET") {
		p.Consume()
		if p.Current().Type == tokenizer.TokenNumber {
			offset = p.ParseInt(p.Consume().Value)
		}
	}
	return &sqlstmt.Limit{
		Text:   p.TextFrom(start),
		Count:  rowCount,
		Offset: offset,
	}
}

// ---------- INSERT 解析 ----------

func (p *Parser) parseInsert() sqlstmt.Stmt {
	start := p.Pos
	p.Consume()
	if p.Current().IsKeyword("INTO") {
		p.Consume()
	}

	tableRef := p.parseTableRef()
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

	valuesStart := p.Pos
	if p.Current().IsKeyword("VALUES") {
		p.Consume()
		for !p.Current().IsEOF() && !p.IsExprEnd() {
			if p.Current().Value == "(" {
				p.SkipParentheses()
				continue
			}
			p.Consume()
		}
	} else if p.Current().IsKeyword("SELECT") {
		p.parseSelect()
	}

	return &sqlstmt.InsertStmt{
		Base:    sqlstmt.Base{Text: p.TextFrom(start)},
		Table:   tableRef,
		Columns: columns,
		Values:  p.TextFrom(valuesStart),
	}
}

// ---------- UPDATE 解析 ----------

func (p *Parser) parseUpdate() sqlstmt.Stmt {
	start := p.Pos
	p.Consume()

	tables := make([]sqlstmt.TableRef, 0)
	for !p.Current().IsEOF() {
		if p.Current().IsKeyword("SET") || p.Current().IsKeyword("WHERE") || p.Current().Value == ";" {
			break
		}
		if p.Current().Value == "," {
			p.Consume()
			continue
		}
		if p.Current().IsKeyword("JOIN") || p.IsJoinStart() {
			p.parseJoinClause()
			continue
		}
		tableRef := p.parseTableRef()
		if tableRef.Name != "" {
			tables = append(tables, tableRef)
		} else {
			break
		}
	}

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

	var where *sqlstmt.Expr
	if p.Current().IsKeyword("WHERE") {
		p.Consume()
		whereStart := p.Pos
		p.SkipExpr()
		where = &sqlstmt.Expr{Text: p.TextFromExclusive(whereStart)}
	}

	// MySQL UPDATE 支持 ORDER BY, LIMIT
	if p.Current().IsKeyword("ORDER") {
		p.Consume()
		if p.Current().IsKeyword("BY") {
			p.Consume()
		}
		p.SkipOrderByExpr()
	}
	if p.Current().IsKeyword("LIMIT") {
		p.parseLimit()
	}

	return &sqlstmt.UpdateStmt{
		Base:   sqlstmt.Base{Text: p.TextFrom(start)},
		Tables: tables,
		Set:    assignments,
		Where:  where,
	}
}

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
	p.Consume()

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
		Text:   p.TextFrom(start),
	}
}

// ---------- DELETE 解析 ----------

func (p *Parser) parseDelete() sqlstmt.Stmt {
	start := p.Pos
	p.Consume()
	if p.Current().IsKeyword("FROM") {
		p.Consume()
	}

	tables := make([]sqlstmt.TableRef, 0)
	for !p.Current().IsEOF() {
		if p.Current().IsKeyword("WHERE") || p.Current().IsKeyword("USING") ||
			p.Current().IsKeyword("ORDER") || p.Current().IsKeyword("LIMIT") ||
			p.Current().Value == ";" {
			break
		}
		if p.Current().Value == "," {
			p.Consume()
			continue
		}
		tableRef := p.parseTableRef()
		if tableRef.Name != "" {
			tables = append(tables, tableRef)
		} else {
			break
		}
	}

	var where *sqlstmt.Expr
	if p.Current().IsKeyword("WHERE") {
		p.Consume()
		whereStart := p.Pos
		p.SkipExpr()
		where = &sqlstmt.Expr{Text: p.TextFromExclusive(whereStart)}
	}

	// MySQL DELETE 支持 ORDER BY, LIMIT
	if p.Current().IsKeyword("ORDER") {
		p.Consume()
		if p.Current().IsKeyword("BY") {
			p.Consume()
		}
		p.SkipOrderByExpr()
	}
	if p.Current().IsKeyword("LIMIT") {
		p.parseLimit()
	}

	return &sqlstmt.DeleteStmt{
		Base:   sqlstmt.Base{Text: p.TextFrom(start)},
		Tables: tables,
		Where:  where,
	}
}

// ---------- DDL 解析 ----------

func (p *Parser) parseCreate() sqlstmt.Stmt {
	start := p.Pos
	p.SkipToNextStatement()
	return &sqlstmt.DdlStmt{
		Base:    sqlstmt.Base{Text: p.TextFrom(start)},
		DdlKind: "CREATE",
	}
}

func (p *Parser) parseDrop() sqlstmt.Stmt {
	start := p.Pos
	p.SkipToNextStatement()
	return &sqlstmt.DdlStmt{
		Base:    sqlstmt.Base{Text: p.TextFrom(start)},
		DdlKind: "DROP",
	}
}

func (p *Parser) parseAlter() sqlstmt.Stmt {
	start := p.Pos
	p.SkipToNextStatement()
	return &sqlstmt.DdlStmt{
		Base:    sqlstmt.Base{Text: p.TextFrom(start)},
		DdlKind: "ALTER",
	}
}

func (p *Parser) parseGenericDdl() sqlstmt.Stmt {
	start := p.Pos
	p.SkipToNextStatement()
	return &sqlstmt.DdlStmt{
		Base:    sqlstmt.Base{Text: p.TextFrom(start)},
		DdlKind: "DDL",
	}
}

// ---------- SHOW 解析（MySQL 特有）----------

func (p *Parser) parseShow() sqlstmt.Stmt {
	start := p.Pos
	p.SkipToNextStatement()
	return &sqlstmt.SelectStmt{
		Base: sqlstmt.Base{Text: p.TextFrom(start)},
	}
}

// ---------- WITH 解析 ----------

func (p *Parser) parseWith() sqlstmt.Stmt {
	start := p.Pos
	p.SkipToNextStatement()
	return &sqlstmt.WithStmt{
		Base: sqlstmt.Base{Text: p.TextFrom(start)},
	}
}

// ---------- 通用语句解析 ----------

func (p *Parser) parseGenericStmt() sqlstmt.Stmt {
	start := p.Pos
	p.SkipToNextStatement()
	return &sqlstmt.OtherStmt{
		Base: sqlstmt.Base{Text: p.TextFrom(start)},
	}
}

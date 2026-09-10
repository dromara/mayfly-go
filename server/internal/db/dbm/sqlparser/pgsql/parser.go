package pgsql

import (
	"strconv"
	"strings"

	"mayfly-go/internal/db/dbm/sqlparser/base"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// Parser PostgreSQL 方言 SQL 解析器
type Parser struct {
	*base.Lexer
}

// NewParser 创建 PostgreSQL 解析器
func NewParser(sql string) *Parser {
	p := &Parser{Lexer: base.NewLexer(sql, tokenizer.PgConfig)}
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
	case tok.IsKeyword("COMMENT", "GRANT", "REVOKE"):
		// PostgreSQL 的 COMMENT ON、GRANT、REVOKE 等均为非查询语句，统一按 DDL 执行
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
	if selectStmt.OrderBy == nil && p.Current().IsKeyword("ORDER") {
		p.Consume()
		if p.Current().IsKeyword("BY") {
			p.Consume()
		}
		selectStmt.OrderBy = p.parseOrderBy()
	}

	// LIMIT（UNION 之后的 LIMIT 覆盖子查询内部的 LIMIT）
	if p.Current().IsKeyword("LIMIT") || p.Current().IsKeyword("OFFSET") {
		if limit := p.parseLimit(); limit != nil {
			selectStmt.Limit = limit
		}
	}

	// FOR UPDATE
	if p.Current().IsKeyword("FOR") {
		p.Consume()
		if p.Current().IsKeyword("UPDATE") {
			p.Consume()
		}
	}

	// RETURNING（PostgreSQL 特有）
	if p.Current().IsKeyword("RETURNING") {
		p.Consume()
		// 跳过 RETURNING 后面的所有列（可能包含逗号）
		for !p.Current().IsEOF() && p.Current().Value != ";" {
			p.Consume()
		}
	}

	// 更新完整文本（包含所有已解析的内容）
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

	// WHERE
	if p.Current().IsKeyword("WHERE") {
		p.Consume()
		selectStmt.Where = p.ParseCondExpr(false)
	}

	// GROUP BY（表达式不建模，回填原始文本标记其存在，供调用方判定语句形态）
	if p.Current().IsKeyword("GROUP") {
		p.Consume()
		if p.Current().IsKeyword("BY") {
			p.Consume()
		}
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

	// ORDER BY
	if p.Current().IsKeyword("ORDER") {
		p.Consume()
		if p.Current().IsKeyword("BY") {
			p.Consume()
		}
		selectStmt.OrderBy = p.parseOrderBy()
	}

	// LIMIT/OFFSET（子查询或简单 SELECT 的 LIMIT）
	if p.Current().IsKeyword("LIMIT") || p.Current().IsKeyword("OFFSET") {
		selectStmt.Limit = p.parseLimit()
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
			// 如果 unionSelect 有 LIMIT 或 ORDER BY，移动到外层 selectStmt
			if nextSelect.Limit != nil {
				selectStmt.Limit = nextSelect.Limit
				nextSelect.Limit = nil
			}
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

// LIMIT 解析（PostgreSQL: OFFSET ... LIMIT 或 LIMIT ... ALL）
func (p *Parser) parseLimit() *sqlstmt.Limit {
	start := p.Pos
	limit := &sqlstmt.Limit{}

	// PostgreSQL 支持两种顺序：
	// 1. LIMIT count OFFSET offset
	// 2. OFFSET offset LIMIT count
	if p.Current().IsKeyword("LIMIT") {
		p.Consume()
		// LIMIT ALL 或 LIMIT 值
		if p.Current().IsKeyword("ALL") {
			p.Consume()
			limit.Text = p.TextFrom(start)
			return limit
		}
		if p.Current().Type == tokenizer.TokenNumber {
			countStr := p.Consume().Value
			limit.Count, _ = strconv.Atoi(countStr)
		}
		// 检查是否有 OFFSET
		if p.Current().IsKeyword("OFFSET") {
			p.Consume()
			if p.Current().Type == tokenizer.TokenNumber {
				offsetStr := p.Consume().Value
				limit.Offset, _ = strconv.Atoi(offsetStr)
			} else if p.Current().Type == tokenizer.TokenIdentifier {
				p.Consume()
			}
		}
	} else if p.Current().IsKeyword("OFFSET") {
		p.Consume()
		// 跳过 OFFSET 值
		if p.Current().Type == tokenizer.TokenNumber {
			offsetStr := p.Consume().Value
			limit.Offset, _ = strconv.Atoi(offsetStr)
		} else if p.Current().Type == tokenizer.TokenIdentifier {
			p.Consume()
		}
		// 检查是否有限制
		if p.Current().IsKeyword("LIMIT") {
			p.Consume()
			if p.Current().IsKeyword("ALL") {
				p.Consume()
				limit.Text = p.TextFrom(start)
				return limit
			}
			if p.Current().Type == tokenizer.TokenNumber {
				countStr := p.Consume().Value
				limit.Count, _ = strconv.Atoi(countStr)
			}
		}
	}

	limit.Text = p.TextFrom(start)
	return limit
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
			// 使用完整解析（包含 UNION）
			p.parseSelect()
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

	// 使用 parseTableRef 正确解析表名（支持 schema.table 和双引号）
	tableRef := p.parseTableRef()

	// 解析列名列表 (col1, col2, ...)
	columns := []string{}
	if p.Current().Value == "(" {
		p.Consume() // (
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
			p.Consume() // )
		}
	}

	// VALUES 或 SELECT 或直接是 ON CONFLICT
	if p.Current().IsKeyword("VALUES") || p.Current().Value == "VALUES" {
		p.Consume()
		// 消费到 ON 或 RETURNING 或 EOF
		for !p.Current().IsEOF() {
			val := strings.ToUpper(p.Current().Value)
			if val == "ON" || val == "RETURNING" {
				break
			}
			if p.Current().Value == "(" {
				p.SkipParentheses()
			} else {
				p.Consume()
			}
		}
	} else if p.Current().IsKeyword("SELECT") || p.Current().Value == "SELECT" {
		// INSERT INTO ... SELECT ...
		p.parseSelect()
	}

	// ON CONFLICT（PostgreSQL 特有）
	if strings.ToUpper(p.Current().Value) == "ON" {
		p.Consume()
		if p.Current().IsKeyword("CONFLICT") {
			p.Consume()
			// 跳过 ON CONFLICT 后面的所有内容直到 RETURNING 或 EOF
			for !p.Current().IsEOF() && !p.Current().IsKeyword("RETURNING") {
				p.Consume()
			}
		}
	}

	// RETURNING（PostgreSQL 特有）
	if p.Current().IsKeyword("RETURNING") {
		p.Consume()
		// 跳过 RETURNING 后面的内容
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
// PostgreSQL UPDATE 不支持 ORDER BY/LIMIT

func (p *Parser) parseUpdate() sqlstmt.Stmt {
	start := p.Pos
	p.Consume() // UPDATE

	// 使用 parseTableRef 正确解析表名（支持 schema.table 和双引号）
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

	// RETURNING（PostgreSQL 特有）
	if p.Current().IsKeyword("RETURNING") {
		p.Consume()
		// 跳过 RETURNING 后面的所有列（可能包含逗号）
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
// PostgreSQL DELETE 不支持 ORDER BY/LIMIT

func (p *Parser) parseDelete() sqlstmt.Stmt {
	start := p.Pos
	p.Consume() // DELETE

	if p.Current().IsKeyword("FROM") {
		p.Consume()
	}

	// 使用 parseTableRef 正确解析表名（支持 schema.table 和双引号）
	tableRef := p.parseTableRef()
	tables := []sqlstmt.TableRef{tableRef}

	// WHERE
	var where *sqlstmt.Expr
	if p.Current().IsKeyword("WHERE") {
		p.Consume()
		where = p.ParseCondExpr(false)
	}

	// RETURNING（PostgreSQL 特有）
	if p.Current().IsKeyword("RETURNING") {
		p.Consume()
		// 跳过 RETURNING 后面的所有列（可能包含逗号）
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

// ---------- WITH 解析 ----------

// ---------- 通用语句解析 ----------

func (p *Parser) parseGenericStmt() sqlstmt.Stmt {
	start := p.Pos
	p.SkipToNextStatement()
	return &sqlstmt.OtherStmt{
		Base: sqlstmt.Base{Text: p.TextFrom(start)},
	}
}

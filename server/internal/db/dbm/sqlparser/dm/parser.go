package dm

import (
	"mayfly-go/internal/db/dbm/sqlparser/base"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// Parser 达梦方言 SQL 解析器。
//
// 达梦兼容 Oracle 的层次查询与 FETCH 分页，同时支持 MySQL 兼容模式的 LIMIT / SELECT TOP，
// 这些形态全部由 base 骨架覆盖；本方言只需登记词法能力（q-quote、PL-SQL 块、MINUS）与解析入口
type Parser struct {
	*base.Lexer
}

// NewParser 创建达梦解析器
func NewParser(sql string) *Parser {
	return &Parser{Lexer: base.NewLexer(sql, tokenizer.DmConfig)}
}

// Parse 解析单条 SQL 语句
func (p *Parser) Parse() (sqlstmt.Stmt, error) {
	p.SkipSemicolons()
	if p.Current().IsEOF() {
		return nil, nil
	}
	return p.parseStatement(), nil
}

func (p *Parser) parseStatement() sqlstmt.Stmt {
	if stmt, ok := p.ParseSharedStatement(); ok {
		return stmt
	}
	return p.ParseOtherStmt()
}

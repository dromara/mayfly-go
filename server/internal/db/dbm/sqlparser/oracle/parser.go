package oracle

import (
	"mayfly-go/internal/db/dbm/sqlparser/base"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// Parser Oracle 方言 SQL 解析器。
//
// Oracle 的 SELECT / 层次查询（START WITH / CONNECT BY）/ 分页（OFFSET n ROWS、
// FETCH FIRST n ROWS ONLY）/ 锁（FOR UPDATE NOWAIT | WAIT n | SKIP LOCKED）/ RETURNING
// 与 UPDATE/DELETE/INSERT 均由 base 骨架实现，本方言只需登记词法能力（q-quote、PL-SQL 块、MINUS）
// 与解析入口：骨架对这些子句的处理是跨方言的，Oracle 特有形态通过能力表与无歧义位置的单词匹配生效
type Parser struct {
	*base.Lexer
}

// NewParser 创建 Oracle 解析器
func NewParser(sql string) *Parser {
	return &Parser{Lexer: base.NewLexer(sql, tokenizer.OracleConfig)}
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

package mssql

import (
	"mayfly-go/internal/db/dbm/sqlparser/base"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// Parser SQL Server（T-SQL）方言解析器。
//
// 语法骨架与各方言共用 base，本方言的差异全在词法能力表：MssqlConfig 打开方括号标识符
// （[dbo].[users]）与 BEGIN..END 块感知。曾直接复用 pgsql 解析器，使 SQL Server 的
// `[x]` 引用符在解析阶段从未被当作标识符 —— 表名/列名提取因此失真
type Parser struct {
	*base.Lexer
}

// newParser 创建 SQL Server 语法解析器（对外入口是本包的 MssqlParser，它额外实现分页改写与语句分类能力）
func newParser(sql string) *Parser {
	return &Parser{Lexer: base.NewLexer(sql, tokenizer.MssqlConfig)}
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

package clickhouse

import (
	"mayfly-go/internal/db/dbm/sqlparser/base"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// Parser ClickHouse 方言解析器。
//
// 语法骨架与各方言共用 base，本方言差异只在词法能力表：ClickHouse 用反引号作标识符引用符，
// # 与 -- 都是行注释（-- 不要求后随空白）。曾复用 pgsql 解析器，使反引号列名/表名在解析阶段
// 不被识别为引用名；ClickHouse 原生的 LIMIT o, c 形态也依赖 base 的统一分页解析
type Parser struct {
	*base.Lexer
}

// NewParser 创建 ClickHouse 解析器
func NewParser(sql string) *Parser {
	return &Parser{Lexer: base.NewLexer(sql, tokenizer.ClickhouseConfig)}
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

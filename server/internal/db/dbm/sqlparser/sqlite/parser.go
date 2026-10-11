package sqlite

import (
	"mayfly-go/internal/db/dbm/sqlparser/base"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// Parser SQLite 方言解析器。
//
// 语法骨架与各方言共用 base，本方言差异只在词法能力表：SQLite 同时接受 `x`、"x"、[x] 三种引用符，
// 且反斜杠在字符串内不是转义符。曾复用 pgsql 解析器（Pg 配置），使反引号与方括号引用名在解析阶段
// 不被识别，表名/列名提取失真
type Parser struct {
	*base.Lexer
}

// NewParser 创建 SQLite 解析器
func NewParser(sql string) *Parser {
	return &Parser{Lexer: base.NewLexer(sql, tokenizer.SQLiteConfig)}
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
	// INSERT 先于共享分发：本方言的 INSERT 有 INSERT OR <动作> 与 ON CONFLICT 两种冲突处理形态
	if p.Current().IsKeyword("INSERT") {
		return p.parseInsert()
	}
	if stmt, ok := p.ParseSharedStatement(); ok {
		return stmt
	}
	return p.ParseOtherStmt()
}

// parseInsert INSERT 骨架 + SQLite 的 INSERT OR REPLACE/IGNORE 冲突形态。
//
// OR REPLACE / OR IGNORE 与 pg 的 ON CONFLICT、mysql 的 REPLACE INTO 同属「目标行已存在时不新增行」
// 的 upsert 语义，标记冲突处理形态让采集层跳过插入快照，否则回滚的 DELETE 会删掉那条既有行
func (p *Parser) parseInsert() sqlstmt.Stmt {
	start := p.Pos
	hasConflict := p.PeekIsWord(1, "OR") && p.PeekIsWord(2, "REPLACE", "IGNORE", "ROLLBACK", "ABORT", "FAIL")
	table, columns, valueRows := p.ParseInsertHead()
	if p.Current().IsKeyword("ON") {
		p.Consume()
		if p.Current().IsKeyword("CONFLICT") {
			hasConflict = true
			p.Consume()
			for !p.Current().IsEOF() && !p.Current().IsKeyword("RETURNING") {
				p.Consume()
			}
		}
	}
	p.ParseReturning()

	return &sqlstmt.InsertStmt{
		Base:              sqlstmt.Base{Text: p.TextFrom(start)},
		Table:             table,
		Columns:           columns,
		ValueRows:         valueRows,
		HasConflictClause: hasConflict,
	}
}

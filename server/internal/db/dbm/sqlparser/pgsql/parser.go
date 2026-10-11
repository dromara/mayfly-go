package pgsql

import (
	"mayfly-go/internal/db/dbm/sqlparser/base"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// Parser PostgreSQL 方言 SQL 解析器。
//
// SELECT / 表引用 / JOIN / 分页 / 锁与 RETURNING / UPDATE / DELETE / SET 赋值都由 base 骨架实现一次；
// PostgreSQL 真正特有的部分只有 ON CONFLICT 冲突处理子句。
//
// 注意本解析器同时被 SQLite（dialect/sqlite）复用：pg 侧漏解析的行限制形态会连带让
// SQLite 的 UPDATE/DELETE … LIMIT fail-closed 守卫失效，因此分页与尾部子句一律以 base 的超集实现兜住
type Parser struct {
	*base.Lexer
}

// NewParser 创建 PostgreSQL 解析器
func NewParser(sql string) *Parser {
	return &Parser{Lexer: base.NewLexer(sql, tokenizer.PgConfig)}
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
	// INSERT 先于共享分发：本方言的 INSERT 带 ON CONFLICT 冲突处理子句
	if p.Current().IsKeyword("INSERT") {
		return p.parseInsert()
	}
	if stmt, ok := p.ParseSharedStatement(); ok {
		return stmt
	}
	return p.ParseOtherStmt()
}

// ---------- INSERT 解析 ----------

// parseInsert 解析 INSERT，含 PostgreSQL 的 ON CONFLICT 冲突处理子句。
//
// ON CONFLICT DO NOTHING / DO UPDATE 在目标行已存在时并不新增行（忽略或改为更新），
// 而 INSERT 的回滚是「删掉刚插入的行」——按 VALUES 字面量生成的 DELETE 会命中并删掉那条
// 既有行（数据丢失），故标记为冲突处理形态，由采集层 fail-closed 跳过插入快照
func (p *Parser) parseInsert() sqlstmt.Stmt {
	start := p.Pos
	table, columns, valueRows := p.ParseInsertHead()

	hasConflict := false
	if p.Current().IsKeyword("ON") {
		p.Consume()
		if p.Current().IsKeyword("CONFLICT") {
			hasConflict = true
			p.Consume()
			// 冲突目标与 DO 动作不建模，消费到 RETURNING 或语句结束
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

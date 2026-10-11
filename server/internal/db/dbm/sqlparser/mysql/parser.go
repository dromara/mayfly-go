package mysql

import (
	"slices"

	"mayfly-go/internal/db/dbm/sqlparser/base"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// Parser MySQL 方言 SQL 解析器。
//
// SELECT / 表引用 / JOIN / 分页 / 层次与锁子句 / UPDATE / DELETE / SET 赋值都由 base 骨架实现一次；
// MySQL 真正特有的部分只有：REPLACE INTO、INSERT IGNORE、ON DUPLICATE KEY UPDATE、SHOW，
// 以及已在能力表（ExtraKeywords）登记的 STRAIGHT_JOIN 连接词
type Parser struct {
	*base.Lexer
}

// NewParser 创建 MySQL 解析器
func NewParser(sql string) *Parser {
	return &Parser{Lexer: base.NewLexer(sql, tokenizer.MysqlConfig)}
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
	switch tok := p.Current(); {
	case tok.IsKeyword("INSERT", "REPLACE"):
		// 先于共享分发：本方言的 INSERT 有 IGNORE / ON DUPLICATE KEY UPDATE 冲突处理形态，
		// REPLACE INTO 与 INSERT 同构（否则会被判为 OtherStmt 而走不到写入分支）
		return p.parseInsert()
	case tok.IsKeyword("SHOW"):
		return p.parseShow()
	}
	if stmt, ok := p.ParseSharedStatement(); ok {
		return stmt
	}
	return p.ParseOtherStmt()
}

// ---------- INSERT 解析 ----------

// parseInsert 解析 INSERT / REPLACE INTO。
//
// 三种 upsert 写形态都会让「INSERT 的回滚 = 删掉刚插入的行」失真，故统一标记冲突处理形态，
// 由采集层 fail-closed 跳过插入快照：
//   - REPLACE INTO：先删冲突行再插入，被顶掉的既有行无法恢复；
//   - INSERT IGNORE：冲突时跳过插入，目标行可能原本已存在；
//   - ON DUPLICATE KEY UPDATE：冲突时改为更新既有行
func (p *Parser) parseInsert() sqlstmt.Stmt {
	start := p.Pos
	// REPLACE INTO 与 INSERT IGNORE 都是 upsert 写形态，与 ON DUPLICATE KEY UPDATE 同等处理：
	// 目标行可能原本已存在，而 INSERT 的回滚是「删掉刚插入行」，会连带删掉那条既有行（数据丢失）。
	// 修饰词个数不定（INSERT DELAYED IGNORE INTO ...），故扫整段而不是只看 INSERT 之后第一位
	hasConflict := p.Current().IsKeyword("REPLACE") || slices.Contains(p.ModifiersAfter(1), "IGNORE")
	table, columns, valueRows := p.ParseInsertHead()

	// ON DUPLICATE KEY UPDATE：整段消费到语句结束
	if p.Current().IsKeyword("ON") {
		hasConflict = true
		p.SkipToStatementEnd()
	}

	return &sqlstmt.InsertStmt{
		Base:              sqlstmt.Base{Text: p.TextFrom(start)},
		Table:             table,
		Columns:           columns,
		ValueRows:         valueRows,
		HasConflictClause: hasConflict,
	}
}

// ---------- SHOW 解析（MySQL 特有）----------

// parseShow SHOW 按查询形态返回（执行链路走查询分支），列结构由服务端决定故不建模
func (p *Parser) parseShow() sqlstmt.Stmt {
	start := p.Pos
	p.SkipToNextStatement()
	return &sqlstmt.SelectStmt{
		Base: sqlstmt.Base{Text: p.TextFrom(start)},
	}
}

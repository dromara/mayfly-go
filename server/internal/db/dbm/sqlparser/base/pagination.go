package base

// 行限制（分页）子句的统一解析。
//
// 各方言分页写法曾各写一份（mysql 的 LIMIT o,c、pg 的 LIMIT ALL、oracle/dm 的 OFFSET n ROWS
// FETCH NEXT m ROWS ONLY），漂移的代价不对称且都很硬：
//   - SELECT 侧丢 FETCH → 消费方看不到行限制（数据同步的语句准入按「无 LIMIT」放行）；
//   - UPDATE/DELETE 侧丢 LIMIT → 采集层的 fail-closed 守卫失效，回查会捞回 WHERE 匹配的全部行，
//     为未被本句影响的那些行生成覆盖式回滚（DELETE 场景还会重插未被删的行）。
//   - pg 解析器同时服务 SQLite/ClickHouse：ClickHouse 原生的 LIMIT o,c、SQLite 的
//     UPDATE…ORDER BY…LIMIT 都落在这一份实现上。
//
// 因此这里一次覆盖全部形态。解析器是宽容解析而非语法校验器：接受超集不会误伤任何合法 SQL
// （某方言把别种分页写法用错，服务端执行时自然报错），而少解析一段则是静默的错误决策。

import (
	"strconv"
	"strings"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// statementModifiers DML 的语句级修饰词。
//
// 按关键字类匹配（只有登记进该方言 ExtraKeywords 的词才会命中），因此 mysql 的 IGNORE/DELAYED/
// LOW_PRIORITY/HIGH_PRIORITY/QUICK 不会在别的方言里吞掉同名裸表名。
// 单一清单：骨架消费它们、方言判定冲突形态时也读同一份（见 ModifiersAfter）
var statementModifiers = []string{"IGNORE", "DELAYED", "LOW_PRIORITY", "HIGH_PRIORITY", "QUICK"}

// ModifiersAfter 自偏移 offset 起扫描连续的语句级修饰词，返回出现过的修饰词（大写）。
// 修饰词个数不定（如 `INSERT DELAYED IGNORE INTO`），故方言要判断「这一段里是否有 IGNORE」
// 不能只看 INSERT 之后的第一个 token
func (l *Lexer) ModifiersAfter(offset int) []string {
	var out []string
	for i := offset; i < l.Length; i++ {
		tok := l.Peek(i)
		if !tok.IsKeyword(statementModifiers...) {
			break
		}
		out = append(out, strings.ToUpper(tok.Value))
	}
	return out
}

// parseRowLimitTail 解析位于子句尾部的行限制子句；当前不处于行限制形态时返回 nil。
//
// 覆盖形态：LIMIT n / LIMIT o, c / LIMIT n OFFSET o / OFFSET o [LIMIT n] / LIMIT ALL /
// OFFSET n ROW|ROWS / [OFFSET n ROWS] FETCH FIRST|NEXT m ROW|ROWS ONLY|WITH TIES。
// 各片段可以任意先后顺序出现，因此循环消费直到不再匹配。
//
// 返回非 nil 即表示语句只作用于一个行子集（Limit.ALL 之类的「不限制」形态也返回非 nil，
// 语义上更保守：消费方按有行限制处理，宁可不采集回滚也不生成错误的回滚）。
func (l *Lexer) parseRowLimitTail() *sqlstmt.Limit {
	start := l.Pos
	lim := &sqlstmt.Limit{}
	matched := false

	for {
		switch {
		case l.Current().IsKeyword("LIMIT"):
			l.Consume()
			matched = true
			if l.Current().IsKeyword("ALL") {
				// pg 的 LIMIT ALL：不限制行数（仍按「带行限制子句」上报，保持保守语义）
				l.Consume()
				continue
			}
			if isNumberToken(l.Current()) {
				first, _ := strconv.Atoi(l.Consume().Value)
				if l.Current().Value == "," {
					// LIMIT offset, count（mysql / ClickHouse / 达梦兼容模式）
					l.Consume()
					lim.Offset = first
					if isNumberToken(l.Current()) {
						lim.Count, _ = strconv.Atoi(l.Consume().Value)
					}
					continue
				}
				// LIMIT count [OFFSET offset]
				lim.Count = first
			}
			continue
		case l.Current().IsKeyword("OFFSET"):
			l.Consume()
			matched = true
			if isNumberToken(l.Current()) {
				lim.Offset, _ = strconv.Atoi(l.Consume().Value)
				// OFFSET n ROW[S]（oracle 12c+ / 达梦）
				if l.Current().IsKeyword("ROW", "ROWS") {
					l.Consume()
				}
			}
			continue
		case l.Current().IsKeyword("FETCH"):
			l.Consume()
			matched = true
			if l.Current().IsKeyword("FIRST", "NEXT") {
				l.Consume()
			}
			if isNumberToken(l.Current()) {
				lim.Count, _ = strconv.Atoi(l.Consume().Value)
			}
			if l.Current().IsKeyword("ROW", "ROWS") {
				l.Consume()
			}
			if l.Current().IsKeyword("ONLY") {
				l.Consume()
			} else if l.Current().IsKeyword("WITH") {
				l.Consume()
				if l.Current().IsKeyword("TIES") {
					l.Consume()
				}
			}
			continue
		}
		break
	}

	if !matched {
		return nil
	}
	lim.Text = l.TextFromExclusive(start)
	return lim
}

// parseTopLimit 解析 SELECT 与投影之间的 TOP n（SQL Server / 达梦形态）。
//
// 仅当 TOP 后紧跟数字时才成立，所以 `SELECT top FROM t`（top 作列名）与 `SELECT a, b` 之类
// 都不受影响；这也让本函数无需把 TOP 登记成关键字即可工作（TOP 在 Oracle 系不是保留字）。
// 未识别到时返回 nil，不消费任何 token。
func (l *Lexer) parseTopLimit() *sqlstmt.Limit {
	if !l.IsWord("TOP") || !isNumberToken(l.Peek(1)) {
		return nil
	}
	start := l.Pos
	l.Consume() // TOP
	count, _ := strconv.Atoi(l.Consume().Value)
	if l.IsWord("PERCENT") {
		// TOP n PERCENT（SQL Server）：百分比形态无法折算成行数，仅消费文本
		l.Consume()
		count = 0
	}
	if l.isWordPair("WITH", "TIES") {
		l.Consume()
		l.Consume()
	}
	return &sqlstmt.Limit{Count: count, Text: l.TextFromExclusive(start)}
}

// parseDmlRowLimitTail 解析 UPDATE / DELETE 的尾部：可选的 ORDER BY 与被作用行数限制。
//
// ORDER BY 在 DML 上只决定「哪些行被作用」，本身无消费方（UpdateStmt/DeleteStmt 不持有排序字段），
// 因此按表达式跳过而不是建模——但必须消费掉，否则其后紧跟的 LIMIT 会被漏掉，
// 采集层的行限制守卫随之失效（SQLite 启用 UPDATE_DELETE_LIMIT、达梦 MySQL 兼容模式都会出现该形态）。
func (l *Lexer) parseDmlRowLimitTail() *sqlstmt.Limit {
	if l.Current().IsKeyword("ORDER") {
		l.Consume()
		l.ExpectValue("BY")
		l.SkipOrderByExpr()
	}
	return l.parseRowLimitTail()
}

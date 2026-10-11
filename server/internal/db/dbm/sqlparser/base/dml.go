package base

// DML（INSERT/UPDATE/DELETE）共享解析片段。
//
// 各方言真正不同的只有「冲突处理子句」与「多表写法」，留在方言侧；
// 列名清单、SET 赋值、数据来源（VALUES/SELECT）、WHERE 与尾部行限制/RETURNING 都是同一份语义，
// 在此实现一次。子句-span 文本统一用 TextFromExclusive（不含下一个未消费 token），
// 避免旧实现里 TextFrom 与 TextFromExclusive 混用导致「文本多带一个词」。

import (
	"strings"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// parseColumnList 解析括号包裹的列名清单，如 INSERT 的 (col1, col2)。
// 接受标识符与该方言下以字符串形态出现的引用名（pg/Oracle 系双引号、mysql ANSI_QUOTES）
func (l *Lexer) parseColumnList() []string {
	columns := []string{}
	if l.Current().Value != "(" {
		return columns
	}
	l.Consume() // (
	for !l.Current().IsEOF() && l.Current().Value != ")" {
		if l.Current().Value == "," {
			l.Consume()
			continue
		}
		if l.IsUsableName() {
			columns = append(columns, l.Unquote(l.Consume().Value))
			continue
		}
		l.Consume() // 列位上出现非常规 token（罕见畸形形态）：消费以保持进展
	}
	l.ExpectValue(")")
	return columns
}

// parseInsertSource 解析 INSERT 的数据来源：VALUES 字面量行，或 SELECT 子查询。
//
// 返回 nil 表示「来源不是可确定求值的纯字面量」（SELECT 来源，或 VALUES 含函数/表达式/子查询/占位符），
// 这些形态的最终写入值只有执行后才知道，采集层据此放弃生成 INSERT 回滚而不是猜。
func (l *Lexer) parseInsertSource() []sqlstmt.InsertValueRow {
	if l.Current().IsKeyword("VALUES") {
		l.Consume()
		return l.ParseInsertValueRows()
	}
	if l.Current().IsKeyword("SELECT") {
		// INSERT INTO ... SELECT ...：整段消费，来源值不在此判定
		l.parseSelectStatement()
	}
	return nil
}

// parseSetAssignments 解析 UPDATE 的 SET 赋值列表，遇 WHERE 或语句结束停止
func (l *Lexer) parseSetAssignments() []sqlstmt.Assignment {
	assignments := make([]sqlstmt.Assignment, 0)
	if !l.Current().IsKeyword("SET") {
		return assignments
	}
	l.Consume()
	for !l.Current().IsEOF() && !l.IsExprEnd() {
		before := l.Pos
		assign := l.parseAssignment()
		if assign != nil {
			assignments = append(assignments, *assign)
		}
		if l.Current().Value == "," {
			l.Consume()
			continue
		}
		// 解析不出赋值（如畸形 SQL `SET a` 缺 '='，parseAssignment 已把位置复位）：
		// 跳过一个 token 保证进展，否则本循环会原地复位无限循环
		if assign == nil && l.Pos == before {
			l.Consume()
		}
	}
	return assignments
}

// parseAssignment 解析一个 SET 赋值：column = expr。解析不出 '=' 时复位位置并返回 nil。
func (l *Lexer) parseAssignment() *sqlstmt.Assignment {
	start := l.Pos
	var colText strings.Builder
	for !l.Current().IsEOF() && l.Current().Value != "=" && l.Current().Value != "," &&
		!l.Current().IsKeyword("WHERE") && l.Current().Value != ";" {
		colText.WriteString(l.Consume().Value)
	}
	if l.Current().Value != "=" {
		l.Pos = start
		return nil
	}
	l.Consume() // =

	// 值的扫描在逗号、分号或下一个子句头部结束（IsExprEnd 覆盖 FROM/WHERE/ORDER/LIMIT/RETURNING 等）；
	// 括号内的子查询整体跳过，所以其中的 FROM 等词不会被误判为边界
	valStart := l.Pos
	for !l.Current().IsEOF() && l.Current().Value != "," && !l.IsExprEnd() {
		if l.Current().Value == "(" {
			l.SkipParentheses()
			continue
		}
		l.Consume()
	}

	return &sqlstmt.Assignment{
		Column: l.Unquote(strings.TrimSpace(colText.String())),
		Value:  &sqlstmt.Expr{Text: l.TextFromExclusive(valStart)},
		Text:   l.TextFromExclusive(start),
	}
}

// isDmlTableListEnd 判断 DML 前置表引用列表是否结束：SET/WHERE 是后续子句头，
// FROM/USING 是「目标表列表 → 来源表列表」的分界（mysql 多表 DELETE 与 pg 多表 UPDATE 都用得到）
func (l *Lexer) isDmlTableListEnd() bool {
	tok := l.Current()
	return tok.IsKeyword("SET", "WHERE", "FROM", "USING") || tok.Value == ";"
}

// parseDmlTableRefs 解析 DML 的表引用列表：逗号交叉表与 JOIN 的操作数都计入。
// JOIN 条件不建模，但被连接进来的表必须一并登记——消费方（回滚采集层）据此判定「涉及多表」并 fail-closed；
// 只登记第一个表会让 `UPDATE t1 a JOIN t2 b SET b.z = 1` 按 t1 的列去生成回滚（改错表的列）
func (l *Lexer) parseDmlTableRefs() []sqlstmt.TableRef {
	tables := make([]sqlstmt.TableRef, 0)
	for !l.Current().IsEOF() {
		if l.isDmlTableListEnd() {
			break
		}
		if l.Current().Value == "," {
			l.Consume()
			continue
		}
		if l.IsJoinStart() || l.Current().IsKeyword("JOIN") {
			joinStart := l.Pos
			tables = append(tables, l.parseJoinTableRefs()...)
			if l.Pos == joinStart {
				break // 起始 token 解析不出 JOIN 且位置无进展（如 `UPDATE t1 JOIN SET ...`），退出防死循环
			}
			continue
		}
		before := l.Pos
		ref := l.parseTableRef()
		if ref.Name == "" {
			if l.Pos == before {
				break // 无进展：当前 token 不是表引用（如子句关键字），退出防死循环
			}
			continue
		}
		tables = append(tables, ref)
	}
	return tables
}

// parseJoinTableRefs 解析连续的 JOIN 链并只取其中的表引用（DML 场景不建模连接条件）
func (l *Lexer) parseJoinTableRefs() []sqlstmt.TableRef {
	var tables []sqlstmt.TableRef
	for _, join := range l.parseJoinClauses() {
		if join.Table.Name != "" {
			tables = append(tables, join.Table)
		}
	}
	return tables
}

// ParseUpdate 解析 UPDATE：表列表（可含 JOIN）+ SET + 可选 FROM 来源表 + WHERE + 尾部子句。
//
// 覆盖 mysql 的多表写法（UPDATE t1 JOIN t2 SET ...）与 pg/SQLite 的来源写法（UPDATE t SET ... FROM s WHERE ...）：
// 两种形态的涉及表都会全部登记，因此采集层统一按「不是单表」fail-closed，
// 而不是靠「没解析出来」碰巧跳过
func (l *Lexer) ParseUpdate() sqlstmt.Stmt {
	start := l.Pos
	l.Consume() // UPDATE
	// 语句级修饰词只影响调度或约束错误处理，不改变被作用的行集，消费即可
	for l.Current().IsKeyword(statementModifiers...) {
		l.Consume()
	}

	tables := l.parseDmlTableRefs()
	assignments := l.parseSetAssignments()

	if l.Current().IsKeyword("FROM") {
		// pg/SQLite 的多表更新来源表
		l.Consume()
		tables = append(tables, l.parseFromTables()...)
		tables = append(tables, l.parseJoinTableRefs()...)
	}

	where := l.parseDmlCond()
	limit := l.parseDmlRowLimitTail()
	l.ParseReturning()

	return &sqlstmt.UpdateStmt{
		Base:   sqlstmt.Base{Text: l.TextFrom(start)},
		Tables: tables,
		Set:    assignments,
		Where:  where,
		Limit:  limit,
	}
}

// ParseDelete 解析 DELETE：
//
//	DELETE [FROM] 目标表列表 [FROM 来源表列表] [USING 来源表列表] [WHERE] [ORDER BY] [行限制] [RETURNING]
//
// 「目标 + 来源」两段都登记进 Tables，多表 DELETE 因此是显式的「涉及多表」而非解析失败的副产品：
// 按 WHERE 回查只能还原 WHERE 命中的行，无法还原被连接条件筛掉的那部分
func (l *Lexer) ParseDelete() sqlstmt.Stmt {
	start := l.Pos
	l.Consume() // DELETE
	for l.Current().IsKeyword(statementModifiers...) {
		l.Consume()
	}
	if l.Current().IsKeyword("FROM") {
		l.Consume()
	}

	tables := l.parseDmlTableRefs()
	if l.Current().IsKeyword("FROM") {
		l.Consume()
		tables = append(tables, l.parseDmlTableRefs()...)
	}
	if l.Current().IsKeyword("USING") {
		l.Consume()
		tables = append(tables, l.parseDmlTableRefs()...)
	}

	where := l.parseDmlCond()
	limit := l.parseDmlRowLimitTail()
	l.ParseReturning()

	return &sqlstmt.DeleteStmt{
		Base:   sqlstmt.Base{Text: l.TextFrom(start)},
		Tables: tables,
		Where:  where,
		Limit:  limit,
	}
}

// ParseInsertHead 解析 INSERT 的公共头部：[INTO] 表 (列清单) 数据来源。
// 冲突处理子句（ON DUPLICATE KEY UPDATE / ON CONFLICT / REPLACE / IGNORE）各方言不同，
// 由方言在拿到本骨架结果后补写 Base 与 HasConflictClause
// parseInsert 解析不带冲突处理子句的 INSERT（含 RETURNING）。
// 有冲突子句的方言（mysql 的 REPLACE/IGNORE/ON DUPLICATE、pg 的 ON CONFLICT）
// 用 ParseInsertHead 自行组装并补 HasConflictClause
func (l *Lexer) parseInsert() sqlstmt.Stmt {
	start := l.Pos
	table, columns, valueRows := l.ParseInsertHead()
	l.ParseReturning()
	return &sqlstmt.InsertStmt{
		Base:      sqlstmt.Base{Text: l.TextFrom(start)},
		Table:     table,
		Columns:   columns,
		ValueRows: valueRows,
	}
}

func (l *Lexer) ParseInsertHead() (sqlstmt.TableRef, []string, []sqlstmt.InsertValueRow) {
	l.Consume() // INSERT | REPLACE
	// 语句级修饰词（mysql 的 INSERT DELAYED / INSERT IGNORE 等）必须在此消费，
	// 否则紧随其后的 INTO 与表引用会被这个关键字 token 打断；
	// 是否构成冲突处理形态由方言在调用前判定（骨架不代替方言解释修饰词的语义）
	for l.Current().IsKeyword(statementModifiers...) {
		l.Consume()
	}

	// Oracle/达梦的 INSERT ALL 一次写入多张表：不登记单一目标表，整段消费到语句结束。
	// INSERT 的回滚是按 VALUES 字面量删掉刚插入的行，对多表插入形态不成立，
	// 目标表留空即让采集层自然跳过插入快照（而不是猜一张表去删行）
	if l.Current().IsKeyword("ALL") {
		l.Consume()
		l.SkipToStatementEnd()
		return sqlstmt.TableRef{}, nil, nil
	}
	// SQLite 的 INSERT OR REPLACE / OR IGNORE / OR ROLLBACK 冲突解决子句：
	// 动作词在 SQLite 不是保留字（REPLACE 除外），故按词值匹配后随位；是否构成 upsert 由方言自行判定
	if l.Current().IsKeyword("OR") && l.PeekIsWord(1, "REPLACE", "IGNORE", "ROLLBACK", "ABORT", "FAIL") {
		l.Consume()
		l.Consume()
	}
	if l.Current().IsKeyword("INTO") {
		l.Consume()
	}
	tableRef := l.parseTableRef()
	columns := l.parseColumnList()
	return tableRef, columns, l.parseInsertSource()
}

// parseDmlCond 解析 DML 的 WHERE 条件；当前不是 WHERE 时返回 nil
func (l *Lexer) parseDmlCond() *sqlstmt.Expr {
	if !l.Current().IsKeyword("WHERE") {
		return nil
	}
	l.Consume()
	return l.ParseCondExpr(false)
}

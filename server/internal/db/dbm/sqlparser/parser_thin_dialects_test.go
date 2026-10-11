package sqlparser

// 「薄方言」（只登记词法能力 + 共用 base 语法骨架的方言）的解析覆盖：
// mssql / sqlite / clickhouse 曾直接复用 pgsql 解析器，等于用 pg 的词法能力表去读
// T-SQL 的 [方括号]、SQLite 的 [方括号]与反引号、ClickHouse 的反引号 —— 引用名在解析阶段
// 不被识别，表名/列名提取随之失真（脱敏血缘与回滚采集都读这些字段）。
//
// 本文件同时是「开闭原则」的验收：新增这三个方言没有改动任何一条子句解析代码，
// 通用语料在它们身上也必须完整消费。
import (
	"strings"
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/clickhouse"
	"mayfly-go/internal/db/dbm/sqlparser/mssql"
	"mayfly-go/internal/db/dbm/sqlparser/sqlite"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

func thinParsers() map[string]sqlparserParser {
	return map[string]sqlparserParser{
		"mssql":      mssql.NewParser(),
		"sqlite":     new(sqlite.SqliteParser),
		"clickhouse": new(clickhouse.ClickhouseParser),
	}
}

// TestThinDialectsConsumeUniversalCorpus 通用语料（未登记方言例外的条目）在薄方言上必须完整消费。
//
// 这些 SQL 都是标准形态，任何方言都原生支持；解析不出即为骨架缺陷而非方言差异
func TestThinDialectsConsumeUniversalCorpus(t *testing.T) {
	for dialect, parser := range thinParsers() {
		dialect, parser := dialect, parser
		t.Run(dialect, func(t *testing.T) {
			t.Parallel()
			for _, c := range snapshotCorpus {
				if scope, scoped := corpusScope[c.id]; scoped && !contains(scope, dialect) {
					continue // 该形态不在本方言的适用面内
				}
				stmt, err := parser.Parse(c.sql)
				if err != nil {
					t.Errorf("[%s/%s] 解析失败: %v", dialect, c.id, err)
					continue
				}
				if stmt == nil {
					t.Errorf("[%s/%s] 解析结果为空", dialect, c.id)
					continue
				}
				if got, want := normText(stmt.GetText()), strings.TrimSuffix(normText(c.sql), ";"); got != want {
					t.Errorf("[%s/%s] 语句未被完整消费\n got=%q\nwant=%q", dialect, c.id, got, want)
				}
			}
		})
	}
}

// TestMssqlNativeForms T-SQL 特有形态：方括号引用名、n 段限定名、TOP、WITH (NOLOCK)、SELECT INTO
func TestMssqlNativeForms(t *testing.T) {
	parser := mssql.NewParser()

	stmt, err := parser.Parse("SELECT [id], [name] AS [n] FROM [dbo].[users]")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	sel := stmt.(*sqlstmt.SelectStmt)
	if len(sel.Items) != 2 || sel.Items[0].ColumnName != "id" || sel.Items[1].Alias != "n" {
		t.Errorf("方括号列名/别名未被正确提取: %+v", sel.Items)
	}
	if len(sel.From) != 1 || sel.From[0].Name != "users" || sel.From[0].Schema != "dbo" {
		t.Errorf("方括号限定表名解析异常: %+v", sel.From)
	}

	// 服务器.库.架构.表 的四段名：表名取末段，前缀整体作为限定（采集层据此 fail-closed）
	stmt, _ = parser.Parse("SELECT [a] FROM [srv].[db].[dbo].[t]")
	sel = stmt.(*sqlstmt.SelectStmt)
	if len(sel.From) != 1 || sel.From[0].Name != "t" || sel.From[0].Schema != "srv.db.dbo" {
		t.Errorf("四段限定名解析异常: %+v", sel.From)
	}

	// TOP n 必须折算成行限制；WITH (NOLOCK) 不得被当成第二张表
	stmt, _ = parser.Parse("SELECT TOP 5 a FROM t WITH (NOLOCK) WHERE a > 1")
	sel = stmt.(*sqlstmt.SelectStmt)
	if sel.Limit == nil || sel.Limit.Count != 5 {
		t.Errorf("TOP 未折算为行限制: %+v", sel.Limit)
	}
	if len(sel.From) != 1 || sel.From[0].Name != "t" {
		t.Errorf("表提示被当成表引用: %+v", sel.From)
	}
	if sel.Where == nil || normText(sel.Where.Text) != "a > 1" {
		t.Errorf("提示之后的 WHERE 丢失: %+v", sel.Where)
	}

	// SELECT ... INTO 目标表：其后的 FROM/WHERE 不能因此丢失
	stmt, _ = parser.Parse("SELECT id INTO bak_users FROM users WHERE id > 1")
	sel = stmt.(*sqlstmt.SelectStmt)
	if len(sel.From) != 1 || sel.From[0].Name != "users" || sel.Where == nil {
		t.Errorf("SELECT INTO 之后的 FROM/WHERE 丢失: from=%+v where=%+v", sel.From, sel.Where)
	}

	// OFFSET-FETCH 与 ORDER BY 组合
	stmt, _ = parser.Parse("SELECT a FROM t ORDER BY a OFFSET 10 ROWS FETCH NEXT 20 ROWS ONLY")
	sel = stmt.(*sqlstmt.SelectStmt)
	if sel.Limit == nil || sel.Limit.Count != 20 || sel.Limit.Offset != 10 {
		t.Errorf("OFFSET-FETCH 未建模: %+v", sel.Limit)
	}
	if len(sel.OrderBy) != 1 || sel.OrderBy[0].Text != "a" {
		t.Errorf("ORDER BY 丢失: %+v", sel.OrderBy)
	}

	// 语句分类：T-SQL 的 DDL/DML 形态都要落到正确类型
	for sql, want := range map[string]sqlstmt.StmtType{
		"INSERT INTO t ([a]) VALUES (1)":     sqlstmt.StmtTypeInsert,
		"UPDATE t SET [a] = 1 WHERE [a] = 2": sqlstmt.StmtTypeUpdate,
		"DELETE FROM t WHERE [a] = 2":        sqlstmt.StmtTypeDelete,
		"SELECT a FROM t":                    sqlstmt.StmtTypeSelect,
	} {
		got, err := parser.ClassifyStmt(sql)
		if err != nil {
			t.Errorf("[%s] 分类失败: %v", sql, err)
			continue
		}
		if got != want {
			t.Errorf("[%s] 分类=%v 期望%v", sql, got, want)
		}
	}
}

// TestSqliteNativeForms SQLite 特有形态：三种引用符、INSERT OR 冲突动作、ON CONFLICT、RETURNING
func TestSqliteNativeForms(t *testing.T) {
	parser := new(sqlite.SqliteParser)

	stmt, err := parser.Parse("SELECT `a`, [b], \"c\" FROM `t`")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	sel := stmt.(*sqlstmt.SelectStmt)
	if len(sel.Items) != 3 {
		t.Fatalf("三种引用符的列项数=%d 期望3: %+v", len(sel.Items), sel.Items)
	}
	for i, want := range []string{"a", "b", "c"} {
		if sel.Items[i].ColumnName != want {
			t.Errorf("列项#%d ColumnName=%q 期望%q", i, sel.Items[i].ColumnName, want)
		}
	}
	if len(sel.From) != 1 || sel.From[0].Name != "t" {
		t.Errorf("反引号表名解析异常: %+v", sel.From)
	}

	// INSERT OR REPLACE / OR IGNORE：与 pg 的 ON CONFLICT 同属 upsert，必须标记冲突形态
	for _, sql := range []string{
		"INSERT OR REPLACE INTO t (a) VALUES (1)",
		"INSERT OR IGNORE INTO t (a) VALUES (1)",
		"INSERT INTO t (a) VALUES (1) ON CONFLICT (a) DO NOTHING",
	} {
		got, err := parser.Parse(sql)
		if err != nil {
			t.Fatalf("[%s] 解析失败: %v", sql, err)
		}
		ins := got.(*sqlstmt.InsertStmt)
		if !ins.HasConflictClause {
			t.Errorf("[%s] 冲突处理形态未标记（回滚的 DELETE 会删掉既有行）", sql)
		}
		if normText(ins.GetText()) != normText(sql) {
			t.Errorf("[%s] 未被完整消费: %q", sql, ins.GetText())
		}
	}

	// UPDATE ... LIMIT：采集层的行限制守卫依赖它
	got, _ := parser.Parse("UPDATE t SET a = 1 WHERE st = 1 ORDER BY id LIMIT 3")
	up := got.(*sqlstmt.UpdateStmt)
	if up.Limit == nil || up.Limit.Count != 3 {
		t.Errorf("SQLite 的 UPDATE…ORDER BY…LIMIT 未捕获行限制: %+v", up.Limit)
	}
	if len(up.Set) != 1 || up.Set[0].Column != "a" {
		t.Errorf("SET 赋值异常: %+v", up.Set)
	}
}

// TestClickhouseNativeForms ClickHouse 特有形态：反引号引用名与 LIMIT o, c
func TestClickhouseNativeForms(t *testing.T) {
	parser := new(clickhouse.ClickhouseParser)

	stmt, err := parser.Parse("SELECT `a`, count() AS `c` FROM `t` WHERE `a` > 1 GROUP BY `a`")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	sel := stmt.(*sqlstmt.SelectStmt)
	if len(sel.Items) != 2 || sel.Items[0].ColumnName != "a" {
		t.Errorf("反引号列项解析异常: %+v", sel.Items)
	}
	if len(sel.From) != 1 || sel.From[0].Name != "t" {
		t.Errorf("反引号表名解析异常: %+v", sel.From)
	}
	// 条件/分组文本按契约保留原文（含引用符）：WHERE 是逐字回放的，改写字面会换语义；
	// 需要裸名的消费方读 ColumnName/TableAlias
	if sel.Where == nil || normText(sel.Where.Text) != "`a` > 1" {
		t.Errorf("反引号条件解析异常: %+v", sel.Where)
	}
	if len(sel.GroupBy) != 1 || sel.GroupBy[0] != "`a`" {
		t.Errorf("GROUP BY 回填异常: %+v", sel.GroupBy)
	}

	// ClickHouse 原生 LIMIT offset, count
	stmt, _ = parser.Parse("SELECT a FROM t LIMIT 10, 20")
	sel = stmt.(*sqlstmt.SelectStmt)
	if sel.Limit == nil || sel.Limit.Offset != 10 || sel.Limit.Count != 20 {
		t.Errorf("ClickHouse 的 LIMIT o,c 解析异常: %+v", sel.Limit)
	}
}

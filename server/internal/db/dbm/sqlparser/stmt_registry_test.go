package sqlparser_test

import (
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/mysql"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
)

// TestSharedStmtRegistry 注册表的完整性：每种共享语句都能分发解析，且策略选项覆盖
// 执行层会产出的全部语义类型。
//
// 这是「新增语句 = 注册 1 处」的护栏：注册项漏了 Match/Parse 会在这里直接失败，
// 而不是等到某条 SQL 落进兜底分支（那会走写通道，虽然安全但体验异常）。
func TestSharedStmtRegistry(t *testing.T) {
	// 名称顺序即分发优先级，且不应重复
	names := sqlparser.SharedStatementNames()
	if len(names) < 8 {
		t.Fatalf("共享语句注册表过小: %v", names)
	}
	seen := map[string]bool{}
	for _, n := range names {
		if seen[n] {
			t.Fatalf("注册表出现重复语句名: %s", n)
		}
		seen[n] = true
	}

	// 每种语句一个最小样例：必须能分发到对应解析函数（不被方言兜底成 OtherStmt）
	parser := new(mysql.MysqlParser)
	splitter := sqlparser.NewSplitter(tokenizer.MysqlConfig)
	cases := map[string]string{
		"select":  "SELECT 1",
		"insert":  "INSERT INTO t (id) VALUES (1)",
		"update":  "UPDATE t SET a = 1",
		"delete":  "DELETE FROM t WHERE id = 1",
		"merge":   "MERGE INTO t USING s ON (t.id = s.id) WHEN MATCHED THEN UPDATE SET t.a = 1",
		"explain": "EXPLAIN SELECT 1",
		"create":  "CREATE TABLE t (id INT)",
		"drop":    "DROP TABLE t",
		"alter":   "ALTER TABLE t ADD COLUMN c INT",
		"with":    "WITH c AS (SELECT 1) SELECT * FROM c",
		"ddl":     "TRUNCATE TABLE t",
	}
	for name, sql := range cases {
		stmt, err := parser.Parse(sql)
		if err != nil || stmt == nil {
			t.Fatalf("%s 样例解析失败: %q err=%v", name, sql, err)
		}
		typ := sqlparser.Classify(parser, splitter, sql, stmt, err)
		if typ == "" {
			t.Fatalf("%s 样例分类为空: %q", name, sql)
		}
	}

	// 策略选项必须覆盖执行层全部语义类型（否则策略无法针对该类型配置）
	policy := sqlparser.PolicyStatementTypes()
	required := []string{"select", "insert", "update", "delete", "merge", "ddl", "read", "other-read", "other-write"}
	has := map[string]bool{}
	for _, p := range policy {
		has[p] = true
	}
	for _, r := range required {
		if !has[r] {
			t.Fatalf("策略选项缺少 %q，实际: %v", r, policy)
		}
	}
}

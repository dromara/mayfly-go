//go:build it

package itest

// MERGE（upsert）真机集成测试：验证两件事，缺一不可——
//  1. 真实方言解析器 + sqlparser.Classify 把该方言的 MERGE 判成写类型 merge（不是 other/select），
//     否则它会走只读通道、绕过 DML 审批；
//  2. 真库能执行该 MERGE，且经通用写通道（doMerge→doExec 即 conn.Exec）后合并效果正确
//     （匹配行更新、未匹配行插入）。
//
// pg（必装）与 mssql（容器可选，不可达 SkipOrRequire 跳过）语法差异大，各写各的。
// 运行：cd server && go test -tags it -count=1 -run TestITMerge ./internal/db/dbm/itest/

import (
	"testing"

	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// assertMergeClassified 用该连接真实方言的解析器判定 MERGE 必须归为写类型 merge。
func assertMergeClassified(t *testing.T, conn *dbi.DbConn, mergeSQL string) {
	t.Helper()
	parser := conn.GetDialect().GetSQLParser()
	splitter := conn.GetDialect().GetSQLSplitter()
	stmt, parseErr := parser.Parse(mergeSQL)
	class := sqlparser.Classify(parser, splitter, mergeSQL, stmt, parseErr)
	require.Equalf(t, sqlstmt.StmtTypeMerge, class, "真实方言解析器应把 MERGE 判为 merge，实际=%q", class)
}

// assertMergeEffect 断言 MERGE 后目标表：id=1 被更新为 'new'，id=2 被插入为 'ins'。
func assertMergeEffect(t *testing.T, conn *dbi.DbConn) {
	t.Helper()
	rows := readAllRows(t, conn, "it_merge_tgt", "id")
	require.Len(t, rows, 2, "MERGE 后目标表应有 2 行（1 更新 + 1 插入）")
	assertCell(t, "name", "new", rows[0]["name"])
	assertCell(t, "name", "ins", rows[1]["name"])
}

func TestITMergePostgres(t *testing.T) {
	conn := pgConn(t)
	defer conn.Close()
	defer func() { _, _ = conn.Exec("DROP TABLE IF EXISTS it_merge_tgt") }()

	mustExec(t, conn, "DROP TABLE IF EXISTS it_merge_tgt")
	mustExec(t, conn, "CREATE TABLE it_merge_tgt (id int PRIMARY KEY, name varchar(32))")
	mustExec(t, conn, "INSERT INTO it_merge_tgt VALUES (1, 'old')")

	// pg15+ MERGE：源用 VALUES 行构造，SET 目标列不加限定符
	mergeSQL := `MERGE INTO it_merge_tgt t USING (VALUES (1, 'new'), (2, 'ins')) AS s(id, name) ON t.id = s.id WHEN MATCHED THEN UPDATE SET name = s.name WHEN NOT MATCHED THEN INSERT (id, name) VALUES (s.id, s.name)`

	assertMergeClassified(t, conn, mergeSQL)

	// 断言以「合并后的实际行数据」为准，而非 RowsAffected：
	// pg 驱动对 MERGE 的命令标签不回填影响行数（返回 0），据它断言会假失败；真效果由 assertMergeEffect 兜底
	_, err := conn.Exec(mergeSQL)
	require.NoError(t, err, "MERGE 应能经通用写通道执行")
	assertMergeEffect(t, conn)
}

func TestITMergeMssql(t *testing.T) {
	conn := gcMssqlConn(t) // 容器不可达则 SkipOrRequire 跳过
	defer conn.Close()
	defer func() {
		_, _ = conn.Exec("DROP TABLE IF EXISTS it_merge_tgt")
		_, _ = conn.Exec("DROP TABLE IF EXISTS it_merge_src")
	}()

	mustExec(t, conn, "DROP TABLE IF EXISTS it_merge_tgt")
	mustExec(t, conn, "DROP TABLE IF EXISTS it_merge_src")
	mustExec(t, conn, "CREATE TABLE it_merge_tgt (id int PRIMARY KEY, name nvarchar(32))")
	mustExec(t, conn, "CREATE TABLE it_merge_src (id int, name nvarchar(32))")
	mustExec(t, conn, "INSERT INTO it_merge_tgt VALUES (1, N'old')")
	mustExec(t, conn, "INSERT INTO it_merge_src VALUES (1, N'new'), (2, N'ins')")

	// T-SQL MERGE：SET 目标列用别名限定，且语句必须以分号结尾
	mergeSQL := "MERGE INTO it_merge_tgt AS t USING it_merge_src AS s ON t.id = s.id WHEN MATCHED THEN UPDATE SET t.name = s.name WHEN NOT MATCHED THEN INSERT (id, name) VALUES (s.id, s.name);"

	assertMergeClassified(t, conn, mergeSQL)
	mustExec(t, conn, mergeSQL)
	assertMergeEffect(t, conn)
}

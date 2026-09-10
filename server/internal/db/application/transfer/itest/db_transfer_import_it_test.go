package itest

// 迁移导入路径（ImportDumpStream）真实链路集成测试：
// 注：连接底座 transferTestConn 已上移至 it_env_it_test.go（IT 公共底座）。
// 验证事务控制语句过滤、批级提交（500语句/批）、失败回滚语义在真实库（mysql/pg）上的行为。
//
// 运行方式：cd server && go test -tags it -count=1 -v ./internal/db/application/

import (
	"mayfly-go/internal/db/application/transfer"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

// buildDumpScript 构造模拟dump产物：DDL + BEGIN/COMMIT包装 + n条INSERT（含事务控制语句穿插）
func buildDumpScript(quote func(string) string, table string, n int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", quote(table)))
	sb.WriteString(fmt.Sprintf("CREATE TABLE %s (id int PRIMARY KEY, val varchar(64));\n", quote(table)))
	sb.WriteString("BEGIN;\n")
	for i := 1; i <= n; i++ {
		sb.WriteString(fmt.Sprintf("INSERT INTO %s (id, val) VALUES (%d, 'v%d');\n", quote(table), i, i))
		// 每200行穿插一条事务控制语句，验证全部被过滤且不影响导入
		if i%200 == 0 {
			sb.WriteString("set autocommit=1;\n")
		}
	}
	sb.WriteString("COMMIT;\n")
	return sb.String()
}

func countRows(t *testing.T, conn *dbi.DbConn, quote func(string) string, table string) int {
	t.Helper()
	_, rows, err := conn.Query(fmt.Sprintf("SELECT COUNT(*) AS cnt FROM %s", quote(table)))
	require.NoError(t, err)
	return int(rows[0]["cnt"].(int64))
}

// TestITImportDumpStream_PgTarget pg目标：过滤BEGIN/COMMIT/SET autocommit + 批级提交 + 全量数据落库
func TestITImportDumpStream_PgTarget(t *testing.T) {
	conn := transferTestConn(t, &dbi.DbInfo{Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it"})
	defer conn.Close()
	quote := conn.GetDialect().Quoter().Quote
	table := "it_import_pg_target"
	_, err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table)))
	require.NoError(t, err)

	app := &transfer.DbTransferAppImpl{}
	script := buildDumpScript(quote, table, 600) // 600条 > 500：至少触发一次批级提交
	require.NoError(t, app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader(script)))

	assert.Equal(t, 600, countRows(t, conn, quote, table), "BEGIN/COMMIT/autocommit应被过滤，600行数据全部落库")
}

// TestITImportDumpStream_MysqlTarget mysql目标：同构迁移链路回归（原实现在显式事务内执行BEGIN会隐式提交）
func TestITImportDumpStream_MysqlTarget(t *testing.T) {
	conn := transferTestConn(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"})
	defer conn.Close()
	quote := conn.GetDialect().Quoter().Quote
	table := "it_import_mysql_target"
	_, err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table)))
	require.NoError(t, err)

	app := &transfer.DbTransferAppImpl{}
	script := buildDumpScript(quote, table, 300)
	require.NoError(t, app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader(script)))

	assert.Equal(t, 300, countRows(t, conn, quote, table))
}

// TestITImportDumpStream_BatchCommitOnFailure 批级提交语义：失败语句之前的完整批次已提交，当前批次回滚
func TestITImportDumpStream_BatchCommitOnFailure(t *testing.T) {
	conn := transferTestConn(t, &dbi.DbInfo{Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it"})
	defer conn.Close()
	quote := conn.GetDialect().Quoter().Quote
	table := "it_import_batch_fail"
	_, err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table)))
	require.NoError(t, err)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE TABLE %s (id int PRIMARY KEY, val varchar(64));\n", quote(table)))
	for i := 1; i <= 600; i++ {
		sb.WriteString(fmt.Sprintf("INSERT INTO %s (id, val) VALUES (%d, 'v%d');\n", quote(table), i, i))
	}
	// 注意：CREATE TABLE 也计入批内语句数（第1个槽位），故首批提交的INSERT为 500-1=499 条；
	// 第601条语句（坏语句）失败后，第2批（INSERT 500~600）整体回滚
	sb.WriteString("INSERT INTO it_table_not_exist VALUES (1);\n")

	app := &transfer.DbTransferAppImpl{}
	err = app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader(sb.String()))
	require.Error(t, err)

	assert.Equal(t, 499, countRows(t, conn, quote, table), "失败前完整批次应已提交（DDL占1槽故为499条INSERT），当前批次回滚")
}

// TestITImportDumpStream_EmptyStream 空流与纯控制语句流安全
func TestITImportDumpStream_EmptyStream(t *testing.T) {
	conn := transferTestConn(t, &dbi.DbInfo{Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it"})
	defer conn.Close()

	app := &transfer.DbTransferAppImpl{}
	require.NoError(t, app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader("")))
	require.NoError(t, app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader("BEGIN;\nCOMMIT;\nset autocommit=1;\n")))
}

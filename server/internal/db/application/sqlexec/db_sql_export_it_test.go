//go:build it

package sqlexec

// 查询结果集全量导出与截断标记的真实数据库集成测试。
//
// 钉的是两件「文件看起来正常、内容却少了」的事：
//  1. 页面结果被行数上限截断时，响应必须带 truncated 标记——否则用户分不清
//     「这张表只有 10 行」与「库里还有 1 万行没显示」，还会把截断结果导出去；
//  2. 全量导出真的取全：一次导出 1200 行时 CSV 必须有 1200 条记录，
//     行数按「记录起始模式」数而不是 split('\n')（字段里可以含换行，那样会多数）。
//
// 另外钉住导出入口不得成为写语句的旁路：给一条 UPDATE 必须被拒，且数据一个字都不能变。
//
// 用例中的表名/列名全部是本文件常量并走方言 Quoter，不接受任何外部输入，拼字符串不引入注入面。
//
// 运行：cd server && go test -tags it -count=1 -run TestITQueryExport ./internal/db/application/

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/domain/entity"
	flowentity "mayfly-go/internal/flow/domain/entity"
)

const exportTable = "it_query_export"

// exportRowPattern CSV 记录起始：id + 名称，字段内含换行时也不会被误计
var exportRowPattern = regexp.MustCompile(`(?m)^[0-9]+,n[0-9]+`)

// writerTo 把 bytes.Buffer 包成延迟写入器：产物写到第一字节才算「真的开始导出」
func writerTo(buf *bytes.Buffer) OpenWriter {
	return func() (io.Writer, error) { return buf, nil }
}

// exportApp 导出用例的应用实例：策略闸门需要流程定义查询能力，
// 裸构造的应用里该依赖为 nil，会直接 panic（生产由 IOC 注入）
func exportApp() *DbSQLExecAppImpl {
	return &DbSQLExecAppImpl{flowProcdefApp: fakeProcdefReader{}}
}

// exportPrepare 建表并写入 count 行
func exportPrepare(t *testing.T, conn *dbi.DbConn, count int) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	if _, err := conn.Exec("DROP TABLE IF EXISTS " + quote(exportTable)); err != nil {
		t.Fatalf("drop failed: %s", err.Error())
	}
	if _, err := conn.Exec(fmt.Sprintf("CREATE TABLE %s (%s INT PRIMARY KEY, %s VARCHAR(64))", quote(exportTable), quote("id"), quote("name"))); err != nil {
		t.Fatalf("create failed: %s", err.Error())
	}
	for i := 1; i <= count; i++ {
		if _, err := conn.Exec(fmt.Sprintf("INSERT INTO %s (%s, %s) VALUES (%d, 'n%d')", quote(exportTable), quote("id"), quote("name"), i, i)); err != nil {
			t.Fatalf("insert failed at %d: %s", i, err.Error())
		}
	}
}

func exportSql(conn *dbi.DbConn, action string) string {
	quote := conn.GetDialect().Quoter().QuoteIdent
	return fmt.Sprintf("%s FROM %s", action, quote(exportTable))
}

func exportParams(conn *dbi.DbConn, sql string, stmt sqlstmt.Stmt) *sqlExecParam {
	record := new(entity.DbSQLExec)
	record.DbId = conn.Info.Id
	record.Db = conn.Info.Database
	record.SQL = sql
	record.Status = entity.DbSQLExecStatusSuccess
	return &sqlExecParam{DbConn: conn, SQL: sql, Stmt: stmt, SQLExecRecord: record}
}

func TestITQueryExportGetsAllRows(t *testing.T) {
	for _, node := range appDialectNodes[:2] {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			const total = 1200
			exportPrepare(t, conn, total)

			var buf bytes.Buffer
			require.NoError(t, exportQuery(context.Background(), exportApp(), conn, &dto.QueryExportReq{
				DbId: 1, Db: conn.Info.Database, SQL: exportSql(conn, "SELECT id, name"), Format: "csv",
			}, writerTo(&buf)))

			// 记录数按 CSV 记录的起始模式数，不能用 split('\n')：字段里含换行会把一条记录数成多条
			records := exportRowPattern.FindAllString(buf.String(), -1)
			assert.Len(t, records, total, "全量导出必须拿到全部行，产物前 200 字：%s", firstRunes(buf.String(), 200))
			assert.Contains(t, buf.String(), "name", "CSV 必须带表头")
			assert.Contains(t, buf.String(), fmt.Sprintf("%d,n%d", total, total), "最后一行必须在产物里")
		})
	}
}

// TestITQueryExportRefusesWriteStatement 导出入口不得成为写语句的旁路。
//
// 导出会重新执行这条语句：允许 UPDATE 就等于给了一条不经过策略确认、不显示受影响行数的改数据通道
func TestITQueryExportRefusesWriteStatement(t *testing.T) {
	for _, node := range appDialectNodes[:2] {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			exportPrepare(t, conn, 3)

			quote := conn.GetDialect().Quoter().QuoteIdent
			var buf bytes.Buffer
			err := exportQuery(context.Background(), exportApp(), conn, &dto.QueryExportReq{
				DbId:   1,
				Db:     conn.Info.Database,
				SQL:    fmt.Sprintf("UPDATE %s SET %s = 'hacked' WHERE %s = 1", quote(exportTable), quote("name"), quote("id")),
				Format: "csv",
			}, writerTo(&buf))
			require.Error(t, err, "写语句必须被拒绝")
			assert.Empty(t, buf.String(), "拒绝时不得写出任何产物")

			_, rows, queryErr := conn.QueryContext(context.Background(), "SELECT "+quote("name")+" FROM "+quote(exportTable)+" WHERE "+quote("id")+" = 1")
			require.NoError(t, queryErr)
			require.Len(t, rows, 1)
			assert.Equal(t, "n1", fmt.Sprint(rows[0]["name"]), "被拒绝的导出绝不能真的改到数据")
		})
	}
}

// TestITQueryExportScriptFormatRejected SQL 脚本格式不适用于任意查询结果：
// 生成 INSERT 需要知道目标表，拿库名或语句猜一个都会生成一份「看着能跑」的错脚本
func TestITQueryExportScriptFormatRejected(t *testing.T) {
	conn := appDialectNodes[0].conn(t)
	defer conn.Close()
	exportPrepare(t, conn, 2)

	var buf bytes.Buffer
	err := exportQuery(context.Background(), exportApp(), conn, &dto.QueryExportReq{
		DbId: 1, Db: conn.Info.Database, SQL: exportSql(conn, "SELECT id, name"), Format: "sql",
	}, writerTo(&buf))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "csv", "拒绝时必须给出可用格式清单")
	assert.Empty(t, buf.String())

	// 未知格式同样拒绝，而不是回退成某个默认格式：静默换格式会让用户拿错产物
	err = exportQuery(context.Background(), exportApp(), conn, &dto.QueryExportReq{
		DbId: 1, Db: conn.Info.Database, SQL: exportSql(conn, "SELECT id, name"), Format: "xlsx",
	}, writerTo(&buf))
	require.Error(t, err)
}

// TestITQueryExportEmptyResult 空结果也要出一份格式合法的产物：
// 零字节文件与「查询确实没结果」在下载后长得一样，用户只能猜。
// 表头拿不到是已知限制：列信息要等第一行，零行时无人可问，这里只要求产物结构合法
func TestITQueryExportEmptyResult(t *testing.T) {
	conn := appDialectNodes[0].conn(t)
	defer conn.Close()
	exportPrepare(t, conn, 2)
	quote := conn.GetDialect().Quoter().QuoteIdent

	var jsonBuf bytes.Buffer
	require.NoError(t, exportQuery(context.Background(), exportApp(), conn, &dto.QueryExportReq{
		DbId:   1,
		Db:     conn.Info.Database,
		SQL:    fmt.Sprintf("SELECT id, name FROM %s WHERE %s = 999999", quote(exportTable), quote("id")),
		Format: "json",
	}, writerTo(&jsonBuf)))
	// JSON 产物必须能被解析（0 字节文件不行）：这是「空结果」与「导出坏了」的唯一区别
	assert.JSONEq(t, "[]", strings.TrimSpace(jsonBuf.String()), "空结果的 JSON 产物必须是合法空数组")
}

// TestITQueryResultTruncatedFlag 页面结果的截断标记必须给出。
//
// 拿 len(res)==上限 去判断不可靠：正好那么行的完整结果会被误报成截断（反向则是漏报），
// 而用户按这个标记决定要不要去导全量
func TestITQueryResultTruncatedFlag(t *testing.T) {
	for _, node := range appDialectNodes[:2] {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			exportPrepare(t, conn, 20)

			app := exportApp()
			quote := conn.GetDialect().Quoter().QuoteIdent
			sql := exportSql(conn, "SELECT id, name")

			cut, err := app.doQuery(context.Background(), exportParams(conn, sql, &sqlstmt.SelectStmt{}), 10)
			require.NoError(t, err)
			assert.Len(t, cut.Res, 10)
			assert.True(t, cut.Truncated, "还有数据没取完时必须标截断")
			assert.Equal(t, 10, cut.RowLimit)

			small := fmt.Sprintf("%s WHERE %s <= 3", sql, quote("id"))
			exact, err := app.doQuery(context.Background(), exportParams(conn, small, &sqlstmt.SelectStmt{}), 3)
			require.NoError(t, err)
			assert.Len(t, exact.Res, 3)
			assert.False(t, exact.Truncated, "刚好取完的结果不该标成截断: %+v", exact.Truncated)

			all, err := app.doQuery(context.Background(), exportParams(conn, sql, &sqlstmt.SelectStmt{}), 0)
			require.NoError(t, err)
			assert.False(t, all.Truncated, "不限行数时不该有截断标记")
			assert.Len(t, all.Res, 20)
		})
	}
}

func firstRunes(text string, n int) string {
	runes := []rune(text)
	if len(runes) <= n {
		return text
	}
	return string(runes[:n])
}

// TestITQueryExportPassesTriggerPolicy 导出必须过与执行同一道策略闸门。
//
// 少了这道闸，「某些 SELECT 必须走工单」这类策略就有一个静默旁路：同一个查询在
// 执行入口被拦下，换成导出却能整表拉走，而且页面上一句"需要审批"都不会出现
func TestITQueryExportPassesTriggerPolicy(t *testing.T) {
	conn := appMysqlConn(t)
	defer conn.Close()
	exportPrepare(t, conn, 5)

	governed := &flowentity.Procdef{TriggerPolicy: &flowentity.TriggerPolicy{
		Version:         flowentity.TriggerPolicyVersion,
		DefaultSeverity: flowentity.SeverityRequired,
		Checks: []*flowentity.CheckConfig{{
			Key: dbCheckDmlRequiresApproval, BizType: DbSQLExecFlowBizType, Severity: flowentity.SeverityRequired,
			Params: map[string]any{"stmtTypes": []any{"select"}},
		}},
	}}
	app := &DbSQLExecAppImpl{flowProcdefApp: fakeProcdefReader{governed: governed}}

	var buf bytes.Buffer
	err := exportQuery(context.Background(), app, conn, &dto.QueryExportReq{
		DbId: 1, Db: conn.Info.Database, SQL: exportSql(conn, "SELECT id, name"), Format: "csv",
	}, writerTo(&buf))
	require.Error(t, err, "被策略要求审批的查询不得被导出绕过")
	assert.Empty(t, buf.String(), "拒绝时不得写出任何产物")
	assert.Contains(t, err.Error(), "审批")

	// 同一道策略下，未要求审批的导出正常完成（证明上面那条不是恒假）
	free := &DbSQLExecAppImpl{flowProcdefApp: fakeProcdefReader{}}
	var okBuf bytes.Buffer
	require.NoError(t, exportQuery(context.Background(), free, conn, &dto.QueryExportReq{
		DbId: 1, Db: conn.Info.Database, SQL: exportSql(conn, "SELECT id, name"), Format: "csv",
	}, writerTo(&okBuf)))
	assert.Contains(t, okBuf.String(), "n1")
}

// TestITQueryExportKeepsDuplicateColumns 重复列名的结果必须两列都取到各自的值。
//
// 驱动会给重复列 id/id1 这样的去重键，取值若按列名（name）而不是键（key）来查，
// 第二列会静默变 NULL 或复制第一列——两份数据看起来都"完整"，只有逐列比对才发现错位
func TestITQueryExportKeepsDuplicateColumns(t *testing.T) {
	for _, node := range appDialectNodes[:2] {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			exportPrepare(t, conn, 5)
			quote := conn.GetDialect().Quoter().QuoteIdent
			table := quote(exportTable)

			sql := fmt.Sprintf("SELECT a.%s, b.%s FROM %s a, %s b WHERE a.%s = 1 AND b.%s = 2", quote("name"), quote("name"), table, table, quote("id"), quote("id"))
			var buf bytes.Buffer
			require.NoError(t, exportQuery(context.Background(), exportApp(), conn, &dto.QueryExportReq{
				DbId: 1, Db: conn.Info.Database, SQL: sql, Format: "csv",
			}, writerTo(&buf)))

			content := buf.String()
			assert.Contains(t, content, "n1,n2", "两列必须分别取到各自的值，产物：%s", firstRunes(content, 200))
			assert.NotContains(t, content, "n1,n1", "第二列不得复制第一列")
		})
	}
}

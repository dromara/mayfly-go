//go:build it

package sqlexec

// SQL 变更预检的真实数据库集成测试。
//
// 单元测试钉的是「预检与执行同口径」，真库钉的是「库侧事实确实取到了」：
// 估算行数、是否全表扫描、目标表有无主键这三类结论只有连真库才能证伪，而它们正是策略判定
// 与预检面板显示给用户的内容。取不到时用户看到的是一句「拿不到估算」，两者必须能被区分。
//
// 另外钉住一件不可让渡的事：预检绝不执行被检语句（EXPLAIN 不带 ANALYZE）。
// 否则「提交前先检查一下」本身就变成了绕过审批改数据的通道。
//
// 覆盖 mysql / postgres（已声明 FeatExplain）与 sqlite（未声明，必须如实报「估算不可用」而不是放行）。
//
// 用例里的表名/列名全部是本文件常量并走方言 Quoter，不接受任何外部输入，因此拼字符串不引入注入面。
//
// 运行：cd server && go test -tags it -count=1 -run TestITSqlReview ./internal/db/application/

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/dbi/value"
)

const (
	reviewTable       = "it_sql_review_plan"
	reviewNoPkTable   = "it_sql_review_nopk"
	reviewNoPkColumn  = "note"
	reviewPrepareRows = 5
)

// withReviewConn 把派生字段的取连接入口指向真库连接：预检的库侧事实全部经这条缝取值
func withReviewConn(t *testing.T, conn *dbi.DbConn) {
	t.Helper()
	original := getDbConnForFacts
	getDbConnForFacts = func(context.Context, uint64, string) (*dbi.DbConn, error) {
		return conn, nil
	}
	t.Cleanup(func() { getDbConnForFacts = original })
}

// reviewPrepare 建两张表：有主键的样本表与无主键表，各写入若干行
func reviewPrepare(t *testing.T, conn *dbi.DbConn) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	execReview(t, conn, "DROP TABLE IF EXISTS "+quote(reviewTable))
	execReview(t, conn, "DROP TABLE IF EXISTS "+quote(reviewNoPkTable))
	execReview(t, conn, fmt.Sprintf("CREATE TABLE %s (id INT PRIMARY KEY, name VARCHAR(64))", quote(reviewTable)))
	execReview(t, conn, fmt.Sprintf("CREATE TABLE %s (id INT, name VARCHAR(64))", quote(reviewNoPkTable)))
	for i := 1; i <= reviewPrepareRows; i++ {
		execReview(t, conn, fmt.Sprintf("INSERT INTO %s (id, name) VALUES (%d, 'n%d')", quote(reviewTable), i, i))
		execReview(t, conn, fmt.Sprintf("INSERT INTO %s (id, name) VALUES (%d, 'n%d')", quote(reviewNoPkTable), i, i))
	}
}

// execReview 执行准备用 DDL/DML，失败即中止用例
func execReview(t *testing.T, conn *dbi.DbConn, sql string) {
	t.Helper()
	if _, err := conn.Exec(sql); err != nil {
		t.Fatalf("prepare failed [%s]: %s", sql, err.Error())
	}
}

// reviewRun 以预检口径评估单条语句。
//
// dbId 给个固定的非零值而不是 conn.Info.Id：集成测试的连接是手造的不带实体 Id，
// 而 0 在预检里是「没有目标库」，会被当成无法判定，测不出真实的库侧事实
func reviewRun(t *testing.T, conn *dbi.DbConn, sql string) []*ReviewFinding {
	t.Helper()
	withReviewConn(t, conn)
	findings, err := reviewStatement(context.Background(), conn.GetDialect(), 1, conn.Info.Database, sql)
	require.NoError(t, err, "预检本身不得失败")
	return findings
}

// hackedCount 被预检语句改写的行数：预检若真的执行了语句，这里会大于 0
func hackedCount(t *testing.T, conn *dbi.DbConn) int64 {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.QueryContext(context.Background(), "SELECT COUNT(*) AS cnt FROM "+quote(reviewTable)+" WHERE "+quote("name")+" = 'hacked'")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	cnt, ok := value.ValToInt64(rows[0]["cnt"])
	require.True(t, ok, "计数取值形态异常: %T", rows[0]["cnt"])
	return cnt
}

// reviewNodes 已实现执行计划分析的方言（与能力声明同源，不额外维护一份名单）
func reviewNodes(t *testing.T) []appDialectNode {
	t.Helper()
	nodes := make([]appDialectNode, 0, 2)
	for _, node := range appDialectNodes {
		conn := node.conn(t)
		declares := conn.Metadata().GetCapabilities().Has(dbi.FeatExplain)
		require.NoError(t, conn.Close())
		if declares {
			nodes = append(nodes, node)
		}
	}
	require.NotEmpty(t, nodes, "至少要有一个方言实现了执行计划分析")
	return nodes
}

// TestITSqlReviewUsesRealPlanAndDoesNotExecute 预检必须用真计划、且不执行被检语句。
//
// 无索引列上的 UPDATE 应被判为全表扫描（提醒级命中），同时「估算行数超阈值」不得命中：
// 5 行远低于默认 10 万阈值。后者若也命中，说明估算没真的取到、走了 fail-closed，
// 那用户在面板上看到的每一条 DML 都会是「需审批」，规则的参考价值归零
func TestITSqlReviewUsesRealPlanAndDoesNotExecute(t *testing.T) {
	for _, node := range reviewNodes(t) {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			reviewPrepare(t, conn)
			quote := conn.GetDialect().Quoter().QuoteIdent

			sql := fmt.Sprintf("UPDATE %s SET %s = 'hacked' WHERE %s LIKE 'n_'", quote(reviewTable), quote("name"), quote("name"))
			findings := reviewRun(t, conn, sql)

			fullScan := findingsOf(findings, dbCheckFullScan)
			require.Len(t, fullScan, 1, "无索引列上的过滤必须给出全表扫描结论, got %+v", findings)
			assert.Equal(t, dbi.EstimateKindExplain, fullScan[0].Detail["estimateKind"], "必须如实标注估算口径")
			rows, ok := fullScan[0].Detail["estimatedRows"].(float64)
			require.True(t, ok, "估算行数必须是数值而非缺失, got %#v", fullScan[0].Detail["estimatedRows"])
			assert.True(t, rows >= 0, "估算值必须真的从计划里取到, got %v", rows)
			// 没有「拿不到估算」的说明等于「计划真的读到了」：新建未分析的小表上 pg 会把行数估成 0，
			// 这里不断言量级的原因是那取决于统计信息，不由平台决定；要断言的是「取到了」而不是「取到了多少」
			assert.NotContains(t, fullScan[0].Detail, "estimateNote", "事实取到了就不该报「拿不到估算」")

			assert.Empty(t, findingsOf(findings, dbCheckRowsExceeds), "小表更新不该撞上默认 10 万行阈值")
			assert.Equal(t, int64(0), hackedCount(t, conn), "预检绝不执行被检语句")
		})
	}
}

// TestITSqlReviewDetectsTableWithoutPrimaryKey 无主键表上的删除必须被点名，且详情要说是哪张表：
// 只报「目标表没有主键」而不报表名，用户在多表语句里无从下手
func TestITSqlReviewDetectsTableWithoutPrimaryKey(t *testing.T) {
	for _, node := range reviewNodes(t) {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			reviewPrepare(t, conn)
			quote := conn.GetDialect().Quoter().QuoteIdent

			findings := reviewRun(t, conn, "DELETE FROM "+quote(reviewNoPkTable)+" WHERE "+quote("id")+" = 1")
			noPk := findingsOf(findings, dbCheckNoPrimaryKey)
			require.Len(t, noPk, 1, "无主键表上的删除必须命中, got %+v", findings)
			tables, ok := noPk[0].Detail["noPkTables"].([]string)
			require.True(t, ok, "详情必须给出表清单, got %#v", noPk[0].Detail["noPkTables"])
			assert.Contains(t, tables, reviewNoPkTable, "命中的表名必须出现在详情里, got %v", tables)

			// 有主键的同一张结构不该命中：证明结论来自内省而不是无条件报警
			assert.Empty(t, findingsOf(reviewRun(t, conn, "DELETE FROM "+quote(reviewTable)+" WHERE "+quote("id")+" = 1"), dbCheckNoPrimaryKey))
		})
	}
}

// TestITSqlReviewDdlUsesTableStatistics 结构变更走表统计而不是行数估算。
//
// ALTER 拿不到执行计划（mysql 的 EXPLAIN 不接受 DDL），若行规则不区分语句类型，
// 每条 DDL 都会被 fail-closed 报成「估算行数超阈值」。这里钉两件事：行规则闭嘴，
// 且小表不命中大表阈值（若表统计读不到会按 fail-closed 命中，说明内省真的works）
func TestITSqlReviewDdlUsesTableStatistics(t *testing.T) {
	for _, node := range reviewNodes(t) {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			reviewPrepare(t, conn)
			quote := conn.GetDialect().Quoter().QuoteIdent

			findings := reviewRun(t, conn, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s VARCHAR(200)", quote(reviewTable), quote(reviewNoPkColumn)))
			assert.Empty(t, findingsOf(findings, dbCheckRowsExceeds), "DDL 没有影响行数，不该拿到行规则结论")
			assert.Empty(t, findingsOf(findings, dbCheckFullScan), "DDL 没有执行计划，不该被报全表扫描")
			assert.Empty(t, findingsOf(findings, dbCheckBigTableDdl), "小表结构变更不该命中大表阈值（命中说明表统计没读到）")
		})
	}
}

// TestITSqlReviewWithoutPlanSaysUnavailable 未实现执行计划分析的方言必须明说「拿不到估算」。
//
// 面板一片绿是最危险的失效形态：sqlite 没有 EXPLAIN，行数规则按 fail-closed 命中，
// 但详情必须带上「本次拿不到估算」，否则用户会以为这条 UPDATE 影响很小
func TestITSqlReviewWithoutPlanSaysUnavailable(t *testing.T) {
	conn := appSQLiteConn(t)
	defer conn.Close()
	reviewPrepare(t, conn)
	quote := conn.GetDialect().Quoter().QuoteIdent

	findings := reviewRun(t, conn, "UPDATE "+quote(reviewTable)+" SET "+quote("name")+" = 'hacked' WHERE "+quote("id")+" > 0")
	rows := findingsOf(findings, dbCheckRowsExceeds)
	require.Len(t, rows, 1, "拿不到估算时行数规则必须 fail-closed, got %+v", findings)
	note, ok := rows[0].Detail["estimateNote"].(string)
	require.True(t, ok, "详情必须说明估算不可用, got %#v", rows[0].Detail)
	assert.Contains(t, note, "拿不到行数估算")
	assert.Equal(t, int64(0), hackedCount(t, conn), "预检绝不执行被检语句")
}

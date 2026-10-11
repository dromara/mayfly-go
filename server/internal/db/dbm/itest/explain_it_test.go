//go:build it

package itest

// 执行计划分析（可选能力 ExplainAnalyzer）的真实数据库集成测试。
//
// 钉住四件只有真库能证伪的事：
//  1. 无索引条件的 UPDATE/DELETE 会被判为全表扫描、用到索引的能报出索引名（预检与策略判定就靠这两个事实）；
//  2. 分析**绝不执行**被分析的语句：对一条 UPDATE 做分析后数据必须原样，
//     否则「预检」本身就成了绕过审批执行变更的通道（用 EXPLAIN ANALYZE 就会这样）；
//  3. EstimateKind 必须带出口径，上层不得把估算值当真实影响行数展示；
//  4. 未实现该能力的方言报 ErrExplainUnsupported 而不是零值结果——零值会被读成「估算 0 行」这个好消息。
//
// 已接 IT 节点的方言：mysql、postgres、mssql、oracle（关系型计划断言：无索引写语句→全表扫描、主键等值→非全表扫描）。
// clickhouse/dm 也已实现并声明 FeatExplain（护栏保证声明⟺接口实现），但其计划形态与关系型语义不同
// （clickhouse 无 UPDATE seek、dm 为文本计划无行数估算），不套用本文件的断言；本环境无可达实例，待真实实例上验证。
//
// 运行：cd server && go test -tags it -count=1 -run TestITExplain ./internal/db/dbm/itest/

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

const explainTable = "it_explain_plan"

// explainNode 各方言的连接与名称（本包的集成测试各自持有连接工厂，无常备节点表）
type explainNode struct {
	name string
	conn func(t *testing.T) *dbi.DbConn
}

func explainNodes() []explainNode {
	return []explainNode{
		{name: "mysql", conn: mysqlConn},
		{name: "pg", conn: pgConn},
		{name: "mssql", conn: gcMssqlConn},
		{name: "oracle", conn: oracleConn},
	}
}

// explainExec 执行测试用 DDL/DML，失败即中止用例
func explainExec(t *testing.T, conn *dbi.DbConn, sql string) {
	t.Helper()
	if _, err := conn.Exec(sql); err != nil {
		t.Fatalf("exec failed [%s]: %s", sql, err.Error())
	}
}

// explainPrepare 建表：id 为主键、name 无索引，用于分别走出索引与全表扫描的计划
//
// 列名必须与表名一样经 q() 引用：断言里的过滤条件用的是 q("id")/q("name")（引用后小写），
// 而建表若裸写 id/name，Oracle 会把未引用标识符折叠成大写 ID/NAME，于是 WHERE "id" 找不到列，
// 整条 EXPLAIN PLAN 报 ORA-00904（mysql/mssql 标识符大小写不敏感、pg 未引用折叠成小写，都掩盖了这个错配，
// 只有 Oracle 会暴露）。两侧统一引用才能让「建出来的列」与「断言引用的列」在四种方言上都是同一个
func explainPrepare(t *testing.T, conn *dbi.DbConn) {
	t.Helper()
	q := conn.GetDialect().Quoter().QuoteIdent
	explainExec(t, conn, "DROP TABLE IF EXISTS "+q(explainTable))
	explainExec(t, conn, fmt.Sprintf("CREATE TABLE %s (%s INT PRIMARY KEY, %s VARCHAR(64))", q(explainTable), q("id"), q("name")))
	for i := 1; i <= 5; i++ {
		explainExec(t, conn, fmt.Sprintf("INSERT INTO %s (%s, %s) VALUES (%d, 'n%d')", q(explainTable), q("id"), q("name"), i, i))
	}
}

func TestITExplainOnWriteStatementDoesNotExecute(t *testing.T) {
	for _, node := range explainNodes() {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			explainPrepare(t, conn)
			q := conn.GetDialect().Quoter().QuoteIdent

			stmt := fmt.Sprintf("UPDATE %s SET %s = 'hacked' WHERE %s LIKE 'n_'", q(explainTable), q("name"), q("name"))
			res, err := conn.Metadata().Explain(context.Background(), stmt)
			require.NoError(t, err, "分析必须成功（该方言已声明此能力）")

			assert.Equal(t, dbi.EstimateKindExplain, res.EstimateKind, "必须如实标注这是优化器估算")
			assert.True(t, res.FullScan, "对无索引列的过滤应被判为全表扫描，实际计划：%+v", res.Plan)

			// 分析不得执行语句：值仍是 n*，被改了说明分析器在用 ANALYZE 形态
			_, rows, err := conn.QueryContext(context.Background(), "SELECT "+q("name")+" FROM "+q(explainTable)+" WHERE "+q("id")+" = 1")
			require.NoError(t, err)
			require.Len(t, rows, 1)
			assert.Equal(t, "n1", fmt.Sprint(rows[0]["name"]), "EXPLAIN 分析绝不能真的执行 UPDATE")
		})
	}
}

func TestITExplainReportsUsedIndex(t *testing.T) {
	for _, node := range explainNodes() {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()
			explainPrepare(t, conn)
			q := conn.GetDialect().Quoter().QuoteIdent

			// 主键等值：pg 走 Index Scan、mysql 走 type=const/range 且 key=PRIMARY
			res, err := conn.Metadata().Explain(context.Background(), "SELECT * FROM "+q(explainTable)+" WHERE "+q("id")+" = 1")
			require.NoError(t, err)
			assert.False(t, res.FullScan, "主键等值不该被判为全表扫描，实际计划：%+v", res.Plan)
			assert.NotEmpty(t, res.Plan, "计划节点必须被解析出来，不能只给一个空壳")
		})
	}
}

// TestITExplainRejectsExplainItself 分析对象就是 EXPLAIN 时必须拒绝：
// 再套一层只会分析出「分析语句自身」的计划，给出与用户预期无关的结论
func TestITExplainRejectsExplainItself(t *testing.T) {
	for _, node := range explainNodes() {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()

			_, err := conn.Metadata().Explain(context.Background(), "EXPLAIN SELECT 1")
			require.Error(t, err)
			assert.False(t, errors.Is(err, dbi.ErrExplainUnsupported), "这是用法错误，不该被报成方言不支持: %v", err)
		})
	}
}

// TestITExplainPlanTreeShape 计划节点形态必须如实反映方言真实给出的结构。
//
// 两边差别是实质性的：pg 的 EXPLAIN (FORMAT JSON) 给了 Plans 嵌套，就必须还原出父子边
// （前端据此画树）；mysql 传统 EXPLAIN 是表形态，id 列不足以还原父子（UNION 的两个分支
// 会被连成假链），就必须保持平铺。凑一棵看着像样的树比不画更坑
func TestITExplainPlanTreeShape(t *testing.T) {
	const nestedSql = "SELECT * FROM %s WHERE %s IN (SELECT %s FROM %s WHERE %s = 'n1')"

	t.Run("pg", func(t *testing.T) {
		conn := pgConn(t)
		defer conn.Close()
		explainPrepare(t, conn)
		q := conn.GetDialect().Quoter().QuoteIdent

		res, err := conn.Metadata().Explain(context.Background(), fmt.Sprintf(nestedSql, q(explainTable), q("id"), q("id"), q(explainTable), q("name")))
		require.NoError(t, err)
		require.NotEmpty(t, res.Plan)
		assert.Positive(t, planEdgeCount(res), "JSON 计划带 Plans 嵌套，必须还原出父子关系，实际节点：%+v", res.Plan)
		assertPlanIndices(t, res)
	})

	t.Run("mysql", func(t *testing.T) {
		conn := mysqlConn(t)
		defer conn.Close()
		explainPrepare(t, conn)
		q := conn.GetDialect().Quoter().QuoteIdent

		res, err := conn.Metadata().Explain(context.Background(), fmt.Sprintf(nestedSql, q(explainTable), q("id"), q("id"), q(explainTable), q("name")))
		require.NoError(t, err)
		require.NotEmpty(t, res.Plan, "表形态的计划节点仍要逐行给出，只是不充当地位不明的父子关系")
		assert.Zero(t, planEdgeCount(res), "mysql 表形态不给出父子，必须留平铺而不是猜一棵树")
	})
}

func planEdgeCount(res *dbi.ExplainResult) int {
	edges := 0
	for _, node := range res.Plan {
		edges += len(node.Children)
	}
	return edges
}

// assertPlanIndices 父子下标必须自洽：前序展开意味着子一定晚于父，且不得越界。
// 前端按下标取节点画边，一个越界下标就是渲染期空指针
func assertPlanIndices(t *testing.T, res *dbi.ExplainResult) {
	t.Helper()
	for i, node := range res.Plan {
		for _, child := range node.Children {
			assert.Greater(t, child, i, "节点 %d 的子边指回了自己或祖先（成环）", i)
			assert.Less(t, child, len(res.Plan), "节点 %d 的子边下标越界", i)
		}
	}
}

// TestITExplainUnsupportedDialectIsNotZeroResult 未实现该能力的方言必须报错，
// 且不能返回一个可用的零值结果：零值会被上层读成「估算 0 行」，等于把「看不了」说成「很安全」
func TestITExplainUnsupportedDialectIsNotZeroResult(t *testing.T) {
	conn := sqliteConn(t)
	defer conn.Close()

	res, err := conn.Metadata().Explain(context.Background(), "SELECT 1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, dbi.ErrExplainUnsupported), "应报方言不支持: %v", err)
	assert.Nil(t, res, "不支持时必须返回 nil 结果，不能给零值")

	// 能力协商同样不得声明该能力：声明了却没实现会被 dbm 的护栏测试拦下
	caps := conn.Metadata().GetCapabilities()
	assert.False(t, caps.Has(dbi.FeatExplain), "sqlite 不应声明 FeatExplain")
	assert.NotContains(t, strings.Join(caps.SupportedFeatures(), ","), "explain")
}

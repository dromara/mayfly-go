package dbtool

import (
	"testing"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
	_ "mayfly-go/internal/db/dbm/dialect/mysql"

	"github.com/stretchr/testify/require"
)

// TestAgentExecOutcome 校验 Agent 复用交互执行链路后的结果折叠语义：
// 任一条语句失败必须整体判失败——绝不能把「前几条成功、后一条失败」的部分成功当成成功回报给模型
// （否则模型会说「已执行完成」，而数据只改了一半）。
func TestAgentExecOutcome(t *testing.T) {
	// 全成功：累加各写语句的 rowsAffected
	effected, err := agentExecOutcome([]*dto.DbSQLExecRes{
		{SQL: "UPDATE t SET a=1", Res: []map[string]any{{"rowsAffected": int64(3)}}},
		{SQL: "INSERT INTO t VALUES (1)", Res: []map[string]any{{"rowsAffected": int64(1)}}},
	})
	require.NoError(t, err)
	require.Equal(t, int64(4), effected)

	// 部分失败：整体判失败，且错误里带上失败的 SQL 与原因
	_, err = agentExecOutcome([]*dto.DbSQLExecRes{
		{SQL: "UPDATE t SET a=1", Res: []map[string]any{{"rowsAffected": int64(3)}}},
		{SQL: "DROP TABLE x", ErrorMsg: "permission denied"},
	})
	require.Error(t, err, "任一条失败必须整体报错，不能把部分成功当成功")
	require.Contains(t, err.Error(), "permission denied")
	require.Contains(t, err.Error(), "DROP TABLE x")

	// 空/nil：0 影响行、无错误
	effected, err = agentExecOutcome(nil)
	require.NoError(t, err)
	require.Equal(t, int64(0), effected)
}

// TestDecideAtomicBatch 守住 Agent 写路径的原子批次判定：全部语句都是数据写才整批走事务
// （全成功才提交、任一条失败整批回滚）；混入 DDL 必须退化为逐条执行——MySQL 的 DDL 隐式提交，
// 把它包进「原子批次」等于给模型一个无法兑现的「整批可回滚」承诺，回滚根本不会发生
func TestDecideAtomicBatch(t *testing.T) {
	dialect := dbi.GetDialect(dbi.ToDbType("mysql"))
	require.NotNil(t, dialect, "mysql 方言未注册，测试无法进行")
	parser, splitter := dialect.GetSQLParser(), dialect.GetSQLSplitter()

	// 全数据写：整批可原子
	require.True(t, decideAtomicBatch(parser, splitter, []string{
		"UPDATE t SET a=1 WHERE id=1",
		"INSERT INTO t VALUES (2)",
		"DELETE FROM t WHERE id=3",
	}), "全数据写批次应可原子")

	// 混入 DDL：退化为逐条执行
	require.False(t, decideAtomicBatch(parser, splitter, []string{
		"CREATE TABLE t (id INT)",
		"INSERT INTO t VALUES (1)",
	}), "混 DDL 批次不得声明原子，否则给出无法兑现的可回滚承诺")

	// 混入只读语句：SELECT 结果不参与回滚语义，同样退化
	require.False(t, decideAtomicBatch(parser, splitter, []string{
		"SELECT * FROM t",
		"DELETE FROM t",
	}), "混 SELECT 批次不得声明原子")

	// TRUNCATE 归 DDL，不得进原子批次
	require.False(t, decideAtomicBatch(parser, splitter, []string{
		"TRUNCATE TABLE t",
	}), "TRUNCATE 不得进原子批次")

	// 空批次兜底 false
	require.False(t, decideAtomicBatch(parser, splitter, nil))
}

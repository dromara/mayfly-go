package itest

// 导入链路「事务边界」真实库集成测试。
//
// 背景：dump产物中分段注释头会与其后的语句切成同一条（实测mysql同构备份产物即为
// "-- ----------------------------\n-- Data: t \n-- ----------------------------\nBEGIN"），
// 若按语句文本前缀判定事务控制语句则漏过滤，该语句会被真实执行：
//   - mysql：BEGIN隐式提交当前事务 → 在途数据提前落库，失败回滚承诺静默失效
//   - pg/sqlite：COMMIT提前结束事务 → 后续语句脱离事务，回滚只能覆盖剩余部分
// 均属"失败后残留部分数据"的数据安全级后果，必须在真实库上验证事务边界确实由
// ImportDumpStream 独占管理。
//
// 与单元测试（db_transfer_import_test.go）的分工：单测验证「切割+过滤」决策本身，
// 本文件验证真实数据库服务器对这些语句的实际反应（隐式提交行为），并证明本用例
// 设计的断言在该行为发生时确实会失败（control 用例，防止断言空转）。
//
// 运行：cd server && go test -tags it -count=1 -run TestITImportDump ./internal/db/application/transfer/

import (
	"mayfly-go/internal/db/application/transfer"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
)

// itCommentHeader 与生产dump产物完全一致的分段注释头（实测见 DumpDbScript 输出）
const itCommentHeader = "-- ----------------------------\n-- Data: %s \n-- ----------------------------\n"

// itAllNodes已提升为公共的 itAllNodes（见 it_env_it_test.go）：它本意就是“全部本地可用方言”，
// 困在本用例文件内会让其他链路用例反向依赖本文件。

// itTxnDDL 各方言建表语句（id/val 两列，导入前置建表，用例只关注数据语句的事务归属）
var itTxnDDL = map[string]string{
	"mysql":    "(id int PRIMARY KEY, val varchar(32))",
	"postgres": "(id int PRIMARY KEY, val varchar(32))",
	"sqlite":   "(id INTEGER PRIMARY KEY, val TEXT)",
	"mssql":    "(id int PRIMARY KEY, val nvarchar(32))",
}

// itPrepareLeakTable 建一张已提交的空表供导入写入，返回表名与查询/写入用的引用名
func itPrepareLeakTable(t *testing.T, conn *dbi.DbConn, table string) string {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table)))
	require.NoError(t, err)
	ddl, ok := itTxnDDL[string(conn.Info.Type)]
	require.True(t, ok, "未配置该方言的建表语句: %s", conn.Info.Type)
	_, err = conn.Exec(fmt.Sprintf("CREATE TABLE %s %s", quote(table), ddl))
	require.NoError(t, err)
	return quote(table)
}

// itCountLeak 读回表内行数（表不存在时返回错误，由调用方按用例语义处理）
func itCountLeak(t *testing.T, conn *dbi.DbConn, table string) int64 {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.Query(fmt.Sprintf("SELECT COUNT(*) AS cnt FROM %s", quote(table)))
	require.NoError(t, err)
	cnt, ok := dbi.ValToInt64(rows[0]["cnt"])
	require.True(t, ok, "count应为数值: %#v", rows[0]["cnt"])
	return cnt
}

// itMissingTableStmt 引用一个必定不存在的表，用于在导入中途制造失败
const itMissingTableStmt = "INSERT INTO it_leak_no_such_table VALUES (1);"

// TestITImportDumpCommentWrappedBeginNoImplicitCommit 注释前缀的BEGIN不得被真正执行：
// 失败后在途写入必须整体回滚（mysql上该语句会隐式提交当前事务）
func TestITImportDumpCommentWrappedBeginNoImplicitCommit(t *testing.T) {
	for _, node := range itAllNodes {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()

			table := "it_leak_begin_" + node.name
			quoteTable := itPrepareLeakTable(t, conn, table)

			// 第1条INSERT在途未提交；紧随其后的注释头+BEGIN若被过滤则不计入事务，
			// 若被执行则mysql隐式提交第1条INSERT
			script := fmt.Sprintf("INSERT INTO %s VALUES (1, 'a');\n%sBEGIN;\nINSERT INTO %s VALUES (2, 'b');\n%s\n",
				quoteTable, fmt.Sprintf(itCommentHeader, table), quoteTable, itMissingTableStmt)

			app := &transfer.DbTransferAppImpl{}
			err := app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader(script))
			require.Error(t, err, "引用不存在表应导入失败")

			assert.Equal(t, int64(0), itCountLeak(t, conn, table),
				"失败后事务内已执行的INSERT必须整体回滚（残留行数>0 说明BEGIN被真正执行并隐式提交了事务）")
		})
	}
}

// TestITImportDumpCommentWrappedCommitNoPrematureCommit 注释前缀的COMMIT不得被真正执行：
// 提前结束事务会让后续语句脱离事务自动提交（pg/sqlite上该语句会真实结束事务）
func TestITImportDumpCommentWrappedCommitNoPrematureCommit(t *testing.T) {
	for _, node := range itAllNodes {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()

			table := "it_leak_commit_" + node.name
			quoteTable := itPrepareLeakTable(t, conn, table)

			script := fmt.Sprintf("INSERT INTO %s VALUES (1, 'a');\n%sCOMMIT;\nINSERT INTO %s VALUES (2, 'b');\n%s\n",
				quoteTable, fmt.Sprintf(itCommentHeader, table), quoteTable, itMissingTableStmt)

			app := &transfer.DbTransferAppImpl{}
			require.Error(t, app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader(script)))

			assert.Equal(t, int64(0), itCountLeak(t, conn, table),
				"失败后事务内已执行的INSERT必须整体回滚（残留行数>0 说明COMMIT被真正执行并提前结束事务）")
		})
	}
}

// TestITImportDumpStream_SqliteTarget 补齐三方言目标一致性：sqlite 目标同样要能
// 过滤dump事务包装并完成全量导入（历史仅覆盖 mysql/pg）
func TestITImportDumpStream_SqliteTarget(t *testing.T) {
	conn := itSqliteNode(t)
	defer conn.Close()

	quote := conn.GetDialect().Quoter().QuoteIdent
	table := "it_import_sqlite_target"
	_, err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table)))
	require.NoError(t, err)

	app := &transfer.DbTransferAppImpl{}
	script := buildDumpScript(quote, table, 600) // 600条 > 500：至少触发一次批级提交
	require.NoError(t, app.ImportDumpStream(context.Background(), 0, conn, strings.NewReader(script)))

	assert.Equal(t, 600, int(itCountLeak(t, conn, table)), "BEGIN/COMMIT/autocommit应被过滤，600行数据全部落库")
}

// TestITRealDumpProductTxnWrapperShape 钉住生产dump产物的语句形态：mysql同构备份中，
// 分段注释头必须与BEGIN切成同一条语句（这正是按文本前缀判定会漏过滤的形态）。
// 若dump产物结构变化（不再带注释头包装），本用例提醒上方事务边界用例需同步重审，
// 而不是静默失去鉴别力
func TestITRealDumpProductTxnWrapperShape(t *testing.T) {
	conn := itMysqlNode(t)
	defer conn.Close()

	table := "it_shape_mysql_txn"
	quoteTable := itPrepareLeakTable(t, conn, table)
	_, err := conn.Exec(fmt.Sprintf("INSERT INTO %s VALUES (1, 'a'), (2, 'b')", quoteTable))
	require.NoError(t, err)

	script := itDumpTable(t, conn, table, "mysql", false, true, "")
	require.Contains(t, script, "BEGIN;", "mysql同构备份应带事务包装")

	var stmts []string
	require.NoError(t, conn.GetDialect().GetSQLSplitter().SplitSQL(strings.NewReader(script), func(s string) error {
		stmts = append(stmts, s)
		return nil
	}))

	var wrapped bool
	for _, s := range stmts {
		if strings.HasPrefix(s, "--") && sqlparser.IsTxnControlStmt(conn.GetDialect().GetSQLSplitter(), s) {
			wrapped = true
			// 文本前缀判定会漏掉它：TrimSpace后以注释开头而非BEGIN开头
			assert.False(t, strings.EqualFold(strings.TrimSpace(s), "BEGIN"), "形态已变化：注释不再与BEGIN同属一条")
		}
	}
	assert.True(t, wrapped, "dump产物中未找到「注释头+事务语句」同属一条的形态，上方事务边界用例的前提已失效")
}

// TestITRealDumpProductNoTxnWrapperPgSqlite pg/sqlite作为**目标方言**时，备份产物不得含事务控制语句。
//
// 这两个方言的DumpHelper显式不输出BEGIN/COMMIT（事务由导入侧自管），无论源库是何方言：
// 产物若带回这些语句，在sqlite上会报cannot start a transaction within a transaction，
// 在pg上会提前结束事务而丢部分提交。判定用目标方言的切割器（与导入侧一致）
func TestITRealDumpProductNoTxnWrapperPgSqlite(t *testing.T) {
	for _, src := range itAllNodes {
		src := src
		for _, target := range []dbi.DbType{itPg.dbType, itSqlite.dbType} {
			target := target
			t.Run(src.name+"->"+string(target), func(t *testing.T) {
				conn := src.conn(t)
				defer conn.Close()

				table := "it_shape_notxn_" + src.name
				quoteTable := itPrepareLeakTable(t, conn, table)
				_, err := conn.Exec(fmt.Sprintf("INSERT INTO %s VALUES (1, 'a'), (2, 'b')", quoteTable))
				require.NoError(t, err)

				script := itDumpTable(t, conn, table, target, true, true, "")
				splitter := dbi.GetDialect(target).GetSQLSplitter()
				require.NoError(t, splitter.SplitSQL(strings.NewReader(script), func(stmt string) error {
					require.False(t, sqlparser.IsTxnControlStmt(splitter, stmt),
						"目标方言[%s]产物不应含事务控制语句: %s", target, strings.TrimSpace(stmt))
					return nil
				}))
				require.Contains(t, script, "INSERT INTO", "产物未含INSERT语句，本用例断言无效")
			})
		}
	}
}

// itLeakExpect 不过滤执行「注释前缀+事务语句」后，本机数据库实际表现出的事务边界破坏形态
// （本机mysql 8.0.46 / postgres 16 / sqlite / SQL Server 2022实测值；survived=回滚后仍残留的行数）：
//   - survived>0：事务被提前提交 → 失败后残留部分数据（数据安全级后果）
//   - stmtErrContains非空：语句直接报错 → 导入中断
//   - 两者皆无：该库在事务内容忍此语句（如pg的BEGIN仅告警），不构成风险
//
// 本表是上方生产用例的**非空转证明**：若某方言两个语句组合均无破坏，则上方rows==0断言
// 无鉴别力，必须重新设计用例而非宣告通过。数据库/驱动行为变化会让本用例变红，提醒重审风险面。
var itLeakExpect = map[string]map[string]struct {
	survived        int64
	stmtErrContains string
	unharmful       bool
}{
	"mysql": {
		"BEGIN":  {survived: 1}, // 隐式提交当前事务，在途的第1行落库
		"COMMIT": {survived: 2}, // 提前结束事务，第1行已提交且第2行转为自动提交
	},
	"postgres": {
		"BEGIN":  {unharmful: true}, // 已有事务时仅告警，事务边界未被破坏
		"COMMIT": {survived: 2},
	},
	"sqlite": {
		"BEGIN":  {stmtErrContains: "cannot start a transaction within a transaction"},
		"COMMIT": {survived: 2},
	},
	// mssql的BEGIN/COMMIT语义与其余三方言均不同（本机SQL Server 2022实测）：
	//   - 裸`BEGIN`不是事务语句而是块起始关键字，直接报语法错误102 → 导入中断（同sqlite属破坏性）；
	//     这也正是mssql的DumpHelper显式不输出事务包装的原因（见mssql/helper.go）
	//   - `COMMIT`会真实结束会话事务，驱动随后报“server does not have an active transaction”，
	//     故仅第1行提前落库（survived=1，而非pg/mysql的2）
	"mssql": {
		"BEGIN":  {stmtErrContains: "Incorrect syntax near"},
		"COMMIT": {survived: 1},
	},
}

// TestITImportDumpLeakControlUnfilteredMustBeDetectable 用例有效性对照：
// 按「不过滤」方式在事务内直接执行上述注释前缀事务语句，逐库核对真实破坏形态。
func TestITImportDumpLeakControlUnfilteredMustBeDetectable(t *testing.T) {
	for _, node := range itAllNodes {
		node := node
		t.Run(node.name, func(t *testing.T) {
			conn := node.conn(t)
			defer conn.Close()

			expectByStmt, ok := itLeakExpect[string(node.dbType)]
			require.True(t, ok, "缺少方言[%s]的对照预期", node.dbType)
			harmfulCount := 0

			for _, txnStmt := range []string{"BEGIN", "COMMIT"} {
				table := fmt.Sprintf("it_leak_ctl_%s_%s", node.name, strings.ToLower(txnStmt))
				quoteTable := itPrepareLeakTable(t, conn, table)
				header := fmt.Sprintf(itCommentHeader, table)

				tx, err := conn.Begin()
				require.NoError(t, err)
				_, err = conn.TxExecContext(context.Background(), tx,
					fmt.Sprintf("INSERT INTO %s VALUES (1, 'a');", quoteTable))
				require.NoError(t, err)

				// 模拟未过滤的旧实现：注释前缀的事务语句被当作普通业务语句真实执行
				_, stmtErr := conn.TxExecContext(context.Background(), tx, header+txnStmt+";")
				_, nextErr := conn.TxExecContext(context.Background(), tx,
					fmt.Sprintf("INSERT INTO %s VALUES (2, 'b');", quoteTable))
				_ = tx.Rollback()

				survived := itCountLeak(t, conn, table)
				t.Logf("[%s] 不过滤执行 %s：stmtErr=%v nextErr=%v 回滚后残留行数=%d", node.name, txnStmt, stmtErr, nextErr, survived)

				expect := expectByStmt[txnStmt]
				switch {
				case expect.stmtErrContains != "":
					require.Error(t, stmtErr, "[%s] 事务内执行%s应报错", node.name, txnStmt)
					assert.Contains(t, stmtErr.Error(), expect.stmtErrContains)
					harmfulCount++
				case expect.unharmful:
					require.NoError(t, stmtErr)
					require.EqualValues(t, 0, survived,
						"[%s] 事务内执行%s已被视为无害容忍，若开始破坏事务边界需重审生产风险结论", node.name, txnStmt)
				default:
					require.NoError(t, stmtErr)
					require.EqualValues(t, expect.survived, survived,
						"[%s] 不过滤执行%s的残留行数与实测预期不符", node.name, txnStmt)
					harmfulCount++
				}
			}

			require.Positive(t, harmfulCount,
				"[%s] 两种事务包装语句均不破坏事务边界，上方生产用例断言属空转，需重新设计", node.name)
		})
	}
}

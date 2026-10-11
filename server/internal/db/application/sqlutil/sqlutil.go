package sqlutil

// SQL 语句「切割 / 分类 / 错误摘要」的共享工具。
//
// 存在的理由：取执行计划（核心库能力 Db.Explain）与触发策略预检（sqlexec 域的 review/rollback）
// 必须共用同一套切割 + 分类 + 错误摘要实现——面板上看到的「第 N 条」与真正被送进数据库的「第 N 条」
// 必须是同一条，错误摘要的口径也不能漂移。这些函数只依赖 dbi/sqlparser 等下层，放在这个中立子包里，
// 让父包（Explain/CancelSession）与 sqlexec 子包共用一份，而不是任何一方私有、也不必互相依赖。

import (
	"context"
	"strings"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/stringx"
)

// SplitConnStatements 按目标库方言切割语句，与执行期用的是同一个 splitter。
//
// 预检与取计划都必须先过这一步：面板上看到的「第 2 条」与真正被送进数据库的「第 2 条」
// 必须是同一条，否则用户核对不了结论；而把未切割的一串语句交给 EXPLAIN，
// 「EXPLAIN select 1; DROP TABLE t」在简单查询协议下会连第二条一起执行
func SplitConnStatements(ctx context.Context, conn *dbi.DbConn, sqls []string) ([]string, error) {
	statements := make([]string, 0, len(sqls))
	splitter := conn.GetDialect().GetSQLSplitter()
	for _, blob := range sqls {
		if err := splitter.SplitSQL(strings.NewReader(blob), func(one string) error {
			if stringx.TrimSpaceAndBr(one) != "" {
				statements = append(statements, one)
			}
			return nil
		}); err != nil {
			return nil, imsg.SplitError(ctx, err)
		}
	}
	return statements, nil
}

// ClassifyStatement 与执行期同源的语句分类：解析失败不报错，按关键字兜底分类
// （预检/取计划都要能回答「这条是什么语句」，而不能因为解析器不认就拒绝服务）
func ClassifyStatement(dialect dbi.Dialect, sql string) (sqlstmt.Stmt, string) {
	parser, splitter := dialect.GetSQLParser(), dialect.GetSQLSplitter()
	stmt, parseErr := parser.Parse(sql)
	if parseErr != nil {
		logx.Warnf("sql review: failed to parse the statement, classified by keyword: %s", parseErr.Error())
	}
	return stmt, string(sqlparser.Classify(parser, splitter, sql, stmt, parseErr))
}

// BriefReason 把数据库返回的原始错误收敛成一行短文本（预检说明与会话取消提示共用）。
func BriefReason(err error) string {
	if err == nil {
		return ""
	}
	reason := strings.TrimSpace(strings.SplitN(strings.TrimSpace(err.Error()), "\n", 2)[0])
	const maxRunes = 160
	if runes := []rune(reason); len(runes) > maxRunes {
		return string(runes[:maxRunes]) + "…"
	}
	return reason
}

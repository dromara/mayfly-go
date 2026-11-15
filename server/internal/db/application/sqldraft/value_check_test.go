package sqldraft

// 取值核对的回归锁。
//
// 真机缺陷：MySQL/PG 生成的语句默认给列名加引用符（`type` = 'oceanbase'），而字面量正则
// 在列名之后直接要求 `\s*=`，遇到收尾的引用符就永不匹配。表现是 valueChecker 对每一条真实
// 草稿空转、界面把「取值根本不存在」的语句报成「已通过校验」—— 正是这套校验要拦的那类错。

import (
	"context"
	"strings"
	"testing"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

func TestLiteralsForColumnAcceptsQuotedIdentifiers(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		want string
	}{
		{"裸列名", "SELECT 1 FROM t WHERE type = 'mysql'", "mysql"},
		{"反引号（mysql）", "SELECT 1 FROM t WHERE `type` = 'oceanbase'", "oceanbase"},
		{"双引号（pg/oracle）", `SELECT 1 FROM t WHERE "type" = 'oceanbase'`, "oceanbase"},
		{"方括号（mssql）", "SELECT 1 FROM t WHERE [type] = 'oceanbase'", "oceanbase"},
		{"表限定 + 反引号", "SELECT 1 FROM t WHERE `i`.`type` = 'oceanbase'", "oceanbase"},
		{"大小写混排的引用符", "SELECT 1 FROM t WHERE `TYPE` = 'oceanbase'", "oceanbase"},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			got := literalsForColumn(item.sql, "type")
			// 大小写那条靠的是 (?i) 忽略大小写匹配列名，取值原样保留
			if len(got) != 1 || got[0] != item.want {
				t.Fatalf("expect [%s], got %v", item.want, got)
			}
		})
	}

	// 前缀相同的列不能被误命中：status_desc 的取值不是 status 的取值域
	if got := literalsForColumn("SELECT 1 FROM t WHERE `status_desc` = '已支付'", "status"); len(got) != 0 {
		t.Fatalf("a longer column must not match, got %v", got)
	}
	// IN 形态同样要认引用符
	got := literalsForColumn("SELECT 1 FROM t WHERE `type` IN ('mysql', 'oceanbase')", "type")
	if len(got) != 2 {
		t.Fatalf("expect two IN literals, got %v", got)
	}
}

func TestValueCheckerFlagsUnsampledLiteralInQuotedStatement(t *testing.T) {
	ctx := context.Background()
	sql := "SELECT `name` FROM `t_db_instance` WHERE `type` = 'oceanbase' AND `is_deleted` = 0"
	cc := &checkCtx{
		sql:     sql,
		sampled: []sampledValue{{Table: "t_db_instance", Column: "type", Values: []string{"mysql", "postgres", "sqlite"}}},
	}
	findings := valueChecker{}.Check(ctx, cc)
	if len(findings) != 1 {
		t.Fatalf("expect one unverified finding for an unsampled value, got %+v", findings)
	}
	if findings[0].Severity != SeverityUnverified || findings[0].Code != CodeValueNotSampled {
		t.Fatalf("wrong classification: %+v", findings[0])
	}
	if !strings.Contains(findings[0].Message, "oceanbase") {
		t.Fatalf("the message must name the offending value, got %q", findings[0].Message)
	}
	if len(findings[0].Candidates) != 3 {
		t.Fatalf("the sampled values must be offered as candidates, got %v", findings[0].Candidates)
	}

	// 取值在采样集内时不该报任何东西（否则每份正常草稿都会被噪声淹没）
	clean := "SELECT `name` FROM `t_db_instance` WHERE `type` = 'mysql' AND `is_deleted` = 0"
	cc.sql = clean
	again := valueChecker{}.Check(ctx, cc)
	if len(again) != 0 {
		t.Fatalf("a sampled value must not be flagged, got %+v", again)
	}
}

// 放弃式产物（把「我不知道」写成一个查询结果列）不能被当成有效草稿。
//
// 真机证据：上下文缺表时模型输出 `SELECT 'ERROR: No metric sampling table found in
// context' AS error_message, NULL AS ...`，语法合法、没有不存在的列，被一路判成
// 「已通过校验」。空表引用 + 文本里没有 from 是这类产物的稳定特征。
func TestNoSourceCheckerCatchesGiveUpStatement(t *testing.T) {
	ctx := context.Background()
	check := func(sql string) []Finding {
		stmt, stmtTyp := parseForCheck(t, sql)
		return noSourceChecker{}.Check(ctx, &checkCtx{sql: sql, stmt: stmt, stmtTyp: stmtTyp})
	}

	giveUp := "SELECT 'ERROR: No metric sampling table found in context' AS error_message, NULL AS instance_id"
	findings := check(giveUp)
	if len(findings) != 1 || findings[0].Code != CodeNoTableSource || findings[0].Severity != SeverityUnverified {
		t.Fatalf("a table-less select must be reported as unverifiable, got %+v", findings)
	}

	// 正常查询不该被牵连
	if got := check("SELECT `name` FROM `t_db_instance` WHERE `type` = 'mysql'"); len(got) != 0 {
		t.Fatalf("a statement with a table must pass, got %+v", got)
	}

	// CTE 的表引用抽不出来是解析器的已知边界（由 identifier 标 uncheckable），这里不重复下结论
	if got := check("WITH x AS (SELECT 1 AS a) SELECT a FROM x"); len(got) != 0 {
		t.Fatalf("a statement mentioning FROM must be left to the identifier checker, got %+v", got)
	}
}

// parseForCheck 用真实 mysql 方言解析（判据依赖 AST 的表引用集合，不能用替身）
func parseForCheck(t *testing.T, sql string) (sqlstmt.Stmt, string) {
	t.Helper()
	dialect := dbi.GetDialect(dbi.ToDbType("mysql"))
	if dialect == nil {
		t.Fatal("mysql dialect not registered")
	}
	return classify(dialect, sql)
}

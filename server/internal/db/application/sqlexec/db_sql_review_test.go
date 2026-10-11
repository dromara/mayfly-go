package sqlexec

// SQL 变更预检的单元测试。
//
// 这里钉住的是「预检与执行必须同口径」这条契约：预检面板给出的结论一旦和执行期不一致，
// 用户就会在「面板说没问题、执行时被拦」与「面板报警、执行却放行」之间失去判断依据，
// 而后者更危险——它会让人以为自己在提交前已经检查过了。
//
// 真实库才能验证的部分（估算行数、全表扫描、主键缺失、分析不得执行语句）在
// db_sql_review_it_test.go 与 dbm/itest/explain_it_test.go，本文件只覆盖无需连库的语义。

import (
	"context"
	"errors"
	"strings"
	"testing"

	"mayfly-go/internal/db/dbm/dbi"
	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/trigger"
)

// stubNoDbConn 让派生字段取不到库连接（等价于「方言不支持执行计划 / 库不可达」），测试结束恢复
func stubNoDbConn(t *testing.T) {
	t.Helper()
	original := getDbConnForFacts
	getDbConnForFacts = func(context.Context, uint64, string) (*dbi.DbConn, error) {
		return nil, errors.New("db connection is unavailable in unit test")
	}
	t.Cleanup(func() { getDbConnForFacts = original })
}

func reviewDialect(t *testing.T) dbi.Dialect {
	t.Helper()
	dialect := dbi.GetDialect(dbi.ToDbType("mysql"))
	if dialect == nil {
		t.Fatal("mysql dialect is not registered, the test cannot run")
	}
	return dialect
}

// findingsOf 按检查项取结论，nil 表示该检查项没有结论（既未命中也没报「无法判定」）
func findingsOf(findings []*ReviewFinding, checkKey string) []*ReviewFinding {
	got := make([]*ReviewFinding, 0, 1)
	for _, finding := range findings {
		if finding.CheckKey == checkKey {
			got = append(got, finding)
		}
	}
	return got
}

// TestReviewAndExecutionShareVerdict 同一语句在预检与执行期必须得出同样的结论。
//
// 两边共用的是检查项定义与结构化事实，因此这里比对的是「按纯解析事实判定的检查项」：
// 无 WHERE 的删除在预检里必须出现为命中，并在执行期确实把语句拦下；
// 正常的 INSERT 两边都不得拦。任何一侧单独改规则都会让这个用例变红
func TestReviewAndExecutionShareVerdict(t *testing.T) {
	stubNoDbConn(t)
	dialect := reviewDialect(t)
	ctx := context.Background()

	executable := &DbSQLExecAppImpl{flowProcdefApp: fakeProcdefReader{governed: &flowentity.Procdef{
		TriggerPolicy: &flowentity.TriggerPolicy{
			Version:         flowentity.TriggerPolicyVersion,
			DefaultSeverity: flowentity.SeverityDisabled,
			Checks: []*flowentity.CheckConfig{{
				Key: dbCheckDmlWithoutWhere, BizType: DbSQLExecFlowBizType, Severity: flowentity.SeverityRequired,
			}},
		},
	}}}

	cases := []struct {
		name       string
		sql        string
		wantReview bool
		wantBlock  bool
	}{
		{"无 WHERE 的删除两边都拦", "DELETE FROM orders", true, true},
		{"无 WHERE 的更新两边都拦", "UPDATE orders SET status = 2", true, true},
		{"带列清单的插入两边都放行", "INSERT INTO orders (id) VALUES (1)", false, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			findings, err := reviewStatement(ctx, dialect, 1, "mayfly", c.sql)
			if err != nil {
				t.Fatalf("review failed: %v", err)
			}
			hits := findingsOf(findings, dbCheckDmlWithoutWhere)
			if got := len(hits) > 0; got != c.wantReview {
				t.Fatalf("the review panel verdict = %v, want %v (findings=%+v)", got, c.wantReview, findings)
			}

			_, execErr := executable.checkSqlsWithoutTicket(ctx, nil, []string{"default/2|x/"}, dialect, []string{c.sql})
			if got := execErr != nil; got != c.wantBlock {
				t.Fatalf("the execution verdict = %v (err=%v), want %v; the preview and execution must not disagree", got, execErr, c.wantBlock)
			}
		})
	}
}

// TestRowsFindingSaysEstimateUnavailable 「算不出估算」必须在详情里说出来，不能只是缺个字段。
//
// 详情里少一个行数和「估算为 0 行」在界面上长得一样，而后者是假安心：
// 用户会以为这条 UPDATE 影响极小，实际上服务端根本没看过执行计划
func TestRowsFindingSaysEstimateUnavailable(t *testing.T) {
	stubNoDbConn(t)
	dialect := reviewDialect(t)

	findings, err := reviewStatement(context.Background(), dialect, 1, "mayfly", "UPDATE orders SET status = 2 WHERE id > 0")
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}

	rows := findingsOf(findings, dbCheckRowsExceeds)
	// fail-closed：算不出来按命中处理，「大改动必须有人看过」不该因为方言不支持而静默放行
	if len(rows) != 1 {
		t.Fatalf("the affected-rows check must fail closed when the estimate is unavailable, got %+v", findings)
	}
	note, ok := rows[0].Detail["estimateNote"]
	if !ok || !strings.Contains(castString(note), "拿不到行数估算") {
		t.Fatalf("the finding must say the estimate is unavailable instead of omitting it, got %+v", rows[0].Detail)
	}

	// 性能提示一侧相反：拿不到计划就不打扰用户，也不该冒出一个「无法判定」的空壳结论
	if got := findingsOf(findings, dbCheckFullScan); len(got) != 0 {
		t.Fatalf("the full-scan check must stay silent when there is no plan, got %+v", got)
	}
}

// TestReviewUsesCheckDefaultParams 预检声称「按各检查项的默认参数评估」，这句文案必须是真的：
// 默认参数只能来自检查项定义本身，与策略页未填参数时的取值同源
func TestReviewUsesCheckDefaultParams(t *testing.T) {
	def, ok := trigger.CheckOf(dbCheckRowsExceeds)
	if !ok {
		t.Fatal("the affected-rows check is not registered")
	}
	params := defaultCheckParams(def)
	if got := trigger.ParamFloat(params, "maxRows", -1); got != 100000 {
		t.Fatalf("the affected-rows default must be the declared one, got %v", got)
	}

	ddlDef, ok := trigger.CheckOf(dbCheckBigTableDdl)
	if !ok {
		t.Fatal("the large-table DDL check is not registered")
	}
	if got := trigger.ParamFloat(defaultCheckParams(ddlDef), "minRows", -1); got != 1000000 {
		t.Fatalf("the large-table DDL default must be the declared one, got %v", got)
	}
}

// TestReviewSeverityComesFromCheckDefault 「建议级别」列必须来自检查项自带的默认级别：
// 预检不裁决处置，若它按自己的偏好标级别，用户会以为面板结论等于策略结论
func TestReviewSeverityComesFromCheckDefault(t *testing.T) {
	stubNoDbConn(t)
	dialect := reviewDialect(t)

	findings, err := reviewStatement(context.Background(), dialect, 1, "mayfly", "DELETE FROM orders")
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}

	hits := findingsOf(findings, dbCheckDmlWithoutWhere)
	if len(hits) != 1 {
		t.Fatalf("expected one hit of the no-where check, got %+v", findings)
	}
	def, ok := trigger.CheckOf(dbCheckDmlWithoutWhere)
	if !ok {
		t.Fatal("the no-where check is not registered")
	}
	if hits[0].Severity != def.Default {
		t.Fatalf("severity = %v, want the check default %v", hits[0].Severity, def.Default)
	}
	if hits[0].Reason == "" || hits[0].TitleKey != def.TitleKey {
		t.Fatalf("the finding must carry the rule title and its reason, got %+v", hits[0])
	}
	if hits[0].SQL != "DELETE FROM orders" {
		t.Fatalf("the finding must echo the statement it judged, got %q", hits[0].SQL)
	}
}

// TestReviewKeepsUnverifiableCheckAsNote 「这一项本次没结论」必须单独成一行，不能当成未命中。
//
// 预检面板一旦把求值失败折叠成「没命中任何检查项」，界面就显示一片绿，
// 而真实情况是规则根本没跑完——这是整条链路上最危险的谎报
func TestReviewKeepsUnverifiableCheckAsNote(t *testing.T) {
	dialect := reviewDialect(t)
	unverifiable := trigger.CheckDef{
		Key:      "db.unverifiable-for-test",
		TitleKey: "flow.check.unverifiable",
		BizTypes: []string{DbSQLExecFlowBizType},
		Default:  flowentity.SeverityRequired,
		Evaluate: func(context.Context, *trigger.Context, map[string]any) (bool, error) {
			return false, errors.New("the statistics table is missing")
		},
	}

	findings, err := reviewWithChecks(context.Background(), dialect, 1, "mayfly", "DELETE FROM orders", []trigger.CheckDef{unverifiable})
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("an unverifiable check must still be reported, got %+v", findings)
	}
	finding := findings[0]
	if finding.Note == "" || !strings.Contains(finding.Note, "statistics table is missing") {
		t.Fatalf("the note must say why nothing could be concluded, got %q", finding.Note)
	}
	// 没结论就不是命中：不能带上命中原因与建议级别，否则用户会把它读成「这条规则已拦下」
	if finding.Reason != "" || finding.Severity != flowentity.Severity(0) {
		t.Fatalf("an unverifiable finding must not look like a hit, got %+v", finding)
	}
}

// TestRowBasedChecksSkipStatementsWithoutRows 「影响行数」对 DDL 与查询不存在，它们不该拿到行规则结论。
//
// mysql 的 EXPLAIN 不接受 ALTER，不拦这一层的话，每条 DDL 都会被 fail-closed 报成
// 「估算行数超阈值」——看着像安全信号，实际是一条毫无信息量的假告警，久而久之只能被关掉
func TestRowBasedChecksSkipStatementsWithoutRows(t *testing.T) {
	stubNoDbConn(t)
	dialect := reviewDialect(t)

	for _, sql := range []string{"ALTER TABLE orders ADD COLUMN note varchar(200)", "SELECT * FROM orders", "CREATE INDEX idx_orders ON orders (status)"} {
		findings, err := reviewStatement(context.Background(), dialect, 1, "mayfly", sql)
		if err != nil {
			t.Fatalf("review failed: %v", err)
		}
		for _, key := range []string{dbCheckRowsExceeds, dbCheckFullScan, dbCheckNoPrimaryKey} {
			if got := findingsOf(findings, key); len(got) != 0 {
				t.Fatalf("statement %q must not get a verdict of %s, got %+v", sql, key, got)
			}
		}
	}
}

// castString 详情里的值是 any，测试只关心可读文案
func castString(v any) string {
	s, _ := v.(string)
	return s
}

package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	flowapp "mayfly-go/internal/flow/application"
	flowdto "mayfly-go/internal/flow/application/dto"
	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/trigger"
)

// fakeProcdefReader 只提供触发求值所需的能力，其余接口方法由嵌入的接口占位
type fakeProcdefReader struct {
	flowapp.Procdef

	// governed 该资源绑定的流程定义，nil 表示未纳管
	governed *flowentity.Procdef

	// broken 非空表示策略求值本身失败（脏数据、检查项报错等），用于验证「判不出来」时不会放行
	broken error
}

func (f fakeProcdefReader) GetProcdefByCodePath(_ context.Context, _ ...string) *flowentity.Procdef {
	return f.governed
}

func (f fakeProcdefReader) CheckProcdefTrigger(_ context.Context, procdef *flowentity.Procdef, req *flowdto.TriggerRequest) (*trigger.Decision, error) {
	if f.broken != nil {
		return nil, f.broken
	}
	tc := trigger.NewContext(context.Background(), req.BizType, 1, req.Raw, nil).WithAttributes(req.Attributes)
	return trigger.Evaluate(context.Background(), procdef.TriggerPolicy, tc), nil
}

// defaultDbPolicy 迁移为存量流程定义写入的默认规则包（DBMS 部分）
func defaultDbPolicy() *flowentity.TriggerPolicy {
	return &flowentity.TriggerPolicy{
		Version:         flowentity.TriggerPolicyVersion,
		DefaultSeverity: flowentity.SeverityRequired,
		Checks: []*flowentity.CheckConfig{
			{
				Key: dbCheckDmlRequiresApproval, BizType: DbSQLExecFlowBizType, Severity: flowentity.SeverityRequired,
				Params: map[string]any{"stmtTypes": []any{"insert", "update", "delete", "ddl"}},
			},
			{Key: dbCheckDmlWithoutWhere, BizType: DbSQLExecFlowBizType, Severity: flowentity.SeverityRequired},
		},
	}
}

// 默认规则包必须能通过保存校验，否则升级后流程定义一保存就被拒
func TestDefaultDbRulePackPassesValidation(t *testing.T) {
	if err := trigger.ValidatePolicy(defaultDbPolicy()); err != nil {
		t.Fatalf("the default db rule pack must be valid, got %v", err)
	}
}

func TestDbTriggerChecks(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		stmt sqlstmt.Stmt
		want flowentity.Severity
	}{
		{"查询语句放行", "SELECT * FROM t", &sqlstmt.SelectStmt{}, flowentity.SeverityDisabled},
		{"更新语句需审批", "UPDATE t SET a = 1 WHERE id = 2", &sqlstmt.UpdateStmt{Tables: []sqlstmt.TableRef{{Name: "t"}}, Where: &sqlstmt.Expr{Text: "id = 2"}}, flowentity.SeverityRequired},
		{"无 WHERE 的删除需审批", "DELETE FROM t", &sqlstmt.DeleteStmt{Tables: []sqlstmt.TableRef{{Name: "t"}}}, flowentity.SeverityRequired},
		{"DDL 需审批", "ALTER TABLE t ADD c int", &sqlstmt.DdlStmt{}, flowentity.SeverityRequired},
		{"只读归类放行", "SHOW TABLES", &sqlstmt.OtherStmt{}, flowentity.SeverityDisabled},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stmtType := string(sqlstmt.StmtTypeOther)
			switch tc.stmt.(type) {
			case *sqlstmt.SelectStmt:
				stmtType = string(sqlstmt.StmtTypeSelect)
			case *sqlstmt.UpdateStmt:
				stmtType = string(sqlstmt.StmtTypeUpdate)
			case *sqlstmt.DeleteStmt:
				stmtType = string(sqlstmt.StmtTypeDelete)
			case *sqlstmt.InsertStmt:
				stmtType = string(sqlstmt.StmtTypeInsert)
			case *sqlstmt.DdlStmt:
				stmtType = string(sqlstmt.StmtTypeDDL)
			}

			ctx := trigger.NewContext(context.Background(), DbSQLExecFlowBizType, 1, map[string]string{"stmtType": stmtType, "sql": tc.sql}, nil)
			ctx = ctx.WithAttributes(sqlTriggerFacts(tc.sql, tc.stmt))
			decision := trigger.Evaluate(context.Background(), defaultDbPolicy(), ctx)
			if decision.Severity != tc.want {
				t.Fatalf("expected severity %v but got %v (findings=%+v)", tc.want, decision.Severity, decision.Findings)
			}
		})
	}
}

// 无 WHERE 检查只对 UPDATE/DELETE 生效，否则每条 INSERT 都会被误判为高危
func TestDmlWithoutWhereIgnoresInsert(t *testing.T) {
	sql := "INSERT INTO t (a) VALUES (1)"
	ctx := trigger.NewContext(context.Background(), DbSQLExecFlowBizType, 1, map[string]string{"stmtType": "insert", "sql": sql}, nil).
		WithAttributes(sqlTriggerFacts(sql, &sqlstmt.InsertStmt{Table: sqlstmt.TableRef{Name: "t"}}))

	policy := &flowentity.TriggerPolicy{
		Version: flowentity.TriggerPolicyVersion,
		Checks:  []*flowentity.CheckConfig{{Key: dbCheckDmlWithoutWhere, BizType: DbSQLExecFlowBizType, Severity: flowentity.SeverityRequired}},
	}
	if decision := trigger.Evaluate(context.Background(), policy, ctx); decision.Severity != flowentity.SeverityDisabled {
		t.Fatalf("an insert without a where clause is normal, got %v", decision.Severity)
	}
}

// 破坏性 DDL 检查必须忽略关键字大小写，否则 `TRUNCATE` 与 `truncate` 会得到不同结论
func TestDestructiveDdlIsCaseInsensitive(t *testing.T) {
	def, ok := trigger.CheckOf(dbCheckDestructiveDdl)
	if !ok {
		t.Fatalf("the check %q must be registered", dbCheckDestructiveDdl)
	}
	params := map[string]any{"keywords": []any{"drop", "truncate"}}

	for _, sql := range []string{"TRUNCATE TABLE orders", "truncate table orders", "Drop INDEX idx ON t"} {
		ctx := trigger.NewContext(context.Background(), DbSQLExecFlowBizType, 1, map[string]string{"stmtType": "ddl", "sql": sql}, nil)
		matched, err := def.Evaluate(context.Background(), ctx, params)
		if err != nil || !matched {
			t.Fatalf("%q must be recognised as destructive DDL, got %v %v", sql, matched, err)
		}
	}
	ctx := trigger.NewContext(context.Background(), DbSQLExecFlowBizType, 1, map[string]string{"stmtType": "ddl", "sql": "CREATE TABLE t (id int)"}, nil)
	if matched, _ := def.Evaluate(context.Background(), ctx, params); matched {
		t.Fatalf("a plain CREATE TABLE must not match the destructive keywords")
	}
}

// 语句类型不在参数集合内时，检查项不得命中（防止「参数留空 = 全拦」的静默扩大）
func TestDmlRequiresApprovalWithEmptyTypesNeverMatches(t *testing.T) {
	def, _ := trigger.CheckOf(dbCheckDmlRequiresApproval)
	ctx := trigger.NewContext(context.Background(), DbSQLExecFlowBizType, 1, map[string]string{"stmtType": "update", "sql": "UPDATE t SET a = 1"}, nil)
	matched, err := def.Evaluate(context.Background(), ctx, map[string]any{"stmtTypes": []any{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if matched {
		t.Fatalf("an empty selection must not match every statement")
	}
}

func TestSqlTriggerFactsFromAst(t *testing.T) {
	sql := "UPDATE orders a SET a.x = 1 WHERE a.id = 2"
	attributes := sqlTriggerFacts(sql, &sqlstmt.UpdateStmt{Tables: []sqlstmt.TableRef{{Schema: "shop", Name: "orders", Alias: "a"}}, Where: &sqlstmt.Expr{Text: "a.id = 2"}})

	if attributes["sqlHasWhere"] != true {
		t.Fatalf("the update statement has a where clause, got %v", attributes["sqlHasWhere"])
	}
	tables, ok := attributes["targetTables"].([]string)
	if !ok || len(tables) != 1 || tables[0] != "shop.orders" {
		t.Fatalf("the target table must keep its schema, got %v", attributes["targetTables"])
	}
	if attributes["sqlLength"] != len(sql) {
		t.Fatalf("the sql length must be reported, got %v", attributes["sqlLength"])
	}
}

// 超大 SQL 仍要走完全部检查项：审批前置校验没有「输入太大就跳过」的旁路，
// 否则一条塞满注释的无 WHERE 删除即可绕过防护
func TestOversizedSqlStillEvaluatesEveryCheck(t *testing.T) {
	huge := "DELETE FROM t /* " + strings.Repeat("a", 3<<20) + " */"
	ctx := trigger.NewContext(context.Background(), DbSQLExecFlowBizType, 1, map[string]string{"stmtType": "delete", "sql": huge}, nil).
		WithAttributes(sqlTriggerFacts(huge, &sqlstmt.DeleteStmt{}))

	policy := &flowentity.TriggerPolicy{
		Version: flowentity.TriggerPolicyVersion,
		Checks: []*flowentity.CheckConfig{
			{Key: dbCheckSqlSizeExceeds, BizType: DbSQLExecFlowBizType, Severity: flowentity.SeverityWarning, Params: map[string]any{"maxKb": 1.0}},
			{Key: dbCheckDmlWithoutWhere, BizType: DbSQLExecFlowBizType, Severity: flowentity.SeverityForbidden},
		},
	}
	decision := trigger.Evaluate(context.Background(), policy, ctx)
	if decision.Severity != flowentity.SeverityForbidden {
		t.Fatalf("an oversized statement must not weaken the no-WHERE rule, got %v", decision.Severity)
	}
}

// 拦截提示必须带上命中原因：只说「需提交工单」的人无法知道是哪条规则拦的，
// 也就无法判断该改 SQL 还是该找管理员调策略
func TestBlockedMessageCarriesReason(t *testing.T) {
	// i18n 默认语言即中文，无需在上下文里额外指定
	ctx := context.Background()

	sql := "DELETE FROM orders"
	param := &sqlExecParam{
		Procdef: &flowentity.Procdef{TriggerPolicy: &flowentity.TriggerPolicy{
			Version: flowentity.TriggerPolicyVersion,
			Checks: []*flowentity.CheckConfig{
				{Key: dbCheckDmlWithoutWhere, BizType: DbSQLExecFlowBizType, Severity: flowentity.SeverityRequired},
			},
		}},
		DbConn: &dbi.DbConn{},
		SQL:    sql,
		Stmt:   &sqlstmt.DeleteStmt{},
	}
	app := &dbSQLExecAppImpl{flowProcdefApp: fakeProcdefReader{}}
	err := app.checkSQLTrigger(ctx, param, "delete")
	if err == nil {
		t.Fatalf("a delete without WHERE must be blocked")
	}
	msg := err.Error()
	if !strings.Contains(msg, "缺少 WHERE") {
		t.Fatalf("the block message must name the matched rule, got %q", msg)
	}
	// 原因不应泄露未命中的规则
	if strings.Contains(msg, "体积") {
		t.Fatalf("unmatched rules must not leak into the reason, got %q", msg)
	}
}

// 提醒级结论要能随成功结果回传，否则「仅提醒」只进服务端日志等于没配
func TestWarningNoticesRideAlongWithResult(t *testing.T) {
	param := &sqlExecParam{
		Procdef: &flowentity.Procdef{TriggerPolicy: &flowentity.TriggerPolicy{
			Version: flowentity.TriggerPolicyVersion,
			Checks: []*flowentity.CheckConfig{
				{Key: dbCheckSqlSizeExceeds, BizType: DbSQLExecFlowBizType, Severity: flowentity.SeverityWarning, Params: map[string]any{"maxKb": 0.001}},
			},
		}},
		SQL: "SELECT " + strings.Repeat("x", 4096),
	}
	app := &dbSQLExecAppImpl{flowProcdefApp: fakeProcdefReader{}}
	if err := app.checkSQLTrigger(context.Background(), param, "select"); err != nil {
		t.Fatalf("a warning must not block execution, got %v", err)
	}
	if len(param.Notices) != 1 || param.Notices[0].Title != "flow.check.sqlSizeExceeds" {
		t.Fatalf("the warning must be reported back to the operator, got %+v", param.Notices)
	}
}

// 查询语句的目标表必须解析出来：否则「指定表需审批」这类按表配置的策略对 SELECT
// 恒不命中，管理员看到界面配好了，实际完全没生效
func TestSelectTargetTablesAreExtracted(t *testing.T) {
	stmt := &sqlstmt.SelectStmt{
		From:  []sqlstmt.TableRef{{Name: "orders"}, {Schema: "shop", Name: "users"}, {Name: "orders"}},
		Joins: []sqlstmt.JoinClause{{Table: sqlstmt.TableRef{Name: "order_items"}}},
		Unions: []sqlstmt.UnionClause{{Select: &sqlstmt.SelectStmt{
			From: []sqlstmt.TableRef{{Name: "archived_orders"}},
		}}},
	}

	attributes := sqlTriggerFacts("SELECT 1", stmt)
	tables, ok := attributes["targetTables"].([]string)
	if !ok {
		t.Fatalf("a select statement must expose its target tables, got %v", attributes["targetTables"])
	}
	want := map[string]bool{"orders": true, "shop.users": true, "order_items": true, "archived_orders": true}
	if len(tables) != len(want) {
		t.Fatalf("duplicates must be collapsed and every referenced table kept, got %v", tables)
	}
	for _, name := range tables {
		if !want[name] {
			t.Fatalf("unexpected table %q in %v", name, tables)
		}
	}

	// 纯常量查询没有表，就不能凭空给出 targetTables：字段缺失与「表列表为空」在条件里语义不同
	if _, exist := sqlTriggerFacts("SELECT 1", &sqlstmt.SelectStmt{})["targetTables"]; exist {
		t.Fatalf("a select without tables must not report the field at all")
	}
}

// 只命中自定义条件时，拦截提示也必须给出原因，不能是空括号
func TestBlockedMessageNamesCustomCondition(t *testing.T) {
	ctx := context.Background()
	param := &sqlExecParam{
		Procdef: &flowentity.Procdef{TriggerPolicy: &flowentity.TriggerPolicy{
			Version:         flowentity.TriggerPolicyVersion,
			DefaultSeverity: flowentity.SeverityDisabled,
			Customs: []*flowentity.CustomCondition{{
				BizType:  DbSQLExecFlowBizType,
				Severity: flowentity.SeverityRequired,
				When:     &flowentity.RuleNode{Kind: flowentity.NodeKindCondition, Field: dbFieldSQL, Op: trigger.OpContains, Value: "tp_e2e"},
			}},
		}},
		DbConn: &dbi.DbConn{},
		SQL:    "SELECT * FROM tp_e2e_orders",
		Stmt:   &sqlstmt.SelectStmt{},
	}
	app := &dbSQLExecAppImpl{flowProcdefApp: fakeProcdefReader{}}
	err := app.checkSQLTrigger(ctx, param, "select")
	if err == nil {
		t.Fatalf("a statement matching the custom condition must be blocked")
	}
	msg := err.Error()
	if !strings.Contains(msg, "自定义") {
		t.Fatalf("the block message must explain the custom condition, got %q", msg)
	}
	if strings.Contains(msg, "（）") || strings.Contains(msg, "( )") {
		t.Fatalf("the block message must not carry empty parentheses, got %q", msg)
	}
}

// 需审批与禁止执行必须能被程序区分开：前者在拦截处给「提交工单」入口，后者给了就是把
// 用户引向一条走不通的路（批完也不会执行）。所以判定走错误码，不走提示文案
func TestApprovalAndForbiddenAreDistinguishable(t *testing.T) {
	ctx := context.Background()
	app := &dbSQLExecAppImpl{flowProcdefApp: fakeProcdefReader{}}

	newParam := func(severity flowentity.Severity) *sqlExecParam {
		return &sqlExecParam{
			Procdef: &flowentity.Procdef{TriggerPolicy: &flowentity.TriggerPolicy{
				Version: flowentity.TriggerPolicyVersion,
				Checks: []*flowentity.CheckConfig{
					{Key: dbCheckDmlWithoutWhere, BizType: DbSQLExecFlowBizType, Severity: severity},
				},
			}},
			DbConn: &dbi.DbConn{},
			SQL:    "DELETE FROM orders",
			Stmt:   &sqlstmt.DeleteStmt{},
		}
	}

	approvalErr := app.checkSQLTrigger(ctx, newParam(flowentity.SeverityRequired), "delete")
	if approvalErr == nil {
		t.Fatalf("the statement must be blocked for approval")
	}
	if !flowapp.IsNeedApprovalError(approvalErr) {
		t.Fatalf("a rule requiring a work order must carry the need-approval code, got %v", approvalErr)
	}

	forbiddenErr := app.checkSQLTrigger(ctx, newParam(flowentity.SeverityForbidden), "delete")
	if forbiddenErr == nil {
		t.Fatalf("the statement must be forbidden")
	}
	if flowapp.IsNeedApprovalError(forbiddenErr) {
		t.Fatalf("a forbidden operation must not be reported as needing approval, got %v", forbiddenErr)
	}

	// 未命中任何规则时不放行标记：成功执行的路径没有提单入口
	missed := app.checkSQLTrigger(ctx, newParam(flowentity.SeverityDisabled), "delete")
	if flowapp.IsNeedApprovalError(missed) {
		t.Fatalf("a disabled check must not raise the need-approval code, got %v", missed)
	}
}

// TestUnattendedSqlPolicyIsEnforced 拿不到工单通道的入口必须消费策略结论，且一批语句里任何一条被拒就整批不执行。
//
// 这里曾经的行为是：判定出错 continue 放行、「需审批」只记一行日志（注释写着「用户已批准」，
// 把 Agent 里点的执行确认当成工单审批），且从不判「禁止执行」——管理员拦得住编辑器却拦不住 Agent
func TestAgentSqlPolicyIsEnforced(t *testing.T) {
	ctx := context.Background()
	dialect := dbi.GetDialect(dbi.ToDbType("mysql"))
	if dialect == nil {
		t.Fatal("mysql 方言未注册，测试无法进行")
	}
	// 默认规则包：insert/update/delete/ddl 需审批 + 无 WHERE 的更新删除需审批
	governedApp := &dbSQLExecAppImpl{flowProcdefApp: fakeProcdefReader{governed: &flowentity.Procdef{TriggerPolicy: defaultDbPolicy()}}}

	if _, err := governedApp.checkSqlsWithoutTicket(ctx, []string{"default/2|x/"}, dialect, []string{"DELETE FROM t"}); err == nil {
		t.Fatal("Agent 拿不到工单通道，需审批结论必须按拒绝处理")
	} else if !strings.Contains(err.Error(), "工单") {
		t.Fatalf("拒绝原因要说明该走工单审批, got %q", err.Error())
	}

	// 一批语句里有条目被拒就整批拒绝：只跳过那条会让模型以为「执行了前面那些」
	if _, err := governedApp.checkSqlsWithoutTicket(ctx, []string{"default/2|x/"}, dialect, []string{"SELECT 1", "DELETE FROM t"}); err == nil {
		t.Fatal("不得跳过被拒的语句继续执行其余部分")
	}

	warnPolicy := &flowentity.TriggerPolicy{
		Version: flowentity.TriggerPolicyVersion,
		Checks: []*flowentity.CheckConfig{
			{Key: dbCheckSqlSizeExceeds, BizType: DbSQLExecFlowBizType, Severity: flowentity.SeverityWarning, Params: map[string]any{"maxKb": 0.001}},
		},
	}
	notices, err := (&dbSQLExecAppImpl{flowProcdefApp: fakeProcdefReader{governed: &flowentity.Procdef{TriggerPolicy: warnPolicy}}}).
		checkSqlsWithoutTicket(ctx, []string{"default/2|x/"}, dialect, []string{"SELECT " + strings.Repeat("x", 4096)})
	if err != nil {
		t.Fatalf("提醒级别不该阻断执行, got %v", err)
	}
	if len(notices) != 1 || notices[0] == "" {
		t.Fatalf("提醒必须交回调用方展示给模型, got %q", notices)
	}

	// 策略求值失败：结论不明确时必须拒绝，不能「判不出来就放行」
	brokenApp := &dbSQLExecAppImpl{flowProcdefApp: fakeProcdefReader{governed: &flowentity.Procdef{TriggerPolicy: defaultDbPolicy()}, broken: errors.New("policy unreadable")}}
	if _, err := brokenApp.checkSqlsWithoutTicket(ctx, []string{"default/2|x/"}, dialect, []string{"SELECT 1"}); err == nil {
		t.Fatal("策略求值失败不得继续执行")
	}

	// 未绑定流程定义：不治理也不报错，且不为每条语句付解析开销
	freeNotices, err := (&dbSQLExecAppImpl{flowProcdefApp: fakeProcdefReader{}}).checkSqlsWithoutTicket(ctx, []string{"default/2|x/"}, dialect, []string{"DELETE FROM t"})
	if err != nil {
		t.Fatalf("未纳管资源不该被拦, got %v", err)
	}
	if len(freeNotices) != 1 || freeNotices[0] != "" {
		t.Fatalf("未纳管时不该有提醒, got %q", freeNotices)
	}
}

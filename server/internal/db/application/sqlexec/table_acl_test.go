package sqlexec

// 表级权限的判定语义与接线守卫。
//
// 这里的用例盯的是「配了能不能拦住」的两类失效：
//   1. 表名写法差异导致漏判（大小写、schema 前缀、DDL 文本里的引号）；
//   2. 判定点漏接某个入口，或接在「已经决定放行」之后。
//
// 运行：cd server && go test -count=1 -run 'TestTableAcl' ./internal/db/application/sqlexec/

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

func TestTableAclSnapshotDeniedBy(t *testing.T) {
	snap := &tableAclSnapshot{rules: map[string][]uint64{
		"t_account": {},  // 空角色集合 = 对所有账号生效
		"t_order":   {7}, // 仅角色 7
		"t_log":     {8, 9},
	}}
	cases := []struct {
		name   string
		roles  []uint64
		tables []string
		want   string
	}{
		{"everyone", nil, []string{"t_account"}, "t_account"},
		{"role hit", []uint64{7}, []string{"t_order"}, "t_order"},
		{"role miss", []uint64{1}, []string{"t_order"}, ""},
		{"no roles at all", nil, []string{"t_order"}, ""},
		{"case insensitive", []uint64{7}, []string{"T_ORDER"}, "T_ORDER"},
		{"schema qualified", []uint64{7}, []string{"public.t_order"}, "public.t_order"},
		{"backticked", []uint64{7}, []string{"`t_order`"}, "`t_order`"},
		{"other table", []uint64{7}, []string{"t_user"}, ""},
		{"one of many", []uint64{1, 9}, []string{"t_user", "t_log"}, "t_log"},
		{"empty table list", []uint64{7}, nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := snap.deniedBy(tc.roles, false, tc.tables); got != tc.want {
				t.Fatalf("deniedBy = %q, want %q", got, tc.want)
			}
		})
	}

	// 账号身份未知（角色未解析）时，角色维度规则按命中处理；
	// 空角色集合的那条本来就对所有人生效，同样命中
	if got := snap.deniedBy(nil, true, []string{"t_order"}); got != "t_order" {
		t.Fatalf("unresolved roles must deny role-scoped rules, got %q", got)
	}
	if got := snap.deniedBy(nil, true, []string{"t_user"}); got != "" {
		t.Fatalf("unresolved roles must not deny unlisted tables, got %q", got)
	}

	// 没有规则时一律放行：这是「没配保护」的正常路径，不能变成全域封锁
	empty := &tableAclSnapshot{}
	if got := empty.deniedBy(nil, false, []string{"t_account"}); got != "" {
		t.Fatalf("an empty rule set must allow everything, got %q", got)
	}
}

// TestTableAclKeyNormalization 归一化是上面那些用例的底座：这里直接钉死它的规则。
func TestTableAclKeyNormalization(t *testing.T) {
	cases := map[string]string{
		"t_order":            "t_order",
		" T_ORDER ":          "t_order",
		"public.t_order":     "t_order",
		"`t_order`":          "t_order",
		`"t_order"`:          "t_order",
		"[t_order]":          "t_order",
		"`public`.`t_order`": "t_order",
	}
	for input, want := range cases {
		if got := aclTableKey(input); got != want {
			t.Fatalf("aclTableKey(%q) = %q, want %q", input, got, want)
		}
	}
}

// TestTableAclGuardPlacement 判定点必须接在「决定放行之前」。
//
// checkSQLTrigger 是编辑器 / 数据网格 / 脚本执行共用的闸门，assertTableAcl 必须早于
// 策略求值——策略求值在未绑定流程定义时直接返回 nil（放行），接在其后就等于漏判。
func TestTableAclGuardPlacement(t *testing.T) {
	source, err := os.ReadFile("db_trigger.go")
	if err != nil {
		t.Fatalf("read db_trigger.go: %v", err)
	}
	body := string(source)

	// 1) 交互执行闸门
	start := strings.Index(body, "func (d *DbSQLExecAppImpl) checkSQLTrigger(")
	if start == -1 {
		t.Fatal("checkSQLTrigger 应存在")
	}
	fn := body[start:]
	if end := strings.Index(fn[1:], "\nfunc "); end != -1 {
		fn = fn[:end+1]
	}
	aclPos := strings.Index(fn, "d.assertTableAcl(ctx, sqlExecParam)")
	evalPos := strings.Index(fn, "d.evaluateSQLTrigger(ctx, sqlExecParam, stmtType)")
	if aclPos == -1 || evalPos == -1 {
		t.Fatalf("checkSQLTrigger 必须同时含表级权限判定与策略求值 (acl=%d eval=%d)", aclPos, evalPos)
	}
	if aclPos > evalPos {
		t.Fatal("表级权限必须在策略求值之前：策略在未绑定流程定义时直接放行")
	}

	// 2) 无工单通道的入口（导出 / 导入 / AI）：判定必须早于「procdef == nil 就返回」
	start = strings.Index(body, "func (d *DbSQLExecAppImpl) checkSqlsWithoutTicket(")
	if start == -1 {
		t.Fatal("checkSqlsWithoutTicket 应存在")
	}
	fn = body[start:]
	if end := strings.Index(fn[1:], "\nfunc "); end != -1 {
		fn = fn[:end+1]
	}
	enabledPos := strings.Index(fn, "d.tableAclEnabled(")
	earlyReturnPos := strings.Index(fn, "if procdef == nil && !aclEnabled")
	loopAclPos := strings.Index(fn, "d.assertTableAcl(ctx, sqlExec)")
	if enabledPos == -1 || earlyReturnPos == -1 || loopAclPos == -1 {
		t.Fatalf("checkSqlsWithoutTicket 必须载入表级权限并在循环内判定 (enabled=%d early=%d loop=%d)",
			enabledPos, earlyReturnPos, loopAclPos)
	}
	if enabledPos > earlyReturnPos {
		t.Fatal("必须在提前返回之前确认是否配了保护名单")
	}
	loopStart := strings.Index(fn, "for i, sql := range sqls")
	if loopStart == -1 {
		t.Fatal("应存在逐条语句的循环")
	}
	loopBody := fn[loopStart:]
	loopAcl := strings.Index(loopBody, "d.assertTableAcl(ctx, sqlExec)")
	loopEval := strings.Index(loopBody, "d.evaluateSQLTrigger(")
	if loopAcl == -1 || loopEval == -1 {
		t.Fatalf("循环内必须同时含表级权限与策略求值 (acl=%d eval=%d)", loopAcl, loopEval)
	}
	if loopAcl > loopEval {
		t.Fatal("表级权限判定必须在策略求值之前")
	}
}

// TestTableAclWhitelistMode 白名单语义：库内出现 allow 规则后，
// 未列出的表对所有人拒绝，列出的表按角色放行，deny 规则始终优先。
func TestTableAclWhitelistMode(t *testing.T) {
	snap := &tableAclSnapshot{
		rules:  map[string][]uint64{"t_secret": {}},
		allows: map[string][]uint64{"t_order": {}, "t_user": {7}},
	}
	if !snap.whitelistMode() {
		t.Fatal("a snapshot with allow rules must be in whitelist mode")
	}

	// 列出的表：无角色限制的那条对所有人放行
	if !snap.allowedBy(nil, false, "t_order") {
		t.Fatal("an allow rule without roles must apply to every account")
	}
	// 列出的表：角色不匹配时不算放行
	if snap.allowedBy([]uint64{1}, false, "t_user") {
		t.Fatal("an allow rule scoped to another role must not admit this account")
	}
	if !snap.allowedBy([]uint64{7}, false, "t_user") {
		t.Fatal("the matching role must be admitted")
	}
	// 身份未解析：与表级一致按命中处理（证明不了有权就放行会等于摘掉白名单）
	if !snap.allowedBy(nil, true, "t_user") {
		t.Fatal("unresolved roles must be treated as matching (fail-closed for the deny direction)")
	}
	// 未列出的表一律不在允许清单
	if snap.allowedBy(nil, false, "t_other") {
		t.Fatal("a table outside the allow list must not be admitted")
	}

	// deny 规则优先于白名单
	if got := snap.deniedBy(nil, false, []string{"t_secret"}); got != "t_secret" {
		t.Fatalf("deny rules must win over the allow list, got %q", got)
	}
	// 只有 deny 规则时不是白名单模式（否则一配就把全库挡住）
	denyOnly := &tableAclSnapshot{rules: map[string][]uint64{"t_secret": {}}}
	if denyOnly.whitelistMode() {
		t.Fatal("deny-only snapshots must keep blacklist semantics")
	}
}

// TestAclTableScopeMarksUntrustworthyRefs 语句级 ACL 的「表引用可信度」判定。
//
// 这组断言的动机是一个真实绕过：派生表在 AST 里只是一个名为 "(SELECT ...)" 的
// 表引用，INSERT…SELECT 只有写入目标，条件子查询根本不进清单——三条路的 tables
// 都"合法且为空"，只按表清单判就等于放行。
func TestAclTableScopeMarksUntrustworthyRefs(t *testing.T) {
	selectStmt := &sqlstmt.SelectStmt{From: []sqlstmt.TableRef{{Name: "t_secret"}}}
	derivedStmt := &sqlstmt.SelectStmt{From: []sqlstmt.TableRef{{Name: "(SELECT * FROM t_secret)"}}}
	emptySelect := &sqlstmt.SelectStmt{}
	insertStmt := &sqlstmt.InsertStmt{Table: sqlstmt.TableRef{Name: "t_public"}}

	cases := []struct {
		name       string
		sql        string
		stmt       sqlstmt.Stmt
		resolvable bool
	}{
		{"普通查询", "select * from t_secret", selectStmt, true},
		{"派生表内层不可见", "select * from (select * from t_secret) x", derivedStmt, false},
		{"无表查询可放行", "select 1", emptySelect, true},
		{"INSERT…SELECT 来源不可见", "insert into t_public select * from t_secret", insertStmt, false},
		{"INSERT…VALUES 可信", "insert into t_public(a) values (1)", insertStmt, true},
		{"解析失败一律不可信", "call some_proc()", nil, false},
	}
	for _, c := range cases {
		if _, resolvable := aclTableScope(c.sql, c.stmt); resolvable != c.resolvable {
			t.Fatalf("%s: resolvable = %v, want %v", c.name, resolvable, c.resolvable)
		}
	}
}

// TestAssertStatementAllowedFailsClosedOnUnresolvable 配了规则且表引用不可信时拒绝执行；
// 没配规则的库保持既有行为（否则解析失败会把整库查询挡死，而不是挡住越权）。
func TestAssertStatementAllowedFailsClosedOnUnresolvable(t *testing.T) {
	newApp := func(rules map[string][]uint64) *tableAclAppImpl {
		app := &tableAclAppImpl{cache: map[uint64]*tableAclSnapshot{}, expires: map[uint64]time.Time{}}
		app.cache[9] = &tableAclSnapshot{rules: rules, allows: map[string][]uint64{}, columns: map[string]map[string][]uint64{}}
		app.expires[9] = time.Now().Add(time.Minute)
		return app
	}

	if err := newApp(map[string][]uint64{"t_secret": nil}).AssertStatementAllowed(context.Background(), 9, nil, false, nil); err == nil {
		t.Fatal("unresolvable statement must be denied when the database has rules")
	}
	if err := newApp(map[string][]uint64{}).AssertStatementAllowed(context.Background(), 9, nil, false, nil); err != nil {
		t.Fatalf("database without rules must not deny unresolvable statements, got %v", err)
	}
	// SELECT 1 这类「确实没有表」的语句不受影响
	if err := newApp(map[string][]uint64{"t_secret": nil}).AssertStatementAllowed(context.Background(), 9, nil, true, nil); err != nil {
		t.Fatalf("statement without tables must pass, got %v", err)
	}
}

// TestAclOtherStatementClassification OTHER 类语句的两难必须用白名单解：
// SHOW/EXPLAIN/SET 不读业务表，挡了就没法用；CALL/EXECUTE 的过程体在客户端看不见，
// 放行就是一条稳定的绕过通道。新增形态默认「不可判定」。
func TestAclOtherStatementClassification(t *testing.T) {
	other := func() sqlstmt.Stmt { return &sqlstmt.OtherStmt{} }
	safe := []string{
		"SHOW TABLES",
		"show processlist",
		"EXPLAIN SELECT 1",
		"DESC t_a",
		"SET @x = 1",
		"-- 注释\nSHOW VARIABLES LIKE 'log_bin'",
		"/* 块注释 */ SHOW STATUS",
	}
	for _, sql := range safe {
		if _, resolvable := aclTableScope(sql, other()); !resolvable {
			t.Fatalf("%q must be treated as table-free", sql)
		}
	}
	unsafe := []string{"CALL some_proc()", "EXECUTE stmt1", "DO SLEEP(1)", "PREPARE s FROM 'select 1'"}
	for _, sql := range unsafe {
		if _, resolvable := aclTableScope(sql, other()); resolvable {
			t.Fatalf("%q must be treated as unresolvable", sql)
		}
	}
}

// TestTrimDeniedColumnsSkipsWhenNoTables 空表清单不裁剪：否则会按列名与任意保护表
// 撞名而静默删列，用户得到一份「成功但少了列」的结果集。
func TestTrimDeniedColumnsSkipsWhenNoTables(t *testing.T) {
	app := &tableAclAppImpl{cache: map[uint64]*tableAclSnapshot{}, expires: map[uint64]time.Time{}}
	app.cache[9] = &tableAclSnapshot{
		rules:   map[string][]uint64{},
		allows:  map[string][]uint64{},
		columns: map[string]map[string][]uint64{"t_secret": {"secret": nil}},
	}
	app.expires[9] = time.Now().Add(time.Minute)

	cols := []*dbi.QueryColumn{{Name: "secret"}}
	rows := []map[string]any{{"secret": "x"}}
	trimmedCols, trimmedRows, err := app.TrimDeniedColumns(context.Background(), 9, nil, cols, rows)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(trimmedCols) != 1 || len(trimmedRows) != 1 {
		t.Fatalf("columns must not be trimmed without table context, got %d cols / %d rows", len(trimmedCols), len(trimmedRows))
	}
}

// TestViewDefinitionTables 视图定义里的表引用提取。
//
// 视图是解析盲区：语句里只有视图名，真正被读的基表在定义里。提取不到就必须判「不可解析」，
// 让 ACL 层按有规则即拒绝处理——宁拒不放。
func TestViewDefinitionTables(t *testing.T) {
	ddl := "CREATE ALGORITHM=UNDEFINED VIEW `v_acl_probe` AS select `id`,`name` from `t_ai_skill`"
	refs := viewDefinitionTables(ddl)
	if len(refs) != 1 || refs[0] != "t_ai_skill" {
		t.Fatalf("refs = %v, want [t_ai_skill]", refs)
	}

	// JOIN 多表 + schema 限定 + 去重
	ddl = "CREATE VIEW v AS SELECT * FROM mayfly_go.t_a a JOIN `t_b` b ON a.id=b.id JOIN t_a c ON 1=1"
	refs = viewDefinitionTables(ddl)
	joined := strings.Join(refs, ",")
	if joined != "t_a,t_b" {
		t.Fatalf("refs = %v", refs)
	}

	// 视图定义里套子查询时，内层的表同样要被提取出来（多提取是偏安全的：
	// 只要定义中出现过被保护表，这个视图就不该被放行）
	if refs := viewDefinitionTables("CREATE VIEW v AS SELECT * FROM (SELECT * FROM t_secret) x"); len(refs) != 1 || refs[0] != "t_secret" {
		t.Fatalf("nested view refs = %v, want [t_secret]", refs)
	}
}

// TestVisibleTables 表清单/补全的可见性过滤：被保护的表名不应再下发。
func TestVisibleTables(t *testing.T) {
	newApp := func(rules, allows map[string][]uint64) *tableAclAppImpl {
		app := &tableAclAppImpl{cache: map[uint64]*tableAclSnapshot{}, expires: map[uint64]time.Time{}}
		if rules == nil {
			rules = map[string][]uint64{}
		}
		if allows == nil {
			allows = map[string][]uint64{}
		}
		app.cache[9] = &tableAclSnapshot{rules: rules, allows: allows, columns: map[string]map[string][]uint64{}}
		app.expires[9] = time.Now().Add(time.Minute)
		return app
	}

	// 黑名单：命中的被过滤，其余保留
	app := newApp(map[string][]uint64{"t_secret": nil}, nil)
	visible, err := app.VisibleTables(context.Background(), 9, []string{"t_a", "t_secret", "t_b"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(visible) != 2 || visible[0] != "t_a" || visible[1] != "t_b" {
		t.Fatalf("visible = %v, want [t_a t_b]", visible)
	}

	// 白名单模式：只保留命中 allow 的表
	app = newApp(nil, map[string][]uint64{"t_keep": nil})
	visible, err = app.VisibleTables(context.Background(), 9, []string{"t_keep", "t_other"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(visible) != 1 || visible[0] != "t_keep" {
		t.Fatalf("visible = %v, want [t_keep]", visible)
	}

	// 无规则：全部保留（不能因为配了别的库的规则就隐藏）
	app = newApp(nil, nil)
	visible, _ = app.VisibleTables(context.Background(), 9, []string{"t_a", "t_b"})
	if len(visible) != 2 {
		t.Fatalf("visible = %v, want both", visible)
	}
}

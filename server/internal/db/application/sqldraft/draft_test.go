package sqldraft

// 离线管线测试：用替身元数据 + 脚本化模型输出跑完整链路，不依赖真实数据库与真实模型。
//
// 为什么要能离线跑：SQL 草稿的核心风险是「召回 / 校验 / 自纠 / 权限收敛」这些编排逻辑，
// 它们与具体库无关。用真实库测会变成「环境不稳定导致红灯」，最终没人跑。
// 真实库上的正确性由浏览器端到端与后续 golden 用例覆盖。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"mayfly-go/internal/ai/pkg/textgen"
	"mayfly-go/internal/db/application/sqlutil"
	"mayfly-go/internal/db/dbm/dbi"
	// 方言注册（init 期自注册）；测试里按 mysql 方言做解析与分类
	_ "mayfly-go/internal/db/dbm/dialect/mysql"
)

// ---------- 替身 ----------

type fakeMeta struct {
	tables     []dbi.Table
	columns    map[string][]dbi.Column
	fks        []dbi.ForeignKey
	explain    *dbi.ExplainResult
	explainErr error
}

func (f *fakeMeta) GetTables(_ ...string) ([]dbi.Table, error) { return f.tables, nil }

func (f *fakeMeta) GetColumns(names ...string) ([]dbi.Column, error) {
	out := make([]dbi.Column, 0, 8)
	for _, name := range names {
		out = append(out, f.columns[strings.ToLower(name)]...)
	}
	return out, nil
}

func (f *fakeMeta) GetDbServer() (*dbi.DbServer, error) {
	return &dbi.DbServer{Version: "8.0.36", MajorVersion: 8}, nil
}

func (f *fakeMeta) GetForeignKeys(_ context.Context, _ string, _ string) ([]dbi.ForeignKey, error) {
	return f.fks, nil
}

func (f *fakeMeta) Explain(_ context.Context, _ string) (*dbi.ExplainResult, error) {
	if f.explainErr != nil {
		return nil, f.explainErr
	}
	if f.explain == nil {
		return nil, errors.New("explain unsupported")
	}
	return f.explain, nil
}

type fakeAcl struct {
	visible []string
	denied  map[string]struct{}
}

func (f *fakeAcl) VisibleTables(_ context.Context, _ uint64, tables []string) ([]string, error) {
	if f.visible == nil {
		return tables, nil
	}
	allowed := make(map[string]struct{}, len(f.visible))
	for _, name := range f.visible {
		allowed[strings.ToLower(name)] = struct{}{}
	}
	out := make([]string, 0, len(tables))
	for _, name := range tables {
		if _, ok := allowed[strings.ToLower(name)]; ok {
			out = append(out, name)
		}
	}
	return out, nil
}

func (f *fakeAcl) DeniedColumns(_ context.Context, _ uint64, _ []string) (map[string]struct{}, error) {
	if f.denied == nil {
		return map[string]struct{}{}, nil
	}
	return f.denied, nil
}

func (f *fakeAcl) AssertStatementAllowed(_ context.Context, _ uint64, _ []string, _ bool, _ []sqlutil.ColumnRef) error {
	return nil
}

type fakePort struct {
	// chooseTables 语义召回（表选择题）的回复：按 system 提示识别，避免与生成回复串位
	chooseTables []string
	replies      []string
	index        int
}

func (f *fakePort) Complete(_ context.Context, req *textgen.Request) (string, *textgen.Usage, error) {
	// 按 Purpose 识别阶段，不按系统提示的中文子串：提示词已经迁到 prompts/ 模板文件，
	// 靠子串识别的替身会在文案改写时静默改判成「生成调用」，用例照样绿但测的已不是同一条链路
	if req.Purpose == purposeChoose {
		body, _ := json.Marshal(map[string][]string{"tables": f.chooseTables})
		return string(body), &textgen.Usage{TotalTokens: 12}, nil
	}
	if len(f.replies) == 0 {
		return "", nil, errors.New("no canned reply")
	}
	reply := f.replies[min(f.index, len(f.replies)-1)]
	f.index++
	return reply, &textgen.Usage{TotalTokens: 12}, nil
}

func (f *fakePort) Available(_ context.Context) (bool, error) { return true, nil }

// ---------- 夹具 ----------

func fixtureMeta() *fakeMeta {
	return &fakeMeta{
		tables: []dbi.Table{
			{TableName: "t_user", TableComment: "用户信息表"},
			{TableName: "t_order", TableComment: "订单明细表"},
			{TableName: "t_secret", TableComment: "内部密钥表"},
		},
		columns: map[string][]dbi.Column{
			"t_user": {
				{TableName: "t_user", ColumnName: "id", ColumnType: "bigint", IsPrimaryKey: true, ColumnComment: "用户ID"},
				{TableName: "t_user", ColumnName: "name", ColumnType: "varchar(64)", ColumnComment: "用户名"},
				{TableName: "t_user", ColumnName: "phone", ColumnType: "varchar(20)", ColumnComment: "手机号"},
			},
			"t_order": {
				{TableName: "t_order", ColumnName: "id", ColumnType: "bigint", IsPrimaryKey: true},
				{TableName: "t_order", ColumnName: "user_id", ColumnType: "bigint", ColumnComment: "用户ID"},
				{TableName: "t_order", ColumnName: "amount", ColumnType: "decimal(12,2)", ColumnComment: "下单金额"},
			},
		},
		fks: []dbi.ForeignKey{
			{Table: "t_order", Column: "user_id", RefTable: "t_user", RefColumn: "id"},
		},
	}
}

func newTestRun(t *testing.T, replies []string, acl TableAcl) *draftRun {
	t.Helper()
	return newTestRunWithChooser(t, replies, acl, []string{"t_user", "t_order"})
}

func newTestRunWithChooser(t *testing.T, replies []string, acl TableAcl, chooser []string) *draftRun {
	t.Helper()
	meta := fixtureMeta()
	dialect := dbi.GetDialect(dbi.ToDbType("mysql"))
	if dialect == nil {
		t.Fatal("mysql dialect not registered")
	}
	if acl == nil {
		// 默认：t_secret 不可见，phone 无权
		acl = &fakeAcl{
			visible: []string{"t_user", "t_order"},
			denied:  map[string]struct{}{"phone": {}},
		}
	}
	snap, err := buildSnapshot(context.Background(), meta, dialect, dbi.ToDbType("mysql"), acl, 1, "demo")
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	spec, ok := specOf("draft")
	if !ok {
		t.Fatal("draft intent not registered")
	}
	return &draftRun{
		req:     &Req{DbId: 1, Db: "demo", Intent: "draft", Prompt: "统计用户"},
		spec:    spec,
		snap:    snap,
		meta:    meta,
		dialect: dialect,
		acl:     acl,
		gen:     newGenerator(&fakePort{replies: replies, chooseTables: chooser}),
	}
}

// ---------- 用例 ----------

// 注册完整性：意图里点名的召回/片段/校验必须真实存在。
//
// 这条用例有牙：拼错一个 id 时管线是「跳过」而不是报错——校验会静默不跑，
// 而草稿看起来一切正常。只有这条断言能在编译与运行都绿的情况下把它抓出来。
func TestIntentSpecReferencesAreRegistered(t *testing.T) {
	ids := IntentIds()
	if len(ids) == 0 {
		t.Fatal("no intent registered")
	}
	for _, id := range ids {
		spec, ok := specOf(id)
		if !ok {
			t.Fatalf("intent %s missing", id)
		}
		for _, linkerId := range spec.Linkers {
			if _, ok := linkerOf(linkerId); !ok {
				t.Fatalf("intent %s references unknown linker %q", id, linkerId)
			}
		}
		for _, fragmentId := range spec.Fragments {
			if _, ok := fragmentOf(fragmentId); !ok {
				t.Fatalf("intent %s references unknown fragment %q", id, fragmentId)
			}
		}
		for _, checkerId := range spec.Checkers {
			if _, ok := checkerOf(checkerId); !ok {
				t.Fatalf("intent %s references unknown checker %q", id, checkerId)
			}
		}
	}
}

// 幻觉列名：首轮引用不存在的列 → invalid；自纠一轮后通过 → verified
func TestDraftRepairsUnknownColumn(t *testing.T) {
	run := newTestRun(t, []string{
		`{"statements":[{"sql":"SELECT nmae FROM t_user","notes":"查用户名"}]}`,
		`{"statements":[{"sql":"SELECT name FROM t_user","notes":"查用户名"}]}`,
	}, nil)

	draft, err := run.execute(context.Background())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if draft.Repairs != 1 {
		t.Fatalf("expect 1 repair round, got %d", draft.Repairs)
	}
	if draft.Verification != "verified" {
		t.Fatalf("expect verified after repair, got %s (checks=%+v)", draft.Verification, draft.Checks)
	}
	if len(draft.Statements) != 1 || !strings.Contains(strings.ToLower(draft.Statements[0].SQL), "select name") {
		t.Fatalf("expect repaired statement, got %+v", draft.Statements)
	}
}

// 无权列：既不进上下文，也不在提示文案里回显列名
func TestDraftDeniedColumnIsNeverEchoed(t *testing.T) {
	run := newTestRun(t, []string{
		`{"statements":[{"sql":"SELECT phone FROM t_user"}]}`,
	}, nil)

	draft, err := run.execute(context.Background())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if draft.Verification != "unverified" {
		t.Fatalf("expect unverified, got %s", draft.Verification)
	}
	var restricted *Finding
	for index := range draft.Checks {
		if draft.Checks[index].Code == CodeColumnRestricted {
			restricted = &draft.Checks[index]
		}
	}
	if restricted == nil {
		t.Fatalf("expect column_restricted finding, got %+v", draft.Checks)
	}
	if strings.Contains(strings.ToLower(restricted.Message), "phone") {
		t.Fatalf("denied column name must not be echoed, got %q", restricted.Message)
	}
	// 上下文里也不应出现 phone 列
	for _, table := range run.snap.tables {
		for _, column := range table.Columns {
			if strings.EqualFold(column.Name, "phone") {
				t.Fatal("denied column leaked into snapshot")
			}
		}
	}
}

// 不可见表：按「不在可见范围」处理，不回表名细节
func TestDraftHiddenTableIsReportedAsRestricted(t *testing.T) {
	run := newTestRun(t, []string{
		`{"statements":[{"sql":"SELECT * FROM t_secret"}]}`,
	}, nil)

	draft, err := run.execute(context.Background())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if draft.Verification != "unverified" {
		t.Fatalf("expect unverified, got %s", draft.Verification)
	}
	found := false
	for _, finding := range draft.Checks {
		if finding.Code == CodeTableRestricted {
			found = true
			if strings.Contains(finding.Message, "t_secret") {
				t.Fatalf("hidden table name must not be echoed: %q", finding.Message)
			}
		}
		if finding.Code == CodeUnknownTable {
			t.Fatalf("hidden table must not be reported as unknown: %+v", finding)
		}
	}
	if !found {
		t.Fatalf("expect table_restricted finding, got %+v", draft.Checks)
	}
}

// 写语句提示来自语句类型，不来自模型
func TestDraftWriteStatementNote(t *testing.T) {
	run := newTestRun(t, []string{
		`{"statements":[{"sql":"UPDATE t_user SET name = 'x' WHERE id = 1"}]}`,
	}, nil)

	draft, err := run.execute(context.Background())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	found := false
	for _, finding := range draft.Checks {
		if finding.Code == CodeWriteStatement && finding.Severity == SeverityNote {
			found = true
		}
	}
	if !found {
		t.Fatalf("expect write_statement note, got %+v", draft.Checks)
	}
}

// 召回：描述里没有精确标识符时，靠注释相关性把表带进来，且每张表都带依据
func TestResolveTablesUsesCommentAndRelation(t *testing.T) {
	run := newTestRun(t, []string{`{"statements":[]}`}, nil)
	run.req.Prompt = "按用户统计订单金额"

	picked, reasons, warnings, err := run.resolveTables(context.Background(), run.req.Prompt)
	if err != nil {
		t.Fatalf("resolve tables: %v", err)
	}
	if len(picked) == 0 {
		t.Fatalf("expect picked tables, warnings=%v", warnings)
	}
	joined := strings.Join(picked, ",")
	if !strings.Contains(joined, "t_order") && !strings.Contains(joined, "t_user") {
		t.Fatalf("expect t_order or t_user picked, got %v", picked)
	}
	for _, name := range picked {
		if len(reasons[strings.ToLower(name)]) == 0 {
			t.Fatalf("picked table %s must carry a reason", name)
		}
	}
}

// 契约解析：代码块包裹、裸 SQL 兜底、散文不得被当成 SQL
func TestParseContract(t *testing.T) {
	statements, ok := parseContract("```json\n{\"statements\":[{\"sql\":\"select 1\",\"notes\":\"n\"}]}\n```")
	if !ok || len(statements) != 1 || statements[0].SQL != "select 1" {
		t.Fatalf("fenced json not parsed: %+v ok=%v", statements, ok)
	}
	statements, ok = parseContract("SELECT id FROM t_user")
	if !ok || len(statements) != 1 || statements[0].SQL != "SELECT id FROM t_user" {
		t.Fatalf("raw sql not parsed: %+v ok=%v", statements, ok)
	}
	if _, ok := parseContract("抱歉，我无法完成这个请求。"); ok {
		t.Fatal("prose must not be treated as SQL")
	}
}

// 示例脱敏：字符串字面量内容不得进入上下文
func TestStripStringLiterals(t *testing.T) {
	got := stripStringLiterals("select * from t where phone = '13800000000' and name = 'o''brien'")
	if strings.Contains(got, "13800000000") || strings.Contains(got, "brien") {
		t.Fatalf("literal content leaked: %q", got)
	}
	if !strings.Contains(got, "phone =") {
		t.Fatalf("structure must be kept: %q", got)
	}
}

// 回归：同前缀表不得因「名字相似」被召回
//
// 真机实测踩到过：编辑器里已有 t_sys_log 时，t_sys_file 因共享前缀 t_sys_ 拿到 6 分
// 被当成相关表，于是整族同前缀表被拖进上下文、把真正该用的表挤掉。
func TestResolveTablesIgnoresSharedNamePrefix(t *testing.T) {
	meta := &fakeMeta{
		tables: []dbi.Table{
			{TableName: "t_sys_log"},
			{TableName: "t_sys_file"},
			{TableName: "t_db", TableComment: "数据库资源"},
		},
		columns: map[string][]dbi.Column{
			"t_sys_log":  {{TableName: "t_sys_log", ColumnName: "id"}},
			"t_sys_file": {{TableName: "t_sys_file", ColumnName: "id"}},
			"t_db":       {{TableName: "t_db", ColumnName: "id"}},
		},
	}
	dialect := dbi.GetDialect(dbi.ToDbType("mysql"))
	snap, err := buildSnapshot(context.Background(), meta, dialect, dbi.ToDbType("mysql"), &fakeAcl{}, 1, "demo")
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	spec, _ := specOf("draft")
	run := &draftRun{
		req:  &Req{DbId: 1, Db: "demo", Intent: "draft", Prompt: "查日志"},
		spec: spec, snap: snap, meta: meta, dialect: dialect, acl: &fakeAcl{},
		gen: newGenerator(&fakePort{replies: []string{"{}"}, chooseTables: nil}),
	}

	// 文本里只出现 t_sys_log（模拟「编辑器里已有该语句」），不得把 t_sys_file 也带进来
	picked, reasons, _, err := run.resolveTables(context.Background(), "select * from t_sys_log")
	if err != nil {
		t.Fatalf("resolve tables: %v", err)
	}
	for _, name := range picked {
		if strings.EqualFold(name, "t_sys_file") {
			t.Fatalf("t_sys_file must not be recalled merely for sharing the t_sys_ prefix (reasons=%v)", reasons)
		}
	}
}

// 回归：模型引用了「存在但未进上下文」的表时，列必须仍被核对
//
// 命中过：上下文只装了召回表的列，模型引用另一张真实表时 columnExists 按「不知道」放行，
// 于是带幻觉列的草稿被标成「已通过校验」。
func TestDraftVerifiesColumnsOfTablesOutsideContext(t *testing.T) {
	run := newTestRunWithChooser(t, []string{
		`{"statements":[{"sql":"SELECT not_a_column FROM t_order"}]}`,
	}, nil, []string{"t_user"})

	draft, err := run.execute(context.Background())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if draft.Verification != "invalid" {
		t.Fatalf("expect invalid for hallucinated column on out-of-context table, got %s (checks=%+v)", draft.Verification, draft.Checks)
	}
	found := false
	for _, finding := range draft.Checks {
		if finding.Code == CodeUnknownColumn && finding.Identifier == "not_a_column" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expect unknown_column finding, got %+v", draft.Checks)
	}
}

// 回归：编辑器材料里的表名不得把「需求描述没命中」这件事掩盖掉
//
// 命中过：光标停在 `UPDATE t_sys_log ...` 时，材料召回拿到高分，置信度判定通过，
// 表选择题被跳过，于是「统计每个实例下有多少个库」只带了日志表、需求要的库表一张没进上下文。
func TestChooserStillRunsWhenOnlyEditorMaterialMatches(t *testing.T) {
	// t_order 当前账号可见、且不会出现在材料里，只能由选择题带进来
	run := newTestRunWithChooser(t, []string{`{"statements":[]}`}, nil, []string{"t_order"})
	run.req.Prompt = "统计每个实例下有多少个库"

	picked, reasons, _, err := run.resolveTables(context.Background(), "update t_user set name = 'x'")
	if err != nil {
		t.Fatalf("resolve tables: %v", err)
	}
	found := false
	for _, name := range picked {
		if strings.EqualFold(name, "t_order") {
			found = true
		}
	}
	if !found {
		t.Fatalf("chooser result must be present even though editor material matched a table: picked=%v reasons=%v", picked, reasons)
	}
}

// 回归：聚合函数内的列不得因解析残留（COUNT(d.id) → "id)"）被判成不存在的列
func TestDraftIgnoresExpressionResidueInColumns(t *testing.T) {
	run := newTestRunWithChooser(t, []string{
		`{"statements":[{"sql":"SELECT COUNT(id) AS c FROM t_user"}]}`,
	}, nil, []string{"t_user"})

	draft, err := run.execute(context.Background())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	for _, finding := range draft.Checks {
		if strings.Contains(finding.Identifier, ")") {
			t.Fatalf("expression residue must not be verified as a column: %+v", finding)
		}
	}
	if draft.Verification != "verified" {
		t.Fatalf("expect verified, got %s (checks=%+v)", draft.Verification, draft.Checks)
	}
}

// 超时归类：真机实测上游返回的是 `Post "...": context deadline exceeded (Client.Timeout
// exceeded while awaiting headers)`，若只认 context.DeadlineExceeded 就会把它当普通失败，
// 用户看到的仍是 HTTP 原文。
func TestIsTimeoutError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"ctx deadline", context.DeadlineExceeded, true},
		{"wrapped ctx deadline", fmt.Errorf("complete: %w", context.DeadlineExceeded), true},
		{"http client timeout", errors.New(`Post "https://x/v1/chat/completions": context deadline exceeded (Client.Timeout exceeded while awaiting headers)`), true},
		{"gateway 504", errors.New("upstream returned 504"), true},
		{"plain failure", errors.New("invalid api key"), false},
		{"nil", nil, false},
	}
	for _, one := range cases {
		if got := isTimeoutError(one.err); got != one.want {
			t.Fatalf("%s: got %v want %v", one.name, got, one.want)
		}
	}
}

// 回归：定位不到表时必须失败退出，不得拿种子表硬编
//
// 真机实测的坏结局：选择题超时 → 上下文里只剩编辑器里的种子表 → 模型编出一条读
// t_sys_resource 的假语句，耗时 114s、烧掉一轮自纠才被校验判为 invalid。
func TestDraftFailsClosedWhenNoTableLocated(t *testing.T) {
	// 语义召回返回空表名列表（模型没选出任何表）
	run := newTestRunWithChooser(t, []string{`{"statements":[{"sql":"SELECT 1"}]}`}, nil, nil)
	run.req.Prompt = "统计每个实例下有多少个库"

	if _, err := run.execute(context.Background()); err == nil {
		t.Fatal("expect an error when no related table could be located")
	}
}

// fakeGate 闸门替身：按预设结论返回错误，并声明归类结果
type fakeGate struct {
	err  error
	kind string
}

func (f *fakeGate) CheckSqlsWithoutTicket(_ context.Context, _ *dbi.DbConn, _ []string) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []string{""}, nil
}

func (f *fakeGate) PolicyBlockKind(_ error) string { return f.kind }

// 闸门结论必须分流：需审批/需确认是「结论」，不是「读策略失败」
//
// 真机实测：写语句命中默认的 DML 审批策略时，卡片显示「无法读取审批策略」，
// 把一条明确可执行的下一步（去提工单）说成了系统故障。
func TestPolicyCheckerClassifiesBlockConclusions(t *testing.T) {
	ctx := context.Background()
	checks := policyChecker{}
	blockErr := errors.New("need work ticket")

	approval := checks.Check(ctx, &checkCtx{gate: &fakeGate{err: blockErr, kind: policyBlockApproval}, conn: &dbi.DbConn{}, sql: "update t set a=1"})
	if len(approval) != 1 || approval[0].Severity != SeverityNote || approval[0].Code != CodePolicyTicket {
		t.Fatalf("approval conclusion must be a note, got %+v", approval)
	}

	warnAck := checks.Check(ctx, &checkCtx{gate: &fakeGate{err: blockErr, kind: policyBlockWarnAck}, conn: &dbi.DbConn{}, sql: "update t set a=1"})
	if len(warnAck) != 1 || warnAck[0].Severity != SeverityNote {
		t.Fatalf("warn-ack conclusion must be a note, got %+v", warnAck)
	}

	unknown := checks.Check(ctx, &checkCtx{gate: &fakeGate{err: blockErr, kind: ""}, conn: &dbi.DbConn{}, sql: "update t set a=1"})
	if len(unknown) != 1 || unknown[0].Severity != SeverityUnverified || unknown[0].Code != CodeUncheckable {
		t.Fatalf("unknown failure must stay unverified, got %+v", unknown)
	}

	// 放行：无错误且无提醒 → 不产生任何结论
	pass := checks.Check(ctx, &checkCtx{gate: &fakeGate{}, conn: &dbi.DbConn{}, sql: "select 1"})
	if len(pass) != 0 {
		t.Fatalf("passing gate must not produce findings, got %+v", pass)
	}
}

// ---------- 列取值采样 ----------

// 只采枚举语义列：敏感列、描述列、非字符类型一律不采
func TestIsEnumLikeColumn(t *testing.T) {
	cases := []struct {
		name   string
		column columnInfo
		want   bool
	}{
		{"状态列", columnInfo{Name: "status", Type: "varchar(20)"}, true},
		{"指标名列", columnInfo{Name: "metric", Type: "varchar(48)"}, true},
		{"类型列", columnInfo{Name: "db_type", Type: "varchar(32)"}, true},
		{"枚举列", columnInfo{Name: "level", Type: "enum('a','b')"}, true},
		{"数值状态列", columnInfo{Name: "status", Type: "tinyint"}, false},
		{"描述列", columnInfo{Name: "status_desc", Type: "varchar(255)"}, false},
		{"敏感列", columnInfo{Name: "api_key", Type: "varchar(64)"}, false},
		{"手机号", columnInfo{Name: "phone", Type: "varchar(20)"}, false},
		{"普通名字", columnInfo{Name: "username", Type: "varchar(64)"}, false},
	}
	for _, one := range cases {
		if got := isEnumLikeColumn(one.column); got != one.want {
			t.Fatalf("%s (%s %s): got %v want %v", one.name, one.column.Name, one.column.Type, got, one.want)
		}
	}
}

// 采样值清洗：去空、去重、排序、截断，长文本不进入上下文
func TestTrimAndSortValues(t *testing.T) {
	got := trimAndSortValues([]string{" ping_fail ", "ping_fail", "", "lock_wait_count", strings.Repeat("x", 80)}, 5)
	want := []string{"lock_wait_count", "ping_fail"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
	if limited := trimAndSortValues([]string{"c", "a", "b"}, 2); len(limited) != 2 || limited[0] != "a" || limited[1] != "b" {
		t.Fatalf("limit broken: %v", limited)
	}
}

// 字面量抽取：只认「列 = 'x'」与「列 IN (...)」，且能处理限定名与转义引号
func TestLiteralsForColumn(t *testing.T) {
	sql := "select * from t_db_metric_sample m where m.metric = 'query_time' and m.status in ('ok','o''brien') and other = 'x'"
	got := literalsForColumn(sql, "metric")
	if len(got) != 1 || got[0] != "query_time" {
		t.Fatalf("metric literals: %v", got)
	}
	statusLiterals := literalsForColumn(sql, "status")
	if len(statusLiterals) != 2 || statusLiterals[1] != "o'brien" {
		t.Fatalf("status literals: %v", statusLiterals)
	}
	// 列名做整词匹配：metric_name 的取值不能被算作 metric 的
	if extra := literalsForColumn("where metric_name = 'x'", "metric"); len(extra) != 0 {
		t.Fatalf("word boundary broken: %v", extra)
	}
}

// 取值核对：采样里没有的取值必须被标为待确认，并给出真实候选
func TestValueCheckerFlagsUnsAmpledValue(t *testing.T) {
	ctx := context.Background()
	checker := valueChecker{}
	sampled := []sampledValue{{
		Table:  "t_db_metric_sample",
		Column: "metric",
		Values: []string{"lock_wait_count", "ping_fail", "sessions_active"},
	}}

	bad := checker.Check(ctx, &checkCtx{
		sql:     "select avg(value) from t_db_metric_sample where metric = 'query_time'",
		sampled: sampled,
	})
	if len(bad) != 1 || bad[0].Code != CodeValueNotSampled || bad[0].Severity != SeverityUnverified {
		t.Fatalf("expect value_not_sampled, got %+v", bad)
	}
	if len(bad[0].Candidates) == 0 {
		t.Fatal("candidates must carry real sampled values")
	}

	good := checker.Check(ctx, &checkCtx{
		sql:     "select avg(value) from t_db_metric_sample where metric = 'ping_fail'",
		sampled: sampled,
	})
	if len(good) != 0 {
		t.Fatalf("known value must pass, got %+v", good)
	}

	none := checker.Check(ctx, &checkCtx{sql: "select 1"})
	if len(none) != 0 {
		t.Fatalf("no samples must not produce findings, got %+v", none)
	}
}

// 解释意图：产物是散文，没有语句可校验，也不该出现「校验结论」
func TestExplainIntentReturnsProse(t *testing.T) {
	// 模型直接吐散文（不带 JSON 外壳）也必须能解析
	run := newTestRunWithChooser(t, []string{
		`这段 SQL 先按实例聚合未删除的数据库资源数量，再按数量倒序取前 3 个实例。风险：实例表小于 1 万行时全表聚合可接受，否则考虑加 (is_deleted) 索引。`,
	}, nil, []string{"t_order"})
	spec, ok := specOf("explain")
	if !ok {
		t.Fatal("explain intent not registered")
	}
	run.spec = spec
	run.req.Intent = "explain"
	run.req.Prompt = "解释这段 SQL"
	run.req.AnchorStatement = "select count(*) from t_order"

	draft, err := run.execute(context.Background())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if draft.Explanation == "" {
		t.Fatal("explanation must be present")
	}
	if len(draft.Statements) != 0 {
		t.Fatalf("explain must not produce statements, got %+v", draft.Statements)
	}
	if draft.Verification != "" {
		t.Fatalf("prose has nothing to verify, got verification=%q", draft.Verification)
	}
	if draft.Usage == nil {
		t.Fatal("usage must be reported")
	}
}

// 选表候选的预筛顺序：描述里出现过的表必须排到前面，避免被 80 张上限截掉
func TestRankTablesForPrompt(t *testing.T) {
	tables := []*tableInfo{
		{Name: "t_sys_log"},
		{Name: "t_db_metric_sample", Comment: "指标采样"},
		{Name: "t_db_instance", Comment: "数据库实例"},
	}
	ranked := rankTablesForPrompt(tables, "统计每个数据库实例的指标采样")
	if len(ranked) != 3 {
		t.Fatalf("ranking must keep all tables, got %d", len(ranked))
	}
	// t_db_instance 与 t_db_metric_sample 都在描述里命中（注释/名称），必须排到 t_sys_log 前面
	if ranked[2].Name != "t_sys_log" {
		t.Fatalf("irrelevant table must sink to the end, got %v", []string{ranked[0].Name, ranked[1].Name, ranked[2].Name})
	}
	// 空描述保持原顺序
	same := rankTablesForPrompt(tables, "   ")
	if same[0].Name != "t_sys_log" {
		t.Fatalf("empty prompt must keep original order, got %s", same[0].Name)
	}
}

// 写语句目标表核对：写向「需求没命中的同义表」必须被标出来
func TestTargetTableChecker(t *testing.T) {
	ctx := context.Background()
	checker := targetTableChecker{}
	promptTables := map[string]struct{}{"t_db_instance": {}}

	offTopic, _ := classify(dbi.GetDialect(dbi.ToDbType("mysql")), "UPDATE t_ai_plugin_instance SET description = name")
	bad := checker.Check(ctx, &checkCtx{
		stmt:         offTopic,
		stmtTyp:      "update",
		promptTables: promptTables,
	})
	if len(bad) != 1 || bad[0].Code != CodeTargetUnconfirmed || bad[0].Severity != SeverityUnverified {
		t.Fatalf("expect target_table_unconfirmed, got %+v", bad)
	}
	if len(bad[0].Candidates) == 0 || bad[0].Candidates[0] != "t_db_instance" {
		t.Fatalf("candidates must carry prompt-matched tables, got %+v", bad[0].Candidates)
	}

	onTarget, _ := classify(dbi.GetDialect(dbi.ToDbType("mysql")), "UPDATE t_db_instance SET remark = name")
	if got := checker.Check(ctx, &checkCtx{stmt: onTarget, stmtTyp: "update", promptTables: promptTables}); len(got) != 0 {
		t.Fatalf("matched target must pass, got %+v", got)
	}

	// 只读语句与「描述没命中任何表」都不判（后者没有可比对的基准）
	readStmt, readTyp := classify(dbi.GetDialect(dbi.ToDbType("mysql")), "SELECT id FROM t_ai_plugin_instance")
	if got := checker.Check(ctx, &checkCtx{stmt: readStmt, stmtTyp: readTyp, promptTables: promptTables}); len(got) != 0 {
		t.Fatalf("read statement must not be flagged, got %+v", got)
	}
	if got := checker.Check(ctx, &checkCtx{stmt: offTopic, stmtTyp: "update"}); len(got) != 0 {
		t.Fatalf("no prompt tables means no baseline, got %+v", got)
	}
}

// 散文产物的 JSON 必须是空数组而不是 null
//
// 真机踩到过：explain 返回 statements=null/checks=null，前端对数组做 map/length 抛异常，
// 整个结果卡片渲染中断，界面永久停在 loading（后端日志显示请求已成功，极易误判成"没返回"）。
func TestExplainDraftJSONHasNoNullArrays(t *testing.T) {
	run := newTestRunWithChooser(t, []string{"这段 SQL 统计未删除实例的数量。"}, nil, nil)
	spec, _ := specOf("explain")
	run.spec = spec
	run.req.Intent = "explain"
	run.req.Prompt = "解释这条语句"

	draft, err := run.execute(context.Background())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	raw, err := json.Marshal(draft)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), `"statements":null`) || strings.Contains(string(raw), `"checks":null`) {
		t.Fatalf("null arrays break the frontend renderer: %s", string(raw))
	}
	if !strings.Contains(string(raw), `"statements":[]`) || !strings.Contains(string(raw), `"checks":[]`) {
		t.Fatalf("expect empty arrays, got %s", string(raw))
	}
}

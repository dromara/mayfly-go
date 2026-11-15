package sqldraft

// 架构回归锁：每一条都对应一个**曾经真实存在**的缺陷，断言打在「缺陷会重新出现的那一层」上。
//
// 与 draft_test.go 的分工：那边验证管线行为正确，这边验证「结构约定没被悄悄回退」——
// 例如策略知识回流到编排层、召回阶段排错、依据文案绕过 i18n。这类回退往往不改变
// 单次行为的可观测结果（离线夹具太小，看不出少召回一张表），只有钉住结构才拦得住。
//
// 每条用例都做过「把缺陷退回原位、用例是否变红」的变异自证，避免写成永不触发的假锁。

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"mayfly-go/internal/ai/pkg/textgen"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/i18n"
)

// ---------- 替身 ----------

// fakeSavedSql 同库已存脚本仓储替身（记录被取过的次数，用于证明"片段自己取了材料"）
type fakeSavedSql struct {
	items []string
	calls int
}

func (f *fakeSavedSql) Examples(_ context.Context, _ uint64, limit int) ([]string, error) {
	f.calls++
	if limit > 0 && len(f.items) > limit {
		return f.items[:limit], nil
	}
	return f.items, nil
}

// ---------- 召回阶段 ----------

// 阶段划分必须按策略自声明的 Stage 落地，不按意图里的书写顺序。
//
// 有牙之处：扩展类策略的依据是「已选表集合」，一旦被排进提议阶段，它拿到的永远是空集合，
// 于是静默失效——离线夹具与真机都不会报错，只是少召回几张表。
func TestPartitionLinkersSplitsByDeclaredStage(t *testing.T) {
	// 故意把扩展类写在前面：顺序不该决定阶段
	proposers, expanders := partitionLinkers([]string{"sibling", "identifier", "relation", "comment", "unknown_id"})
	if len(proposers) != 2 || proposers[0].Id() != "identifier" || proposers[1].Id() != "comment" {
		t.Fatalf("propose stage must hold identifier+comment in declaration order, got %v", idsOf(proposers))
	}
	if len(expanders) != 2 || expanders[0].Id() != "sibling" || expanders[1].Id() != "relation" {
		t.Fatalf("expand stage must hold sibling+relation in declaration order, got %v", idsOf(expanders))
	}
}

func idsOf(list []Linker) []string {
	out := make([]string, 0, len(list))
	for _, one := range list {
		out = append(out, one.Id())
	}
	return out
}

// 外键邻接召回必须真的经过管线生效。
//
// 这是本包最隐蔽的一个缺陷：relationLinker 曾因为「picked 取自一个只在循环结束后才写入的 map」
// 而恒为空，加上外键数据在 resolveTables **之后**才装载，整条通道从上线到被发现从未产出过一张表。
// 断言打在管线产物（resolveTables 的返回值）上而不是策略单测上：策略单测全绿，
// 也可能因为编排层没喂数据而整条失效。
func TestResolveTablesExpandsForeignKeysAfterChooser(t *testing.T) {
	meta := &fkCountingMeta{fakeMeta: fixtureMeta()}
	run := newTestRunWithChooser(t, []string{`{"statements":[]}`}, nil, []string{"t_order"})
	run.meta = meta
	// 描述里既没有精确表名，也没有能过门槛（≥3 字）的注释重合 → 必然走选择题
	run.req.Prompt = "按用户统计订单金额"

	picked, reasons, _, err := run.resolveTables(context.Background(), run.req.Prompt)
	if err != nil {
		t.Fatalf("resolve tables: %v", err)
	}
	joined := strings.Join(picked, ",")
	if !strings.Contains(joined, "t_order") {
		t.Fatalf("chooser result must be kept, got %v", picked)
	}
	if !strings.Contains(joined, "t_user") {
		t.Fatalf("foreign-key neighbour must be expanded in, got %v", picked)
	}
	if meta.fkCalls == 0 {
		t.Fatal("relation linker must load foreign keys itself (snapshot.ensureRelations)")
	}
	relationReason := strings.Join(reasons["t_user"], ",")
	if !strings.Contains(relationReason, i18n.TC(context.Background(), imsg.SqlDraftRelationReason)) {
		t.Fatalf("expanded table must carry the relation reason, got %q", relationReason)
	}
}

// fkCountingMeta 记录外键查询次数的替身：证明「召回阶段真的去读了外键」
type fkCountingMeta struct {
	*fakeMeta
	fkCalls int
}

func (f *fkCountingMeta) GetForeignKeys(ctx context.Context, schema, table string) ([]dbi.ForeignKey, error) {
	f.fkCalls++
	return f.fakeMeta.GetForeignKeys(ctx, schema, table)
}

// 同后缀同族表补齐：选中 t_ai_plugin_instance 时要把同后缀的 t_db_instance 也带进上下文。
//
// 断言打在「策略已注册 + 意图确实声明了它 + 阶段是扩展」上，而不是打在编排层的某个方法上：
// 这个补齐曾经写成 draftRun 的方法并被管线无条件调用，看着能用，实际把策略知识焊死在管线里
// （新增一种召回就得改管线）。只有钉住注册表，才能保证「新增一种召回 = 注册一行」不被悄悄回退。
func TestSiblingLinkerIsRegisteredAndProposesFamilyTables(t *testing.T) {
	linker, ok := linkerOf("sibling")
	if !ok {
		t.Fatal("sibling linker must be registered")
	}
	if linker.Stage() != LinkStageExpand {
		t.Fatalf("sibling must run in the expand stage, got %v", linker.Stage())
	}
	for _, intentId := range []string{"draft", "optimize", "fix"} {
		spec, found := specOf(intentId)
		if !found {
			t.Fatalf("intent %s must be registered", intentId)
		}
		declared := false
		for _, id := range spec.Linkers {
			if id == "sibling" {
				declared = true
			}
		}
		if !declared {
			t.Fatalf("intent %s must declare the sibling linker, got %v", intentId, spec.Linkers)
		}
	}

	snap := newBareSnapshot("t_ai_plugin_instance", "t_db_instance", "t_sys_log")
	items, err := siblingLinker{}.Propose(context.Background(), &linkCtx{snap: snap, limit: maxTables, picked: []string{"t_ai_plugin_instance"}})
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	joined := strings.Join(candidateNames(items), ",")
	if !strings.Contains(joined, "t_db_instance") {
		t.Fatalf("same-suffix sibling must be included, got %v", joined)
	}
	if strings.Contains(joined, "t_sys_log") {
		t.Fatalf("unrelated table must not be added, got %v", joined)
	}
	// 没有已选表时不产出：扩展阶段的依据就是「已选集合」，集合为空时补齐没有意义
	empty, emptyErr := siblingLinker{}.Propose(context.Background(), &linkCtx{snap: snap, limit: maxTables})
	if emptyErr != nil || len(empty) != 0 {
		t.Fatalf("no picked tables means no siblings, got %v err=%v", empty, emptyErr)
	}

	// 尾段过短（<4）或缺下划线的表名不参与补齐：_log/_id/_v2 这类尾段会把无关表都拖进来
	if suffix := tableSuffix("t_sys_user"); suffix != "user" {
		t.Fatalf("suffix parse broken: %q", suffix)
	}
	if suffix := tableSuffix("t_sys_log"); suffix != "" {
		t.Fatalf("3-char suffix must be ignored, got %q", suffix)
	}
	if suffix := tableSuffix("t_v2"); suffix != "" {
		t.Fatalf("short suffix must be ignored, got %q", suffix)
	}
}

// 表名命中来源要如实标注：描述里命中、与编辑器选区/光标语句里命中，理由必须不同。
//
// 真机坐实过反例：描述「统计每个数据库实例下有多少个数据库资源」里没有任何表名，
// 命中来自光标所在语句（材料），界面理由却写「描述中出现了该表名」——解释性表面
// 说假话比不说更糟，用户会据此误判召回依据。把缺陷退回「理由写死一种文案」即变红。
func TestIdentifierLinkerLabelsHitSource(t *testing.T) {
	ctx := context.Background()
	snap := newBareSnapshot("t_db_instance", "t_sys_log")
	descReason := i18n.TC(ctx, imsg.SqlDraftIdentifierReason)
	materialReason := i18n.TC(ctx, imsg.SqlDraftMaterialIdentifierReason)
	if descReason == materialReason {
		t.Fatal("the two hit-source reasons must differ")
	}

	// 描述无表名、材料有表名 → 材料理由
	items, err := identifierLinker{}.Propose(ctx, &linkCtx{
		text:   "统计每个实例下有多少资源 SELECT * FROM t_db_instance",
		prompt: "统计每个实例下有多少资源",
		snap:   snap,
		limit:  maxTables,
	})
	if err != nil || len(items) != 1 {
		t.Fatalf("expect exactly one candidate, got %d err=%v", len(items), err)
	}
	if items[0].Table != "t_db_instance" || items[0].Reasons[0] != materialReason {
		t.Fatalf("material hit must be labelled as material, got %v %v", items[0].Table, items[0].Reasons)
	}

	// 描述里出现表名 → 描述理由（材料同时包含也不改变来源判定）
	items, err = identifierLinker{}.Propose(ctx, &linkCtx{
		text:   "查 t_sys_log 最近一天的记录 SELECT * FROM t_sys_log",
		prompt: "查 t_sys_log 最近一天的记录",
		snap:   snap,
		limit:  maxTables,
	})
	if err != nil || len(items) != 1 {
		t.Fatalf("expect exactly one candidate, got %d err=%v", len(items), err)
	}
	if items[0].Reasons[0] != descReason {
		t.Fatalf("prompt hit must be labelled as description, got %v", items[0].Reasons)
	}
}

// ---------- 选择题 ----------

// 选表选择题必须给「注释为空」的候选补列名预览。
//
// 真机踩到过：本项目自有库的表注释普遍为空，t_db_instance 与 t_ai_plugin_instance
// 光看名字无从区分，模型只能瞎挑一张——需求说"实例"却挑了插件实例表。
// 同时钉住"只给注释缺失的卡片补"：注释齐全时预览是纯开销（每张卡片一次元数据往返）。
func TestChooseTablesPreviewsColumnsWhenCommentMissing(t *testing.T) {
	port := &recordingPort{fakePort: fakePort{chooseTables: []string{"t_db_instance"}}}
	snap := newBareSnapshot("t_db_instance", "t_ai_plugin_instance")
	meta := &fakeMeta{columns: map[string][]dbi.Column{
		"t_db_instance": {{TableName: "t_db_instance", ColumnName: "db_name", ColumnType: "varchar(64)"}},
	}}

	if _, _, err := chooseTables(context.Background(), newGenerator(port), snap, meta, "一共有多少个库", maxTables); err != nil {
		t.Fatalf("choose tables: %v", err)
	}
	withColumns := cardLine(t, port.lastChooseUser, "t_db_instance")
	if !strings.Contains(withColumns, "cols: db_name") {
		t.Fatalf("comment-less card must carry a column preview, got %q", withColumns)
	}
	withoutColumns := cardLine(t, port.lastChooseUser, "t_ai_plugin_instance")
	if strings.Contains(withoutColumns, "cols:") {
		t.Fatalf("a card whose columns could not be read must not fake a preview, got %q", withoutColumns)
	}
}

// 注释齐全的卡片不该被拉去取列：这是「只在缺失时补」的另一半断言
func TestChooseTablesSkipsPreviewWhenCommentPresent(t *testing.T) {
	port := &recordingPort{fakePort: fakePort{chooseTables: []string{"t_user"}}}
	snap := newBareSnapshot("t_user")
	snap.tables[0].Comment = "用户信息表"
	meta := &fakeMeta{columns: map[string][]dbi.Column{
		"t_user": {{TableName: "t_user", ColumnName: "name", ColumnType: "varchar(64)"}},
	}}

	if _, _, err := chooseTables(context.Background(), newGenerator(port), snap, meta, "统计用户", maxTables); err != nil {
		t.Fatalf("choose tables: %v", err)
	}
	card := cardLine(t, port.lastChooseUser, "t_user")
	if strings.Contains(card, "cols:") {
		t.Fatalf("a card with a comment needs no preview, got %q", card)
	}
	if !strings.Contains(card, "用户信息表") {
		t.Fatalf("comment must still be rendered, got %q", card)
	}
}

// cardLine 取候选清单里某张表所在的那一行。
//
// 按行断言而不是全文 Contains：模板正文里也写着 "cols:" 这个词（用于向模型解释格式），
// 全文匹配会把模板自己的说明当成卡片内容，用例就在空跑
func cardLine(t *testing.T, cards, table string) string {
	t.Helper()
	for _, line := range strings.Split(cards, "\n") {
		if strings.Contains(line, table) {
			return line
		}
	}
	t.Fatalf("table %s not offered to the chooser:\n%s", table, cards)
	return ""
}

// recordingPort 记下选择题那次调用的 user 正文（用于断言卡片内容）
type recordingPort struct {
	fakePort
	lastChooseUser string
}

func (p *recordingPort) Complete(ctx context.Context, req *textgen.Request) (string, *textgen.Usage, error) {
	if req.Purpose == purposeChoose {
		p.lastChooseUser = req.User
	}
	return p.fakePort.Complete(ctx, req)
}

// ---------- 材料长度上界 ----------

// 材料长度必须有上界，且超限是**拒绝**而不是截断。
//
// contextTokenBudget 只管装配后的上下文片段，管不到任务正文：用户把整个脚本当选区提交时，
// 正文会原样进提示词，既拖慢也挤爆输出预算。截断更糟——它会生成一份看起来正常、
// 实际答非所问的草稿，用户无从察觉。
func TestDraftRejectsOversizedMaterial(t *testing.T) {
	app := &DraftAppImpl{}
	req := &Req{DbId: 1, Prompt: strings.Repeat("统计", maxMaterialRunes)}

	_, err := app.Draft(context.Background(), req)
	if err == nil {
		t.Fatal("oversized material must be rejected")
	}
	if !strings.Contains(err.Error(), fmt.Sprint(maxMaterialRunes)) {
		t.Fatalf("the error must tell the user the limit, got %v", err)
	}
}

// 恰好在上界内不该被这条闸门拦下（否则上界就变成了"永远拒绝"）
func TestDraftAcceptsMaterialAtLimit(t *testing.T) {
	app := &DraftAppImpl{}
	req := &Req{DbId: 1, Prompt: strings.Repeat("a", maxMaterialRunes)}

	_, err := app.Draft(context.Background(), req)
	if err == nil {
		t.Fatal("expect a downstream error (no model configured), not success")
	}
	if strings.Contains(err.Error(), fmt.Sprint(maxMaterialRunes)) {
		t.Fatalf("material at the limit must not be rejected as oversized, got %v", err)
	}
}

// ---------- i18n ----------

// 召回依据必须走 i18n：它经 ContextTable.Reason 直达界面，
// 硬编码中文会让英文用户看到中英文混排的依据列。
//
// 用「描述里点出表名」触发 identifier 通道，并让外键把 t_order 扩展进来，
// 于是一次断言同时覆盖 identifier / relation 两类依据的本地化。
func TestContextTableReasonsAreLocalized(t *testing.T) {
	run := newTestRun(t, []string{`{"statements":[]}`}, nil)
	run.req.Prompt = "查询 t_user 表"
	ctx := i18n.NewCtxWithLang(context.Background(), i18n.En)

	picked, reasons, _, err := run.resolveTables(ctx, run.req.Prompt)
	if err != nil {
		t.Fatalf("resolve tables: %v", err)
	}
	if len(picked) < 2 {
		t.Fatalf("expect t_user plus its foreign-key neighbour, got %v", picked)
	}
	for _, name := range picked {
		joined := strings.Join(reasons[strings.ToLower(name)], ",")
		if joined == "" {
			t.Fatalf("picked table %s must carry a reason", name)
		}
		if hasCJK(joined) {
			t.Fatalf("english locale must not render chinese reasons for %s, got %q", name, joined)
		}
	}
}

// 选择题「候选被截断」的告警同样直达界面，也不得硬编码中文
func TestChooserTruncationWarningIsLocalized(t *testing.T) {
	got := i18n.TL(i18n.En, imsg.ErrSqlDraftChooserTruncated, "total", 300, "max", maxTableCards)
	if hasCJK(got) {
		t.Fatalf("english warning must not contain chinese, got %q", got)
	}
	if !strings.Contains(got, "300") {
		t.Fatalf("warning must carry the real table count, got %q", got)
	}
}

func hasCJK(text string) bool {
	for _, r := range text {
		if r >= 0x4e00 && r <= 0x9fff {
			return true
		}
	}
	return false
}

// ---------- 片段自取材料 ----------

// 示例片段必须自己取材料：管线里曾经写着 `if id == "examples"`，
// 新增一种带材料的片段就得改管线。断言打在「不经过管线、单独渲染也能出内容」上——
// 只有这样才证明取数能力长在片段自己身上，而不是靠编排层喂。
func TestExamplesFragmentLoadsItsOwnMaterial(t *testing.T) {
	snap := newBareSnapshot("t_order")
	saved := &fakeSavedSql{items: []string{"SELECT amount FROM t_order GROUP BY user_id"}}
	fc := &fragCtx{snap: snap, picked: snap.tables, req: &Req{DbId: 1}, savedSql: saved}

	body := examplesFragment{}.Render(context.Background(), fc)
	if saved.calls == 0 {
		t.Fatal("examples fragment must read the repository itself")
	}
	if !strings.Contains(body, "t_order") {
		t.Fatalf("examples fragment must render its own material, got %q", body)
	}
}

// 示例按「与本次选中表的重合度」挑，而不是取最新两条。
//
// 最新的脚本很可能属于上一件事（用户刚写完另一张表的语句），与本次需求毫无关系；
// 而表重合的脚本连表名带写法都对得上，是有效得多的 few-shot。
func TestLoadExamplesPrefersTableOverlap(t *testing.T) {
	snap := newBareSnapshot("t_order")
	saved := &fakeSavedSql{items: []string{
		// 仓储顺序在前（通常是"更新"），但与本次选中的表无关
		"SELECT id FROM t_user WHERE name = 'someone'",
		"SELECT amount FROM t_order GROUP BY user_id",
	}}

	got := loadExamples(context.Background(), saved, 1, snap.tables)
	if len(got) != exampleLimit {
		t.Fatalf("expect %d examples, got %v", exampleLimit, got)
	}
	if !strings.Contains(got[0], "t_order") {
		t.Fatalf("the overlapping example must rank first, got %q", got[0])
	}
	if strings.Contains(got[0], "someone") {
		t.Fatalf("string literals must be stripped from examples, got %q", got[0])
	}
}

// ---------- 夹具 ----------

// newBareSnapshot 只有表清单、没有任何结构细节的快照（懒装载类用例的夹具）
func newBareSnapshot(names ...string) *snapshot {
	snap := &snapshot{
		tables:        make([]*tableInfo, 0, len(names)),
		byName:        make(map[string]*tableInfo, len(names)),
		visible:       make(map[string]struct{}, len(names)),
		hidden:        map[string]struct{}{},
		deniedColumns: map[string]struct{}{},
		colLoaded:     map[string]struct{}{},
		relLoaded:     map[string]struct{}{},
		fkSeen:        map[string]struct{}{},
	}
	for _, name := range names {
		info := &tableInfo{Name: name}
		key := strings.ToLower(name)
		snap.tables = append(snap.tables, info)
		snap.byName[key] = info
		snap.visible[key] = struct{}{}
	}
	return snap
}

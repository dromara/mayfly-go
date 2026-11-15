package sqldraft

// 语义层（业务词典）的回归锁。
//
// 覆盖三件事：词典必须能改变召回结果（否则登记等于没登记）、必须能进上下文、
// 以及写侧的 upsert/上限/归属判定不能被绕过。每条都做过「把实现退回原位、用例是否变红」的自证。

import (
	"context"
	"strings"
	"testing"

	"mayfly-go/internal/ai/pkg/textgen"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
)

// ---------- 替身 ----------

// fakeTermRepo 词典仓储替身：只实现被测路径用到的方法，其余走内嵌接口（未实现即 panic，
// 避免「悄悄调了个空实现」这种假通过）
type fakeTermRepo struct {
	repository.DbAiTerm

	items   []*entity.DbAiTerm
	nextId  uint64
	updates [][]string // 每次 UpdateById 传入的列，用于断言审计列没被漏掉
}

func (f *fakeTermRepo) Insert(_ context.Context, e *entity.DbAiTerm) error {
	f.nextId++
	e.Id = f.nextId
	f.items = append(f.items, e)
	return nil
}

func (f *fakeTermRepo) ListByDb(dbId uint64) ([]*entity.DbAiTerm, error) {
	out := make([]*entity.DbAiTerm, 0, len(f.items))
	for _, item := range f.items {
		if item.DbId == dbId {
			out = append(out, item)
		}
	}
	return out, nil
}

func (f *fakeTermRepo) GetById(id uint64, _ ...string) (*entity.DbAiTerm, error) {
	for _, item := range f.items {
		if item.Id == id {
			return item, nil
		}
	}
	return nil, nil
}

func (f *fakeTermRepo) UpdateById(_ context.Context, e *entity.DbAiTerm, columns ...string) error {
	f.updates = append(f.updates, columns)
	for _, item := range f.items {
		if item.Id == e.Id {
			*item = *e
		}
	}
	return nil
}

func (f *fakeTermRepo) DeleteById(_ context.Context, ids ...uint64) error {
	kept := f.items[:0]
	for _, item := range f.items {
		drop := false
		for _, id := range ids {
			drop = drop || item.Id == id
		}
		if !drop {
			kept = append(kept, item)
		}
	}
	f.items = kept
	return nil
}

// ---------- 召回 ----------

// 词典必须真的改变召回结果，并且权重排列正确：高于外键邻接（它带来的是人工事实，
// 不是猜出来的关系），低于「描述里直接写了表名」（用户点名才是完整信号）。
//
// 这是语义层存在的理由：表注释为空的库里，「实例」这类口径在 schema 里不存在，
// 词面通道拿不到、模型只能猜，真机实测就会挑错同义表。
func TestGlossaryLinkerIsRegisteredAndRankedBetweenFkAndExplicitName(t *testing.T) {
	linker, ok := linkerOf("glossary")
	if !ok {
		t.Fatal("glossary linker must be registered")
	}
	if linker.Stage() != LinkStagePropose {
		t.Fatalf("glossary runs on the user material, got %v", linker.Stage())
	}
	for _, intentId := range []string{"draft", "optimize", "fix", "suggest"} {
		spec, found := specOf(intentId)
		if !found {
			t.Fatalf("intent %s must be registered", intentId)
		}
		declared := false
		for _, id := range spec.Linkers {
			declared = declared || id == "glossary"
		}
		if !declared {
			t.Fatalf("intent %s must declare the glossary linker, got %v", intentId, spec.Linkers)
		}
	}

	ctx := context.Background()
	snap := newBareSnapshot("t_db_instance", "t_ai_plugin_instance")
	items := []*entity.DbAiTerm{
		{Term: "实例", Tables: []string{"t_db_instance", "t_ai_plugin_instance", "t_hidden_table"}},
		{Term: "库", Tables: []string{"t_db_instance"}},
		{Term: "订单", Tables: []string{"t_order"}},
	}
	out, err := glossaryLinker{}.Propose(ctx, &linkCtx{text: "统计每个实例的资源数", terms: items, snap: snap, limit: maxTables})
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expect the two registered visible tables, got %v", candidateNames(out))
	}
	for _, item := range out {
		// 依据要指到具体术语，否则用户看不出词典起了什么作用
		if len(item.Reasons) != 1 || !strings.Contains(item.Reasons[0], "实例") {
			t.Fatalf("reason must name the matched term, got %v", item.Reasons)
		}
		// 高于外键邻接（6）：人工登记的口径比「猜出来的关系」更该被保留
		if item.Score <= 6 {
			t.Fatalf("glossary must outrank the foreign-key channel, got %f", item.Score)
		}
	}

	// 描述没提到任何术语时不产出：整本词典的表都进上下文等于没有召回
	empty, err := glossaryLinker{}.Propose(ctx, &linkCtx{text: "随便查点东西", terms: items, snap: snap, limit: maxTables})
	if err != nil || len(empty) != 0 {
		t.Fatalf("no term matched means no candidate, got %v err=%v", candidateNames(empty), err)
	}
}

// ---------- 与选择题的关系 ----------

// countingPort 按阶段统计模型调用次数（判断「该不该再问一次模型」必须看阶段，不能只看总数）
type countingPort struct {
	fakePort
	choose int
	draft  int
}

func (p *countingPort) Complete(ctx context.Context, req *textgen.Request) (string, *textgen.Usage, error) {
	switch req.Purpose {
	case purposeChoose:
		p.choose++
	case purposeDraft:
		p.draft++
	}
	return p.fakePort.Complete(ctx, req)
}

// 词典通道的真实不变量（两条一起才成立）：
//  1. 「仅凭描述」那一趟必须看得到词典——漏传时登记了口径仍去问模型，登记等于没生效；
//  2. 但命中术语不足以宣称「描述只需要这些表」——据此跳过选择题会让描述里的其它实体
//     根本进不了上下文（真机回退：指标表缺失，模型输出一条把错误文本当数据的放弃式语句）。
//
// 只断言「结果看起来对」抓不到第 2 条，必须断言模型调用次数与分数与门槛的大小关系。
func TestGlossaryHitFeedsConfidencePassWithoutClaimingFullCoverage(t *testing.T) {
	terms, repo := newTermApp()
	repo.items = []*entity.DbAiTerm{{DbId: 1, Schema: "demo", Term: "用户订单", Tables: []string{"t_user"}}}
	repo.nextId = 1

	run := newTestRun(t, nil, nil)
	run.termSvc = terms
	// 描述里只有业务说法，没有任何表名（identifier 通道不命中），注释词面公共串也不足 3 字
	run.req.Prompt = "统计每个用户订单的数量"
	run.gen = newGenerator(&countingPort{
		fakePort: fakePort{replies: []string{`{"statements":[{"sql":"SELECT name FROM t_user"}]}`}, chooseTables: []string{"t_order"}},
	})

	draft, err := run.execute(context.Background())
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	port := run.gen.port.(*countingPort)
	if port.draft != 1 {
		t.Fatalf("expect exactly one generation call, got %d", port.draft)
	}
	// 术语命中不等于完整覆盖：选择题仍要跑，它负责描述里没登记成术语的其它实体
	if port.choose != 1 {
		t.Fatalf("a glossary term hit must not claim full coverage, choose calls=%d", port.choose)
	}
	// 登记的表必须进上下文（这是登记词典的目的本身）
	var matched *ContextTable
	for index := range draft.ContextTables {
		if draft.ContextTables[index].Name == "t_user" {
			matched = &draft.ContextTables[index]
		}
	}
	if matched == nil {
		t.Fatalf("the registered table must be recalled, got %v", draft.ContextTables)
	}
	if !strings.Contains(matched.Reason, "用户订单") {
		t.Fatalf("the reason must name the matched term, got %q", matched.Reason)
	}

	// 分数必须低于门槛：跨过它就等于宣称「描述只需要这几张表」
	out, err := glossaryLinker{}.Propose(context.Background(), &linkCtx{
		text:   "统计每个用户订单的数量",
		prompt: "统计每个用户订单的数量",
		snap:   newBareSnapshot("t_user"),
		terms:  []*entity.DbAiTerm{{Term: "用户订单", Tables: []string{"t_user"}}},
		limit:  maxTables,
	})
	if err != nil || len(out) != 1 {
		t.Fatalf("propose: %v err=%v", out, err)
	}
	if out[0].Score >= confidentScore {
		t.Fatalf("glossary score %f must stay below confidentScore %d", out[0].Score, confidentScore)
	}
}

// ---------- 上下文片段 ----------

// 词典要进上下文，且长口径截断而不是整条丢弃（静默丢弃会让用户以为登记没生效）。
func TestGlossaryFragmentRendersTermsWithBoundedDetail(t *testing.T) {
	ctx := context.Background()
	long := strings.Repeat("口径", 300)
	block := glossaryFragment{}.Render(ctx, &fragCtx{terms: []*entity.DbAiTerm{
		{Term: "实例", Tables: []string{"t_db_instance", "t_es_instance"}, Details: "只算未删除的"},
		{Term: "金额单位", Details: long},
		{Term: "  ", Tables: []string{"t_x"}},
		{Term: "空条目"},
	}})
	if !strings.Contains(block, "业务词典") {
		t.Fatal("glossary block must carry its own heading")
	}
	if !strings.Contains(block, "实例 → t_db_instance, t_es_instance：只算未删除的") {
		t.Fatalf("term with tables and detail rendered wrong: %s", block)
	}
	if !strings.Contains(block, "金额单位") || !strings.Contains(block, "已截断") {
		t.Fatalf("oversized detail must be truncated with a marker, not dropped: %s", block)
	}
	if strings.Contains(block, "t_x") || strings.Count(block, "\n") > 4 {
		t.Fatalf("blank term or content-free entry must be skipped: %q", block)
	}
	empty := glossaryFragment{}.Render(ctx, &fragCtx{})
	if empty != "" {
		t.Fatal("no glossary means no block")
	}
}

// ---------- 写侧 ----------

func newTermApp() (*TermsAppImpl, *fakeTermRepo) {
	repo := &fakeTermRepo{}
	return &TermsAppImpl{termRepo: repo}, repo
}

// 同库同名术语（忽略大小写与库名）必须覆盖而不是新增，且显式列更新要带上审计列。
//
// 审计列这条有牙：UpdateById 走 Select(columns).Updates()，只写列出的列——漏了 modifier
// 的表现是「改了词典但查不到谁改的」，编译与常规测试都不会报错。
func TestSaveTermUpsertsAndKeepsAuditColumns(t *testing.T) {
	ctx := context.Background()
	app, repo := newTermApp()

	if err := app.SaveTerm(ctx, &entity.DbAiTerm{DbId: 7, Schema: "demo", Term: " 实例 ", Tables: []string{" t_db_instance ", "", "t_es_instance"}}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if len(repo.items) != 1 || repo.items[0].Term != "实例" {
		t.Fatalf("term must be trimmed on save, got %+v", repo.items)
	}
	if got := strings.Join(repo.items[0].Tables, ","); got != "t_db_instance,t_es_instance" {
		t.Fatalf("blank and duplicate tables must be normalized, got %s", got)
	}

	// 大小写与首尾空白不同的同名条目走更新：插两行会让「哪条生效」取决于查询顺序
	if err := app.SaveTerm(ctx, &entity.DbAiTerm{DbId: 7, Schema: "DEMO", Term: "实例", Details: "只算未删除的"}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if len(repo.items) != 1 {
		t.Fatalf("same term in the same schema must update, got %d rows", len(repo.items))
	}
	if repo.items[0].Details == "" {
		t.Fatal("the update must persist the new detail")
	}
	joined := strings.Join(repo.updates[len(repo.updates)-1], ",")
	for _, column := range termAuditColumns {
		if !strings.Contains(joined, column) {
			t.Fatalf("explicit column update must include the audit column %q, got %s", column, joined)
		}
	}

	// 别的库同名是另一条口径，不能覆盖
	if err := app.SaveTerm(ctx, &entity.DbAiTerm{DbId: 9, Schema: "demo", Term: "实例"}); err != nil {
		t.Fatalf("save other db: %v", err)
	}
	if len(repo.items) != 2 {
		t.Fatalf("a different db keeps its own entry, got %d", len(repo.items))
	}

	// 术语为空必须失败：空术语会让「描述命中」判定退化成空前缀匹配
	if err := app.SaveTerm(ctx, &entity.DbAiTerm{DbId: 7, Term: "   "}); err == nil {
		t.Fatal("a blank term must be rejected")
	}
}

// 条目上限是拒绝而不是静默丢弃：静默丢弃会让用户以为登记生效了。
func TestSaveTermRejectsBeyondLimit(t *testing.T) {
	ctx := context.Background()
	app, repo := newTermApp()
	for index := 0; index < maxTermsPerDb; index++ {
		if err := app.SaveTerm(ctx, &entity.DbAiTerm{DbId: 7, Term: string(rune('a'+index%26)) + strings.Repeat("x", index/26+1)}); err != nil {
			t.Fatalf("fill %d: %v", index, err)
		}
	}
	if len(repo.items) != maxTermsPerDb {
		t.Fatalf("fixture wrong, got %d", len(repo.items))
	}
	if err := app.SaveTerm(ctx, &entity.DbAiTerm{DbId: 7, Term: "再来一条"}); err == nil {
		t.Fatal("exceeding the glossary limit must be rejected")
	}
	// 另一个库不受影响：上限是按库的，不是全局的
	if err := app.SaveTerm(ctx, &entity.DbAiTerm{DbId: 8, Term: "再来一条"}); err != nil {
		t.Fatalf("another db must still accept entries: %v", err)
	}
}

// 删除必须按条目自己的库判定归属，且他库 id 与「不存在」同口径（不泄漏存在性）。
func TestDeleteTermRejectsForeignId(t *testing.T) {
	ctx := context.Background()
	app, repo := newTermApp()
	if err := app.SaveTerm(ctx, &entity.DbAiTerm{DbId: 7, Term: "实例"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	id := repo.items[0].Id
	if err := app.DeleteTerm(ctx, 9, id); err == nil {
		t.Fatal("deleting another db's entry must fail")
	}
	if len(repo.items) != 1 {
		t.Fatal("a rejected delete must not touch the row")
	}
	if err := app.DeleteTerm(ctx, 7, id); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(repo.items) != 0 {
		t.Fatalf("entry must be gone, got %d", len(repo.items))
	}
	if err := app.DeleteTerm(ctx, 7, id); err == nil {
		t.Fatal("deleting twice must report not-found instead of a silent success")
	}
}

// ListForSchema 的口径：本库条目 + 全库共用条目（Schema 为空）。
func TestListForSchemaIncludesSharedEntries(t *testing.T) {
	ctx := context.Background()
	app, _ := newTermApp()
	entries := []*entity.DbAiTerm{
		{DbId: 7, Schema: "demo", Term: "本库"},
		{DbId: 7, Schema: "", Term: "共用"},
		{DbId: 7, Schema: "other", Term: "别库"},
	}
	for _, item := range entries {
		if err := app.SaveTerm(ctx, item); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	got, err := app.ListForSchema(ctx, 7, "DEMO")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	names := make([]string, 0, len(got))
	for _, item := range got {
		names = append(names, item.Term)
	}
	joined := strings.Join(names, ",")
	if !strings.Contains(joined, "本库") || !strings.Contains(joined, "共用") || strings.Contains(joined, "别库") {
		t.Fatalf("expect this-schema plus shared entries only, got %s", joined)
	}
}

package sqldraft

// 多轮细化与候选重排（best-of-N 的条件触发形态）的回归锁。

import (
	"context"
	"strings"
	"testing"

	"mayfly-go/internal/db/domain/entity"
)

// ---------- 多轮细化 ----------

// 上一版草稿必须计入材料：它既是召回依据（细化通常延续同一批表），也必须受长度闸门约束。
//
// 漏掉它的后果是「闸门只管描述、不管这条新入口」——用户可以拿 refineOf 把任意长文本送进模型，
// 而前端与后端都报不出错。
func TestRefineMaterialCountsTowardRecallAndGate(t *testing.T) {
	materials := materialsOf(&Req{Prompt: "统计用户", RefineOf: "SELECT 1 FROM t_user"})
	if !strings.Contains(materials, "SELECT 1 FROM t_user") {
		t.Fatalf("the previous draft must be part of the materials, got %q", materials)
	}

	app := &DraftAppImpl{}
	oversized := &Req{DbId: 1, Db: "demo", Intent: "draft", Prompt: "统计用户", RefineOf: strings.Repeat("x", maxMaterialRunes+1)}
	if _, err := app.Draft(context.Background(), oversized); err == nil {
		t.Fatal("an oversized previous draft must be rejected by the material gate")
	}
}

// 细化块只在有上一版时出现，且必须真的出现在交给模型的任务正文里。
//
// 断言打在渲染结果上而不是某个布尔位：模板里的 {{if .Refine}} 写错、或任务模板忘了放占位符，
// 都会让「细化」静默退化成「重新生成」——表现完全一致，只有渲染结果能区分。
func TestRefineBlockReachesTheTaskPrompt(t *testing.T) {
	run := newTestRun(t, nil, nil)
	if run.refineBlock(context.Background()) != "" {
		t.Fatal("no previous draft means no refine block")
	}
	run.req.RefineOf = "SELECT id FROM t_user"
	block := run.refineBlock(context.Background())
	if !strings.Contains(block, "上一版草稿") || !strings.Contains(block, "SELECT id FROM t_user") {
		t.Fatalf("refine block must quote the previous draft, got %q", block)
	}

	task, err := renderTemplate(run.spec.TaskTemplate, map[string]any{
		"Prompt": run.req.Prompt, "Refine": block, "Context": "ctx",
	})
	if err != nil {
		t.Fatalf("render task: %v", err)
	}
	if !strings.Contains(task, "上一版草稿") {
		t.Fatalf("the task template lost its refine slot, got %q", task)
	}
}

// ---------- 候选重排 ----------

// 排序键：先比结论等级，再比各级条数；note 不参与（它是提示不是问题）。
func TestCandidateRankOrdersByVerificationThenFindings(t *testing.T) {
	verified := []Finding{{Severity: SeverityNote, Code: "write_statement"}}
	unverified := []Finding{{Severity: SeverityUnverified, Code: "value_not_sampled"}}
	twoUnverified := []Finding{
		{Severity: SeverityUnverified, Code: "value_not_sampled"},
		{Severity: SeverityUnverified, Code: "target_table_unconfirmed"},
	}
	invalid := []Finding{{Severity: SeverityInvalid, Code: "unknown_column"}}

	if !isBetterCandidate(verified, unverified) {
		t.Fatal("verified must beat unverified")
	}
	if !isBetterCandidate(unverified, twoUnverified) {
		t.Fatal("fewer unverified findings must win")
	}
	if !isBetterCandidate(unverified, invalid) {
		t.Fatal("any unverified must beat invalid")
	}
	// 平手保留已有结果：换一份「同样好」的草稿只会让用户困惑
	if isBetterCandidate(verified, verified) || isBetterCandidate(twoUnverified, twoUnverified) {
		t.Fatal("an equal candidate must not replace the current result")
	}
	if candidateRank(verified) != candidateRank([]Finding{}) {
		t.Fatal("note-level findings must not count against a candidate")
	}
}

// 已通过的草稿不补采；没通过的补采一份，更好就换过来。
//
// 这条同时钉住两个方向的失败：无条件补采（用户明确要求生成要快，多花一次模型不可接受），
// 以及补采了但不选优（那这次调用等于白烧）。
func TestRerollTriggersOnlyOnUnverifiedResultAndPicksTheBetterOne(t *testing.T) {
	ctx := context.Background()
	const (
		bad  = `{"statements":[{"sql":"SELECT nmae FROM t_user"}]}`
		good = `{"statements":[{"sql":"SELECT name FROM t_user"}]}`
	)

	verified := newTestRun(t, []string{good}, nil)
	statements, checks, _, err := verified.generateAndCheck(ctx, "sys", "task")
	if verificationOf(checks) != verificationVerified {
		t.Fatalf("fixture must produce a verified first round, got %s", verificationOf(checks))
	}
	calls := verified.gen.calls
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	_, _, taken := verified.rerollCandidates(ctx, "sys", "task", statements, checks)
	if taken != 0 || verified.gen.calls != calls {
		t.Fatalf("a verified draft must not trigger a reroll, taken=%d extraCalls=%d", taken, verified.gen.calls-calls)
	}

	// 首轮有硬错误且关掉自纠：只剩补采这一条路，更好的那份必须被采纳
	run := newTestRun(t, []string{bad, good}, nil)
	run.spec.MaxRepairs = -1
	statements, checks, _, err = run.generateAndCheck(ctx, "sys", "task")
	if verificationOf(checks) != verificationInvalid {
		t.Fatalf("fixture must produce an invalid first round, got %s", verificationOf(checks))
	}
	statements, checks, taken = run.rerollCandidates(ctx, "sys", "task", statements, checks)
	if taken != 1 {
		t.Fatalf("expect exactly one taken candidate, got %d", taken)
	}
	if verificationOf(checks) != verificationVerified {
		t.Fatalf("the better candidate must win, got %s", verificationOf(checks))
	}
	if len(statements) != 1 || !strings.Contains(strings.ToLower(statements[0].SQL), "select name") {
		t.Fatalf("the winning statement must replace the result, got %+v", statements)
	}
}

// 补采是意图级旋钮：只有「产物要拿去用」的 SQL 意图开，行内补全与散文不开。
func TestOnlySqlIntentsDeclareCandidates(t *testing.T) {
	for _, intentId := range []string{"draft", "optimize", "fix"} {
		spec, ok := specOf(intentId)
		if !ok {
			t.Fatalf("intent %s must be registered", intentId)
		}
		if spec.ExtraCandidates <= 0 {
			t.Fatalf("intent %s must reroll when the draft is not verified", intentId)
		}
	}
	for _, intentId := range []string{"suggest", "explain"} {
		spec, _ := specOf(intentId)
		if spec.ExtraCandidates != 0 {
			t.Fatalf("intent %s must not reroll (latency / no verifiable output)", intentId)
		}
	}
}

// 词典服务未注册时生成照常：语义层是增强材料，缺它不该把一次生成变成一次报错。
func TestGenerationProceedsWithoutGlossaryService(t *testing.T) {
	run := newTestRun(t, []string{`{"statements":[{"sql":"SELECT name FROM t_user"}]}`}, nil)
	run.termSvc = nil
	run.loadTerms(context.Background())
	if run.terms != nil {
		t.Fatalf("no glossary service must leave no terms, got %d", len(run.terms))
	}
	if _, err := run.execute(context.Background()); err != nil {
		t.Fatalf("execute without glossary must work: %v", err)
	}
}

// 词典条目为空时召回与片段都安静跳过（离线测试与未配置库的常态）。
func TestGlossaryChannelsSkipEmptyInput(t *testing.T) {
	out, err := glossaryLinker{}.Propose(context.Background(), &linkCtx{text: "统计实例", snap: newBareSnapshot("t_user")})
	if err != nil || len(out) != 0 {
		t.Fatalf("no terms means no candidate, got %v err=%v", out, err)
	}
	out, err = glossaryLinker{}.Propose(context.Background(), &linkCtx{
		text:  "统计实例",
		snap:  newBareSnapshot("t_user"),
		terms: []*entity.DbAiTerm{{Term: "实", Tables: []string{"t_user"}}},
	})
	if err != nil || len(out) != 0 {
		t.Fatalf("a single-char term must not match (it would drag the whole glossary in), got %v", out)
	}
}

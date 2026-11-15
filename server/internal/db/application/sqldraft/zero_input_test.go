package sqldraft

// 「零输入」与「描述真的进提示词」的回归锁。
//
// 交互改造（选中 SQL 直接点解释/优化，不要求先打字）依赖两件事同时成立：
//  1. 后端按意图放行空描述——只在前端放宽的话，用户会看到「点了没反应」；
//  2. 用户输入的描述必须出现在交给模型的任务正文里——此前 explain_task.md 根本没有
//     描述槽位，用户写的「重点说明 WHERE 条件」被静默丢弃（界面看不出任何异常）。

import (
	"context"
	"strings"
	"testing"

	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/i18n"
)

func TestEmptyPromptAllowedOnlyForSqlMaterialIntents(t *testing.T) {
	ctx := i18n.NewCtxWithLang(context.Background(), i18n.Zh_CN)
	app := &DraftAppImpl{}

	// 按描述生成：没有描述就没有需求，必须报「请先填写需求描述」
	_, err := app.Draft(ctx, &Req{DbId: 1, Db: "demo", Intent: "draft"})
	if err == nil || !strings.Contains(err.Error(), i18n.TC(ctx, imsg.ErrSqlDraftPromptRequired)) {
		t.Fatalf("draft must still require a description, got %v", err)
	}

	// 解释/优化/改写：材料里有 SQL 就能干活，不得再拿「请填写描述」拦住
	for _, intent := range []string{"explain", "optimize", "fix"} {
		_, err := app.Draft(ctx, &Req{DbId: 1, Db: "demo", Intent: intent, AnchorStatement: "SELECT 1 FROM t_user"})
		if err != nil && strings.Contains(err.Error(), i18n.TC(ctx, imsg.ErrSqlDraftPromptRequired)) {
			t.Fatalf("intent %s must accept an empty description, got %v", intent, err)
		}
	}

	// 描述为空 + 材料也没有 SQL 时，意图仍放行（是否够用由生成侧判断），
	// 但超长材料这条闸门必须先于模型可用性生效（顺序错了就等于绕过上界）
	oversized := &Req{DbId: 1, Db: "demo", Intent: "optimize", AnchorStatement: strings.Repeat("x", maxMaterialRunes+1)}
	if _, err := app.Draft(ctx, oversized); err == nil || !strings.Contains(err.Error(), "需求描述或编辑器材料过长") {
		t.Fatalf("material gate must run for prompt-optional intents, got %v", err)
	}
}

// 每个意图的用户输入都必须落到自己的任务模板里（模板漏槽位时界面毫无异常，只能这样抓）
func TestEveryIntentTemplateCarriesUserPrompt(t *testing.T) {
	focus := "重点说明 WHERE 条件与空值处理"
	for _, name := range []string{"draft_task.md", "optimize_task.md", "fix_task.md", "explain_task.md"} {
		task, err := renderTemplate(name, map[string]any{
			"Prompt":    focus,
			"Materials": "SELECT 1",
			"Selection": "",
			"Refine":    "",
			"Context":   "ctx",
		})
		if err != nil {
			t.Fatalf("render %s: %v", name, err)
		}
		if !strings.Contains(task, focus) {
			t.Fatalf("%s drops the user's description, template output: %q", name, task)
		}
	}
}

// 零输入时模板必须给出明确指令，否则模型会把空目标当成「无需处理」原样返回
func TestPromptOptionalTemplatesFallback(t *testing.T) {
	for _, name := range []string{"optimize_task.md", "fix_task.md"} {
		task, err := renderTemplate(name, map[string]any{
			"Prompt":    "",
			"Materials": "SELECT 1 FROM t_user",
			"Context":   "ctx",
		})
		if err != nil {
			t.Fatalf("render %s: %v", name, err)
		}
		if !strings.Contains(task, "用户没有指定具体目标") {
			t.Fatalf("%s has no fallback for an empty goal: %q", name, task)
		}
	}
	// 解释类没有描述时不该留一个空标题
	explained, err := renderTemplate("explain_task.md", map[string]any{"Prompt": "", "Materials": "SELECT 1", "Context": "ctx"})
	if err != nil || strings.Contains(explained, "用户关注的方面") {
		t.Fatalf("explain must omit the focus section when there is none, err=%v out=%q", err, explained)
	}
}

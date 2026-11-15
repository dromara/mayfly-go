package sqldraft

// 意图注册：一个意图 = 提示词模板 + 召回/片段/校验组合 + 预算。
//
// 新增一种形态（解释、优化、方言转换）时这里加一行注册即可，管线零改动。
//
// Linkers 里同时列出提议阶段与扩展阶段的策略：顺序只在**同阶段内**有意义，
// 跨阶段的先后由策略自己的 Stage() 决定（见 registry.go 的 LinkStage）。
// 这样意图作者不需要、也无法把「依赖已选表集合」的策略排到提议阶段去。

import (
	"strings"
)

func init() {
	RegisterIntent(Spec{
		Id:              "draft",
		SystemTemplate:  "system.md",
		TaskTemplate:    "draft_task.md",
		Linkers:         []string{"identifier", "comment", "glossary", "relation", "sibling"},
		Fragments:       []string{"dialect", "tables", "values", "relations", "glossary", "examples"},
		Checkers:        []string{"parse", "value", "identifier", "write", "target_table", "no_source", "policy", "acl", "plan"},
		ExtraCandidates: 1,
	})
	// 行内补全：注释 → 一条 SQL。只要一次调用、不做自纠，
	// 因此上下文只来自「描述里出现的标识符 + 注释相关表」，准确性由前端只接受 verified 兜住
	RegisterIntent(Spec{
		Id:              "suggest",
		SystemTemplate:  "system_suggest.md",
		TaskTemplate:    "suggest_task.md",
		Linkers:         []string{"identifier", "comment", "glossary"},
		Fragments:       []string{"dialect", "tables", "values", "glossary"},
		Checkers:        []string{"parse", "value", "identifier", "no_source"},
		MaxRepairs:      -1,
		MaxOutputTokens: 220,
		// 不声明扩展类策略：行内补全要的是"光标处立刻出一条"，多带同族表与外键邻接表
		// 只会拉长上下文、拖慢首字节。真机实测库表注释为空时，确定性召回（表名/注释词面）
		// 命中不了中文描述，此时仍会走一次选择题——那是准确率与延迟的既有取舍。
		// AllowNoTables：补全是可选增强，确实无表时安静返回，不弹错。
		AllowNoTables: true,
	})
	// 优化：产物仍是 SQL，与 draft 同样走全套结构/取值/权限校验
	RegisterIntent(Spec{
		Id:              "optimize",
		SystemTemplate:  "system.md",
		TaskTemplate:    "optimize_task.md",
		Linkers:         []string{"identifier", "comment", "glossary", "relation", "sibling"},
		Fragments:       []string{"dialect", "tables", "values", "relations", "glossary"},
		Checkers:        []string{"parse", "value", "identifier", "write", "target_table", "no_source", "policy", "acl", "plan"},
		ExtraCandidates: 1,
		PromptOptional:  true,
	})
	// 解释：产物是散文，没有可校验的语句，也不提供写回
	RegisterIntent(Spec{
		Id:             "explain",
		SystemTemplate: "system_explain.md",
		TaskTemplate:   "explain_task.md",
		// 不做召回：待解释的 SQL 就在材料里，靠描述再找一遍表只会多花 45s
		SkipRecall: true,
		// 零输入即可解释：选中一段 SQL 直接点，不需要先打字说明「解释什么」
		PromptOptional: true,
		Fragments:      []string{"dialect"},
		Checkers:       nil,
		OutputKind:     OutputKindText,
		// 散文不需要自纠（没有校验结论可回喂）
		MaxRepairs: -1,
	})
	RegisterIntent(Spec{
		Id:              "fix",
		SystemTemplate:  "system.md",
		TaskTemplate:    "fix_task.md",
		Linkers:         []string{"identifier", "comment", "glossary", "relation", "sibling"},
		Fragments:       []string{"dialect", "tables", "values", "relations", "glossary"},
		Checkers:        []string{"parse", "value", "identifier", "write", "target_table", "no_source", "acl"},
		ExtraCandidates: 1,
		PromptOptional:  true,
	})
}

// materialsOf 意图的材料文本：参与召回的文本 + 交给模板的正文
//
// 统一从「描述 + 选区 + 光标语句」拼出来，而不是每个意图各取一份：
// 三者在召回里都应当算作「用户这次说的东西」，缺一份就会漏表。
func materialsOf(req *Req) string {
	parts := make([]string, 0, 3)
	if text := strings.TrimSpace(req.Prompt); text != "" {
		parts = append(parts, text)
	}
	if text := strings.TrimSpace(req.Selection); text != "" {
		parts = append(parts, text)
	}
	if text := strings.TrimSpace(req.AnchorStatement); text != "" {
		parts = append(parts, text)
	}
	// 上一版草稿也是材料：它引用的表要参与召回（细化通常延续同一批表），
	// 它的长度也必须计入材料闸门，否则这条入口可以绕过上界把长文本送进模型
	if text := strings.TrimSpace(req.RefineOf); text != "" {
		parts = append(parts, text)
	}
	return strings.Join(parts, "\n")
}

// sqlMaterials fix 意图要改写的 SQL 原文（选区优先，其次光标语句）
func sqlMaterials(req *Req) string {
	if text := strings.TrimSpace(req.Selection); text != "" {
		return text
	}
	return strings.TrimSpace(req.AnchorStatement)
}

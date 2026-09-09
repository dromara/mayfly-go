package contributor

import (
	"sort"
	"unicode/utf8"

	"mayfly-go/pkg/logx"
)

// preambleTokenBudget preamble 总预算（1 token ≈ 4 字符，超出时丢弃低优先级 Droppable 片段）
const preambleTokenBudget = 8000

// fragmentWithDefaults 为片段填充贡献者级默认值（优先级/保留策略，经可选接口声明）
func fragmentWithDefaults(c ContextContributor, frags []PromptFragment) []PromptFragment {
	dp, hasPriority := c.(PriorityContributor)
	dr, hasRetained := c.(RetainedContributor)
	for i := range frags {
		if hasPriority && frags[i].Priority == 0 {
			frags[i].Priority = dp.Priority()
		}
		if hasRetained && !frags[i].Retained {
			frags[i].Retained = dr.Retained()
		}
	}
	return frags
}

// BuildPreamble 拼装片段为 preamble 文本（preamble.rs 的预算感知裁剪）
//
//   - 片段按 Priority 降序排列后以分隔符连接
//   - 总预算超过 preambleTokenBudget 时，自低优先级起丢弃非 Retained 片段
//   - Retained 片段（安全约束等）永不丢弃
func BuildPreamble(fragments []PromptFragment) string {
	if len(fragments) == 0 {
		return ""
	}

	fragments = sortFragments(fragments)
	total := 0
	for _, f := range fragments {
		total += estimateFragmentTokens(f.Content)
	}

	// 预算超限时自低优先级（列表尾部）丢弃 Droppable 片段
	kept := fragments
	if total > preambleTokenBudget {
		kept = make([]PromptFragment, 0, len(fragments))
		remaining := total
		for _, f := range fragments {
			if !f.Retained && remaining > preambleTokenBudget {
				remaining -= estimateFragmentTokens(f.Content)
				logx.Warnf("[contributor] preamble over budget, dropping fragment: %s", f.Source)
				continue
			}
			kept = append(kept, f)
		}
	}

	parts := make([]string, 0, len(kept))
	for _, f := range kept {
		parts = append(parts, f.Content)
	}
	return joinPreambleSections(parts)
}

// sortFragments 按优先级降序排序（稳定排序，保持同优先级原顺序）
func sortFragments(fragments []PromptFragment) []PromptFragment {
	sorted := make([]PromptFragment, len(fragments))
	copy(sorted, fragments)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Priority > sorted[j].Priority
	})
	return sorted
}

// estimateFragmentTokens 估算片段 token 数（1 token ≈ 4 字符）
func estimateFragmentTokens(content string) int {
	return (utf8.RuneCountInString(content) + 3) / 4
}

// joinPreambleSections 以分隔符连接各 section
func joinPreambleSections(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	default:
		result := parts[0]
		for _, p := range parts[1:] {
			result += "\n\n---\n\n" + p
		}
		return result
	}
}

package contributor

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBuildPreambleDropsLowestPriorityFirst 预算超限时必须先丢**低优先级**片段。
//
// 缺陷类（真实发生过）：片段按 Priority 降序排列后，裁剪循环从头部开始遍历——超预算时
// 先丢的是 memory(60)/skill(50)，最后才轮到 environment(40)，与两处注释「自低优先级（列表尾部）
// 丢弃」正好相反。后果是 Priority 只影响拼接顺序、不影响存活，等于该字段半失效，
// 而丢掉的恰恰是最重要的记忆片段。
//
// 判据：三个等大片段（各约 5k token）总预算 15k > 8k 时，环境片段先被丢；
// 若继续超预算则技能片段被丢；记忆片段必须留在结果里
func TestBuildPreambleDropsLowestPriorityFirst(t *testing.T) {
	// 每个片段约 5000 token（20000 rune / 4），三者合计约 15000 > 8000
	big := strings.Repeat("x", 20000)
	fragments := []PromptFragment{
		{Source: "memory", Content: "MEMORY:" + big, Priority: 60},
		{Source: "skill", Content: "SKILL:" + big, Priority: 50},
		{Source: "environment", Content: "ENVIRONMENT:" + big, Priority: 40},
	}

	got := BuildPreamble(fragments)

	require.Contains(t, got, "MEMORY:", "高优先级片段在任何情况下都不能被优先丢弃")
	require.NotContains(t, got, "ENVIRONMENT:", "低优先级的环境片段应最先被丢弃")
	// 丢环境后仍是 10k > 8k，技能片段也应被丢
	require.NotContains(t, got, "SKILL:", "超预算时按优先级从低到高继续丢弃")
}

// TestBuildPreambleKeepsAllWithinBudget 未超预算时不得丢任何片段，且保持高优先级在前
func TestBuildPreambleKeepsAllWithinBudget(t *testing.T) {
	fragments := []PromptFragment{
		{Source: "environment", Content: "ENV", Priority: 40},
		{Source: "memory", Content: "MEM", Priority: 60},
	}
	got := BuildPreamble(fragments)
	require.Contains(t, got, "MEM")
	require.Contains(t, got, "ENV")
	require.Less(t, strings.Index(got, "MEM"), strings.Index(got, "ENV"), "拼接顺序仍须高优先级在前")
}

package application

import (
	"os"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/ai/session"
)

// assistantWithCall 助手发起一次工具调用（与紧随其后的工具结果构成一组）
func assistantWithCall(id string) *session.Message {
	return &session.Message{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{ID: id}}}
}

// toolResult 工具结果消息
func toolResult(id, content string) *session.Message {
	return &session.Message{Role: schema.Tool, ToolCallId: id, Content: content}
}

// TestTrimToLastCompleteGroupsNeverStartsWithOrphanToolResult 历史裁剪不得产出孤立工具结果。
//
// 一个 tool_call item 会展开成「助手调用 + 工具结果」两条消息。按条数硬切尾部窗口
// （msgs[n-limit:]）可能把助手调用切掉而留下它的工具结果，得到以 tool 角色开头的历史
// ——那是模型 API 的非法消息序列（tool 消息必须跟随产生它的助手调用），实际表现为模型
// 困惑、答非所问（曾观测到模型延续上一轮主题作答）
func TestTrimToLastCompleteGroupsNeverStartsWithOrphanToolResult(t *testing.T) {
	msgs := []*session.Message{
		{Role: schema.User, Content: "第一轮问题"},
		assistantWithCall("call-1"), toolResult("call-1", "结果1"),
		assistantWithCall("call-2"), toolResult("call-2", "结果2"),
		{Role: schema.Assistant, Content: "第一轮回答"},
	}

	// limit=4 若硬切，起点会落在 toolResult("call-2") 上（孤立工具结果）
	trimmed := trimToLastCompleteGroups(msgs, 4)
	require.NotEmpty(t, trimmed)
	require.NotEqual(t, schema.Tool, trimmed[0].Role, "历史首条不能是孤立的工具结果")
	require.Equal(t, schema.Assistant, trimmed[0].Role, "应前退到与工具结果成对的助手调用")
	// 前退后窗口多带一条：组完整性优先于「正好 limit 条」
	require.Len(t, trimmed, 5)
}

// TestTrimToLastCompleteGroupsKeepsWholeSegmentWhenAllToolResults 极端脏数据：整段都是工具结果时
// 保留整段而不是丢掉历史（宁可窗口偏大，也不制造缺失上下文的输入）。
func TestTrimToLastCompleteGroupsKeepsWholeSegmentWhenAllToolResults(t *testing.T) {
	msgs := []*session.Message{
		toolResult("a", "1"), toolResult("b", "2"), toolResult("c", "3"),
	}

	trimmed := trimToLastCompleteGroups(msgs, 1)
	require.Len(t, trimmed, 3, "退到头部仍全为工具结果时保留整段，不丢历史")
}

// TestTrimToLastCompleteGroupsRespectsLimitWhenBoundaryIsClean 切点本就落在合法位置时不多带消息。
func TestTrimToLastCompleteGroupsRespectsLimitWhenBoundaryIsClean(t *testing.T) {
	msgs := []*session.Message{
		{Role: schema.User, Content: "q1"},
		{Role: schema.Assistant, Content: "a1"},
		{Role: schema.User, Content: "q2"},
		{Role: schema.Assistant, Content: "a2"},
	}

	trimmed := trimToLastCompleteGroups(msgs, 2)
	require.Len(t, trimmed, 2, "切点落在用户消息上时按 limit 精确裁剪")
	require.Equal(t, "q2", trimmed[0].Content)
}

// TestHistoryTrimSingleEntryPoint 历史裁剪必须只有「组对齐」这一个入口。
//
// 为什么用静态守卫而不是再加一条行为用例：上面几条已锁住函数行为，真正会退化的是**调用点**——
// 有人把 GetHistory 改回按条数硬切时，函数自己的用例照样全绿（实测这条变异存活过一次）。
// 而装配整条 GetHistory 需要 fake 仓储与 payload 构造，静态断言在这里更便宜且同样有牙
func TestHistoryTrimSingleEntryPoint(t *testing.T) {
	src, err := os.ReadFile("session_store.go")
	require.NoError(t, err, "读取 session_store.go 失败")
	text := string(src)

	require.Contains(t, text, "trimToLastCompleteGroups(msgs, readLimit)",
		"历史裁剪必须走组对齐入口，不能绕过它另写一套")
	require.NotRegexp(t, `msgs\s*=\s*msgs\[len\(msgs\)-\w+:\]`,
		"不得保留按条数硬切的写法（会产出以孤立工具结果开头的非法消息序列）")
}

// TestTrimToLastCompleteGroupsKeepsLatestTail 裁剪必须保留的是**最近**的内容（尾部语义不变）。
func TestTrimToLastCompleteGroupsKeepsLatestTail(t *testing.T) {
	msgs := []*session.Message{
		{Role: schema.User, Content: "旧问题"},
		assistantWithCall("c1"), toolResult("c1", "旧结果"),
		{Role: schema.Assistant, Content: "最终答案"},
	}

	trimmed := trimToLastCompleteGroups(msgs, 1)
	require.Equal(t, "最终答案", trimmed[len(trimmed)-1].Content, "裁剪后必须以最近一条消息收尾")
}

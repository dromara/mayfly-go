package application

import (
	"strings"
	"testing"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/ai/pkg/headless"

	"github.com/stretchr/testify/require"
)

// TestParsePlanSteps 编排计划解析：纯 JSON / 代码块包裹 / 非法输出降级 / 字段兜底
func TestParsePlanSteps(t *testing.T) {
	// 纯 JSON 数组
	steps := parsePlanSteps(`[{"title":"查指标","action":"查询 cpu_rate","readonly":true},{"title":"处置","action":"重启服务","readonly":false}]`)
	require.Len(t, steps, 2)
	require.Equal(t, "查指标", steps[0].Title)
	require.True(t, steps[0].ReadOnly)
	require.False(t, steps[1].ReadOnly)

	// 模型输出夹带说明文字/代码围栏：取首个 '[' 到最后一个 ']'
	steps = parsePlanSteps("计划如下：\n```json\n[{\"action\":\"检查磁盘\"}]\n```\n请确认")
	require.Len(t, steps, 1)
	// title 缺省回退 action 截断
	require.Equal(t, "检查磁盘", steps[0].Title)

	// 非法输出：返回空（调用方降级为单步骤）
	require.Empty(t, parsePlanSteps("我认为应该先看看 CPU"))
	require.Empty(t, parsePlanSteps("[{broken json"))

	// 空 action 的步骤被过滤
	require.Empty(t, parsePlanSteps(`[{"title":"x","action":""}]`))
}

// TestApplyDryRun 影子模式变换：强制降只读档 + 指令前置约束；非影子原样透传
func TestApplyDryRun(t *testing.T) {
	run := &entity.Run{ProfileId: headless.ProfileExecutor, Instruction: "执行该变更", DryRun: true}
	profileId, instruction := applyDryRun(run)
	require.Equal(t, headless.ProfileDiagnostician, profileId)
	require.True(t, strings.HasPrefix(instruction, "【影子模式 / DRY-RUN】"))
	require.Contains(t, instruction, "执行该变更")

	// 非影子模式：原样透传
	run2 := &entity.Run{ProfileId: headless.ProfileExecutor, Instruction: "执行该变更"}
	profileId2, instruction2 := applyDryRun(run2)
	require.Equal(t, headless.ProfileExecutor, profileId2)
	require.Equal(t, "执行该变更", instruction2)
}

// TestParseEvalScore 评分提取：纯 JSON / 包裹文本 / 越界夹紧 / 缺失记 0
func TestParseEvalScore(t *testing.T) {
	score, _ := parseEvalScore(`{"score": 8, "verdict": "结论一致"}`)
	require.Equal(t, 8, score)

	score, _ = parseEvalScore("判定如下：\n{\"score\": 10, \"verdict\": \"完全一致\"}\n以上")
	require.Equal(t, 10, score)

	score, _ = parseEvalScore(`{"score": 99}`)
	require.Equal(t, 10, score) // 越界夹紧

	score, verdict := parseEvalScore("无法解析的输出")
	require.Equal(t, 0, score)
	require.Equal(t, "无法解析的输出", verdict)
}

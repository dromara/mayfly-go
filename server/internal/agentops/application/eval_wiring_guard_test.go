package application

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// 评测链路的接线守卫：急停守护与重放会话回写。
//
// 两条都是「摘掉后没有任何红」的静默缺陷：漏 guardDispatch 时急停看板显示已停止派发而评测
// 照常烧两段 LLM（恰是配额消耗最高的入口）；漏会话列回写时每个评测运行的「回放」按钮恒灰、
// 评分不可复核，而评测功能表面完全正常。

func TestReplayEvalGuardsKillSwitch(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(".", "eval_service.go"))
	require.NoError(t, err)
	source := string(b)
	i := strings.Index(source, "func (r *runServiceAppImpl) ReplayEval(")
	require.Greater(t, i, 0, "ReplayEval 改名/删除必须同步本守卫")
	body := source[i : strings.Index(source[i:], "\n}")+i]
	// 守护必须在 InsertIfAbsent 之前：先落 running 再拒绝会留悬挂运行（与 orchestrator 同模式）
	guardAt := strings.Index(body, "guardDispatch")
	insertAt := strings.Index(body, "InsertIfAbsent")
	require.Greater(t, guardAt, 0, "ReplayEval 必须前置急停守护")
	require.Greater(t, insertAt, 0)
	require.Less(t, guardAt, insertAt, "守护必须在落库前")
}

func TestFinishEvalPersistsReplaySession(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(".", "eval_service.go"))
	require.NoError(t, err)
	source := string(b)
	i := strings.Index(source, "func (r *runServiceAppImpl) finishEval(")
	require.Greater(t, i, 0)
	body := source[i : strings.Index(source[i:], "\n}")+i]
	// 重放运行是 Persist:true 产生的持久化会话；不回写列则前端回放按钮恒灰、评测会话成孤儿
	require.Contains(t, body, `"conversationId"`)
	require.Contains(t, body, `"turnId"`)
	// token 三个分量必须同写：只写 total 会让看板 prompt/completion 少计评测那两段，
	// 跑过评测后「输入 + 输出 ≠ 总数」，成本归因指标静默失真（本轮修的就是只写 total）
	require.Contains(t, body, `"promptTokens"`, "评测运行必须回写输入用量")
	require.Contains(t, body, `"completionTokens"`, "评测运行必须回写输出用量")
	require.Contains(t, body, `"totalTokens"`)
}

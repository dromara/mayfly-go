package api

import (
	"os"
	"strings"
	"testing"
	"time"

	"mayfly-go/internal/ai/application/dto"

	"github.com/stretchr/testify/require"
)

// TestToTurnItemVOCarriesExtra TurnItem 的对外投影必须带 Extra。
//
// Extra 是中断徽章与审批卡片的数据源（extra.interrupt.resume）：单 turn 刷新路径曾漏带它，
// 走刷新时卡片会丢「已批准/已完善」状态，而接口返回 200、前端不报错
func TestToTurnItemVOCarriesExtra(t *testing.T) {
	extra := map[string]any{"interrupt": map[string]any{"kind": "approval"}}
	now := time.Now()
	got := toTurnItemVO(&dto.TurnItemDTO{
		Id: 9, TurnId: "t1", ItemType: "tool_call", ItemId: "i1",
		Status: "interrupted", ToolCallId: "call-1", Extra: extra, CreateTime: &now,
	})
	require.Equal(t, extra, got.Extra, "Extra 必须原样透出")
	require.Equal(t, "call-1", got.ToolCallId)
	require.Equal(t, "interrupted", got.Status)
	require.NotNil(t, got.CreateTime)
}

// TestTurnItemVOProjectionIsSingleSourced 两个查询接口必须共用同一投影函数。
//
// 判据：ai.go 里 `&vo.TurnItemVO{` 只允许出现一次（在 toTurnItemVO 内）。
// 分页拉取与单 turn 刷新各写一份字面量正是 Extra 漏带的成因
func TestTurnItemVOProjectionIsSingleSourced(t *testing.T) {
	raw, err := os.ReadFile("ai.go")
	require.NoError(t, err)
	text := stripLineCommentsForTurnItemGuard(string(raw))

	require.Equal(t, 1, strings.Count(text, "&vo.TurnItemVO{"),
		"TurnItem 投影出现了第二个字面量：两个查询接口会再次漂移（Extra/字段漏带）")
	require.Equal(t, 2, strings.Count(text, "collx.ArrayMap(g.Items, toTurnItemVO)")+
		strings.Count(text, "collx.ArrayMap(items, toTurnItemVO)"),
		"两个查询接口都必须走共用投影函数")
}

// stripLineCommentsForTurnItemGuard 去掉行注释（说明注释会引用被判据排斥的字面量）
func stripLineCommentsForTurnItemGuard(source string) string {
	var kept []string
	for _, line := range strings.Split(source, "\n") {
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

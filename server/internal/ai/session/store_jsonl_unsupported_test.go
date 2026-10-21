package session

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestJSONLStoreMessageQueryUnsupported 兜底后端对「按条件查/改单条消息」必须显式报不支持。
//
// 缺陷类（真实发生过）：JSONLStore.GetMessage 用空 sessionKey 读历史再按 OR 条件过滤，
// 结果恒为空切片——调用方把「不支持」当成「查不到」，审批/补全的 resume 信息落不了库
// 却查不出原因；UpdateMessage 则直接返回一句无类型的 errors.New，无法被 errors.Is 识别。
// 现在两者统一返回 ErrStoreUnsupported，调用方可以据此降级并留下可读日志。
func TestJSONLStoreMessageQueryUnsupported(t *testing.T) {
	store, err := NewStoreJSONL(t.TempDir())
	require.NoError(t, err)

	_, err = store.GetMessage(context.Background(), &MessageQuery{ToolCallId: "call-1"})
	require.ErrorIs(t, err, ErrStoreUnsupported)

	err = store.UpdateMessage(context.Background(), &Message{Id: 1})
	require.ErrorIs(t, err, ErrStoreUnsupported)
}

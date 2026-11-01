package sqlexec

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEstimateRowBytes 批预算的行字节估算口径。
//
// 查询导出此前只有行数预算（500 行）：500 行大 blob/text 会把整批原始值同时留在内存里，
// 与 dbm/export 的 8MB 字节预算不等价。估算只用于切批，不追求精确，但必须对
// 「字符串/二进制按长度、NULL 与标量按固定开销」保持单调（越大的值越早触发 flush）。
func TestEstimateRowBytes(t *testing.T) {
	require.Equal(t, 4, estimateRowBytes([]any{nil}))
	require.Equal(t, 10, estimateRowBytes([]any{"0123456789"}))
	require.Equal(t, 10, estimateRowBytes([]any{[]byte("0123456789")}))
	require.Equal(t, 16, estimateRowBytes([]any{int64(1)}))
	require.Equal(t, 56, estimateRowBytes([]any{nil, "0123456789", []byte("0123456789"), int64(1), true}))
	// 单调性：更大的字符串必须估算出更多字节（保证大值能更早触发 flush）
	require.Greater(t, estimateRowBytes([]any{strings.Repeat("x", 100)}), estimateRowBytes([]any{"x"}))
}

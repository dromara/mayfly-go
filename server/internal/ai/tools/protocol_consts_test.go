package tools

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestProtocolMarkersHaveSingleDefinition 跨层协议字符串只允许有一处定义。
//
// 审查抓出的最危险一类重复：工具侧写这些前缀、agent 与 event_mapper 侧读它们。改一边不报错，
// 只表现为「用户拒了命令却被计入工具执行失败」「取消被当成系统故障弹红色提示」。
// 因此这里扫源码（排除测试文件与本定义文件）断言第二份字面量不存在
func TestProtocolMarkersHaveSingleDefinition(t *testing.T) {
	markers := []string{"[OPERATION_REJECTED]", "[PARAM_COMPLETION_CANCELLED]"}

	var offenders []string
	err := filepath.Walk(".", func(path string, info fs.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return walkErr
		}
		if filepath.Base(path) == "protocol_consts.go" {
			return nil
		}
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, marker := range markers {
			if strings.Contains(string(src), marker) {
				offenders = append(offenders, path+" :: "+marker)
			}
		}
		return nil
	})
	require.NoError(t, err)
	require.Empty(t, offenders, "这些跨层协议字符串必须引用 tools 包常量（protocol_consts.go），不得再抄一份字面量")
}

// TestProtocolConstPredicates 判定函数必须认得工具实际产出的完整文案形态。
func TestProtocolConstPredicates(t *testing.T) {
	require.True(t, IsRejectedToolResult(paramCompletionFreeOfMarker()))
	require.False(t, IsRejectedToolResult("db connection failed: invalid connection"))
	require.True(t, IsCancelledToolResult(paramCompletionCancelledText))
	require.False(t, IsCancelledToolResult("ac not found"))
}

// paramCompletionFreeOfMarker 审批拒绝文案的生产者（走真实函数，不在此抄一份字面量）
func paramCompletionFreeOfMarker() string {
	return RejectedToolPrefix + " The tool 'ExecSql' was explicitly rejected by the user."
}

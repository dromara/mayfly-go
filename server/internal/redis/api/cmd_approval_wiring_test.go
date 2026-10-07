package api

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunCmdKeepsTheDecisionCode 命令执行失败的中断必须先保住分流错误码。
//
// 这一条不测出来就会静默失效：错误码被 biz.ErrIsNil 重新包成 400 之后，
// 界面照常显示提示、只是再也不会出现「提交工单」入口或「仅提醒」确认框，
// 被拦下的人只能自己切去发起流程重敲一遍命令。4001 与 4002 走同一个接缝，两个都要保住
func TestRunCmdKeepsTheDecisionCode(t *testing.T) {
	source := readGoFile(t, "cmd.go")

	// 定位 RunCmd 失败后的断言序列：Preserve 必须出现在 biz.ErrIsNil(err) 之前
	pattern := regexp.MustCompile(`(?s)redisApp\.RunCmd\((.*?)biz\.ErrIsNil\(err\)`)
	matched := pattern.FindStringSubmatch(source)
	require.NotNil(t, matched, "cmd.go 里 RunCmd 的失败断言结构变了，请同步检查分流错误码是否还能透出")
	require.Contains(
		t,
		matched[1],
		"flowapp.PreserveDecisionCode(err)",
		"RunCmd 失败必须先经过 PreserveDecisionCode，否则 4001/4002 会被包成 400",
	)
}

// readGoFile 读取同包源码用于接线断言（测试运行目录即包目录）
func readGoFile(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(".", name))
	require.NoError(t, err)
	return string(content)
}

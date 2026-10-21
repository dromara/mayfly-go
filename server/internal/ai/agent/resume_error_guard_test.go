package agent

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestResumeErrorIsPropagatedToFinalize 恢复分支的错误必须回填到收尾用的局部 err。
//
// 缺陷类（真实发生过）：`Run` 里的 err 是为了让 defer 中的 finalizeTurn 看到而提前声明的**局部变量**；
// `startRun` 分支写的是 `events, err = ...`（正确），而 resume 分支曾是 `return nil, resumeErr`——
// 局部 err 仍为 nil，于是同一轮在 turn_item 里落成 completed、通知按成功上报，调用方却拿到失败。
// 审批恢复时 checkpoint 过期是设计内常见路径，所以这种分裂会被用户直接看到。
//
// 判据：文件内不得再出现未回填的 `return nil, resumeErr`，且必须存在回填。
// 行注释先剥掉——本文件的说明注释会原样引用这两种写法
func TestResumeErrorIsPropagatedToFinalize(t *testing.T) {
	raw, err := os.ReadFile("agent.go")
	require.NoError(t, err)
	text := stripLineComments(string(raw))

	require.NotContains(t, text, "return nil, resumeErr",
		"resume 错误直接返回会让 defer 里的 finalizeTurn 看到 nil：item 落成 completed、调用方拿到失败")
	require.Contains(t, text, "err = resumeErr",
		"恢复分支必须先 err = resumeErr 再返回，收尾逻辑依赖这个局部 err")
}

// stripLineComments 去掉行注释（说明注释里会引用被禁的写法）
func stripLineComments(source string) string {
	var kept []string
	for _, line := range strings.Split(source, "\n") {
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

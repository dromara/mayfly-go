package tools

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestInterruptActionsReferToProtocolConstants 中断恢复的动作判定必须引用 protocol 常量，
// 不得在工具层再手写 "approve"/"reject"/"complete" 字面量。
//
// 缺陷类（真实发生过）：同一组动作值散落在映射函数、审批 switch、参数补全的本地常量三处。
// 新增一种决策时改一处不会报错——漏改的一侧会把它当成「无效动作」/「未完成」，
// 而编译、类型与既有用例全绿（无效动作分支只在真机点一次才走到）。
//
// 判据：两个文件里不得出现与这三个值比较的字面量（注释会引用它们，先剥掉）
func TestInterruptActionsReferToProtocolConstants(t *testing.T) {
	files := []string{"interrupt_approval.go", "interrupt_param_completion.go"}
	banned := []string{
		`== "approve"`, `!= "approve"`,
		`== "reject"`, `!= "reject"`,
		`== "complete"`, `!= "complete"`,
		`paramActionComplete`,
	}

	for _, file := range files {
		raw, err := os.ReadFile(file)
		require.NoErrorf(t, err, "读取 %s 失败（文件改名时本守卫须同步更新）", file)
		text := stripLineCommentsForActionGuard(string(raw))
		for _, b := range banned {
			require.NotContainsf(t, text, b,
				"%s 仍手写动作字面量（%s）：应引用 protocol.ActionApprove/ActionReject/ActionComplete", file, b)
		}
	}
}

// stripLineCommentsForActionGuard 去掉行注释（说明注释会引用被禁的字面量）
func stripLineCommentsForActionGuard(source string) string {
	var kept []string
	for _, line := range strings.Split(source, "\n") {
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

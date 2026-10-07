package mcm

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSessionWritesNoticeToTerminal 终端会话拿到提醒后必须真的写回页面。
//
// 这一段没有任何单元测试能够触达（要真 SSH 连接与 ws 会话），却正是「仅提醒」在机器侧的唯一出口：
// 一旦这里丢掉，命令照常执行、页面一句提示也没有，管理员配的提醒级别只剩服务端日志一处，
// 从界面上看不出任何异常，所以只能用接线断言把它钉住
func TestSessionWritesNoticeToTerminal(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(".", "terminal_session.go"))
	require.NoError(t, err)

	session := string(content)
	require.Regexp(t, `notice, handleErr := ts\.handler\.PreWriteHandle\(data\)`, session, "PreWriteHandle 的提醒返回值必须被会话接走")
	// 必须同时断言「写回」与「它自己的守卫条件」：只断言写调用存在的话，
	// 把条件改成 `if false && notice != ""` 这类永远不成立的写法照样能过，提醒依旧静默消失
	require.Regexp(t, `if notice != "" \{\n\s*ts\.WriteToWs\(GetWarnContentRn\(notice\)\)`, session, "提醒必须在拿到非空 notice 时写回终端")

	// 写回必须发生在「命令被拦下」分支之后：拦下时不该再补一行提醒
	blocked := regexp.MustCompile(`(?s)if handleErr != nil \{.*?\n\t\t\t\t\t\}`).FindString(session)
	require.NotEmpty(t, blocked, "拒绝分支的结构变了，请同步检查提醒是否会被误当成执行成功打出来")
	require.NotContains(t, blocked, "GetWarnContentRn")
}

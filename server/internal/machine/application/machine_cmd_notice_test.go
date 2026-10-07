package application

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMachineCmdHandsTheNoticeBack 机器命令的「仅提醒」命中必须交回调用方写进终端。
//
// 机器场景没有审批通道，提醒不能转成工单，终端回显就是它唯一的出口：
// 这一条静默失效后，界面照常执行命令、一句提示也没有，管理员配的「仅提醒」等于没配，
// 而从页面看不出任何异常——只有服务端日志里有
func TestMachineCmdHandsTheNoticeBack(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(".", "machine_trigger.go"))
	require.NoError(t, err)

	matched := regexp.MustCompile(`(?s)func CheckMachineCmd\(.*?
}`).FindString(string(content))
	require.NotEmpty(t, matched, "CheckMachineCmd 的定义结构变了，请同步确认提醒是否仍然交回调用方")
	require.Contains(t, matched, "flowapp.WarnNotice(ctx, decision), nil", "提醒必须作为返回值交回终端会话")
	require.Contains(t, matched, "(notice string, err error)", "返回值少了 notice，终端就再也收不到提醒")
}

// TestTerminalFilterReceivesTheNotice Web 终端的过滤器必须把 notice 透传出去（而不是就地丢弃）
func TestTerminalFilterReceivesTheNotice(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(".", "machine_term_op.go"))
	require.NoError(t, err)

	matched := regexp.MustCompile(`(?s)CmdFilterFuncs = .*?
	\}\}`).FindString(string(content))
	require.NotEmpty(t, matched, "终端命令过滤器的构造结构变了，请同步检查提醒是否仍然透传")
	require.Contains(t, matched, "return CheckMachineCmd(ctx, cli.Info.CodePath, cmd)", "过滤器必须原样返回 CheckMachineCmd 的 notice")
	require.Contains(t, matched, "func(cmd string) (string, error)", "过滤器签名少了返回值，终端就再也打印不出提醒")
}

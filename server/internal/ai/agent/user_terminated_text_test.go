package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestUserTerminatedTextMapsCancelAndReject 用户主动终止的错误原文必须换成用户可读短句。
//
// 端到端实测到的形态：取消决策生效后，落库的 assistant 正文是一整串内部结构
// （[NodeRunError] + [LocalFunc] + toolName + node path），既不可读也泄漏实现细节
func TestUserTerminatedTextMapsCancelAndReject(t *testing.T) {
	ctx := context.Background()

	cancelErr := "[NodeRunError] failed to stream tool call call_78f4efa0: [LocalFunc] failed to invoke tool, " +
		"toolName=MachineCommandExec, err=[PARAM_COMPLETION_CANCELLED] The user has cancelled the parameter " +
		"completion for this tool.\nPlease do not retry...\n------------------------\nnode path: [node_1, ToolNode]"

	text, ok := userTerminatedText(ctx, cancelErr)
	require.True(t, ok, "取消必须映射成用户短句")
	require.NotEmpty(t, text)
	require.NotContains(t, text, "NodeRunError", "不得把内部节点结构给用户")
	require.NotContains(t, text, "LocalFunc")
	require.NotContains(t, text, "node path")
	require.NotContains(t, text, "call_78f4efa0")

	rejectErr := "[NodeRunError] ... err=[OPERATION_REJECTED] The tool 'ExecSql' was explicitly rejected by the user."
	text, ok = userTerminatedText(ctx, rejectErr)
	require.True(t, ok, "审批拒绝同样应换成用户短句")
	require.NotContains(t, text, "OPERATION_REJECTED")
	require.NotContains(t, text, "NodeRunError")
}

// TestUserTerminatedTextKeepsRealFailuresUntouched 真实故障必须原样保留，不能被伪装成「用户已取消」。
//
// 这条反向断言是重点：一旦把匹配条件放宽（例如按「含 error 字样」改写），数据库连接失败这类
// 真实故障就会被伪装成用户主动终止，运维排查时会完全看不到真实原因
func TestUserTerminatedTextKeepsRealFailuresUntouched(t *testing.T) {
	ctx := context.Background()

	for _, errText := range []string{
		"[NodeRunError] db connection failed: Error 1049 (42000): Unknown database 'x'",
		"[LocalFunc] failed to invoke tool, toolName=MachineCommandExec, err=ac not found",
		"dial tcp 127.0.0.1:22: connect: connection refused",
		"",
	} {
		_, ok := userTerminatedText(ctx, errText)
		require.False(t, ok, "非用户主动终止的错误不得被改写：%q", errText)
	}
}

// TestIsUserTerminated api 层用来决定「要不要下发 error 事件」的导出口。
//
// 判据必须与 userTerminatedText 同源：两处各写一套匹配就会漂移（一处认 cancel、一处不认），
// 结果是同一次终止既落收尾气泡又弹一行红色内部错误——正是真机复验发现的残留
func TestIsUserTerminated(t *testing.T) {
	require.False(t, IsUserTerminated(nil), "nil 错误不是用户终止")
	require.False(t, IsUserTerminated(errors.New("dial tcp 127.0.0.1:22: connect: connection refused")),
		"真实故障不能被归为用户终止，否则错误事件被静默吞掉、前端再也看不到失败")

	require.True(t, IsUserTerminated(errors.New("... err=[PARAM_COMPLETION_CANCELLED] The user has cancelled ...")))
	require.True(t, IsUserTerminated(errors.New("... err=[OPERATION_REJECTED] The tool 'ExecSql' was explicitly rejected ...")))
}

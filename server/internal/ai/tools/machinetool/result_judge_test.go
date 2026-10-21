package machinetool

import (
	"testing"

	"mayfly-go/internal/ai/tools"

	"github.com/stretchr/testify/require"
)

// judgeCommandExecResult 的判据边界。
//
// 这条判定的存在理由：本工具在命令退出码非零时仍返回正常结果（为了把输出交给模型），
// 所以「失败」只存在于载荷里。判定过宽会把成功当失败（提案误判 failed、工单被改判处理失败），
// 过窄则让失败被当成功（提案记 executed，实测过）。
func TestJudgeCommandExecResult(t *testing.T) {
	cases := []struct {
		name    string
		output  string
		wantOK  bool
		wantSub string
	}{
		{"成功", `{"machineId":1,"output":"ok","success":true}`, true, ""},
		{"失败并带输出", `{"machineId":1,"output":"bash: false: command failed","success":false}`, false, "bash: false: command failed"},
		{"失败但输出为空", `{"machineId":1,"output":"","success":false}`, false, "exit status != 0"},
		// 拿不到结构化结论就不猜：非 JSON（老数据/异常包裹）与缺字段都按成功处理
		{"非 JSON", `bash: x: command not found`, true, ""},
		{"缺 success 字段", `{"machineId":1,"output":"ok"}`, true, ""},
		{"空串", ``, true, ""},
	}
	for _, tc := range cases {
		ok, reason := judgeCommandExecResult(tc.output)
		require.Equal(t, tc.wantOK, ok, tc.name)
		if tc.wantSub != "" {
			require.Contains(t, reason, tc.wantSub, tc.name)
		} else {
			require.Empty(t, reason, tc.name)
		}
	}
}

// TestCommandExecResultJudgeRegistered 判定必须在包 init 期注册。
//
// 守卫理由：判定注册漏了不会报错，只会让「命令失败」退回成「变更已生效」——这是已经在真机上
// 出现过的假成功形态。而注册早于工具首次构造是硬要求（收集工具消息时就要用）
func TestCommandExecResultJudgeRegistered(t *testing.T) {
	require.True(t, tools.HasResultJudge(tools.ToolNameMachineCommandExec),
		"MachineCommandExec 未注册结果判定：命令退出非零会被当成执行成功")
}

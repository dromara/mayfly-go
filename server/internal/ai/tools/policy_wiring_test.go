package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAgentToolsConsumePolicyDecision 代执行入口必须真正消费策略结论。
//
// 这两处曾经调了引擎再把结论丢掉：db 工具把「需审批」只写成一行日志（注释还写着
// 「用户已批准」，把 Agent 里点「允许本次执行」当成工单审批），且从未判「禁止执行」；
// 判定出错时更是 continue 直接放行。结果就是管理员拦得住编辑器、拦不住 Agent。
func TestAgentToolsConsumePolicyDecision(t *testing.T) {
	dbSource := readToolFile(t, filepath.Join("dbtool", "sql_exec.go"))
	machineSource := readToolFile(t, filepath.Join("machinetool", "command_exec.go"))

	// 判定走应用层的统一入口，而不是在工具里自己拼一套
	require.Contains(t, dbSource, "CheckSqlsWithoutTicket(ctx, conn, sqlStatements)", "db 工具必须复用应用层的策略判定")
	require.NotContains(t, dbSource, "user approval already obtained", "不得把用户放行本次工具调用当成工单审批")
	require.NotContains(t, dbSource, "decisionErr", "判定失败不能吞掉继续执行")

	for _, source := range []string{dbSource, machineSource} {
		// 治理结论一律不可重试：退回给模型重试等于给它换写法绕过治理的机会
		require.NotContains(t, source, "NewToolError(err, RecoverRetry)")
		blocked := policyBlockBranches(source)
		require.NotEmpty(t, blocked, "工具里没有对策略拒绝结论的处理分支")
		for _, branch := range blocked {
			require.Contains(t, branch, "RecoverNone", "策略拒绝必须按不可重试上抛")
			require.NotContains(t, branch, "RecoverRetry", "策略拒绝不得允许重试")
		}
	}

	// 提醒不阻断，但要回传给模型：否则它会向用户汇报「一切正常」
	require.Contains(t, dbSource, "Warnings:")
	require.Contains(t, machineSource, "notice + \"\\n\" + output")
}

// policyBlockBranches 取出对策略结论做处理的那几个分支体
func policyBlockBranches(source string) []string {
	marks := []string{"CheckSqlsWithoutTicket(ctx, conn, sqlStatements)", "CheckMachineCmd(ctx, cli.Info.CodePath, param.Command)"}
	var branches []string
	for _, mark := range marks {
		start := strings.Index(source, mark)
		if start < 0 {
			continue
		}
		end := strings.Index(source[start:], "\n\t\t\t}")
		if end < 0 {
			continue
		}
		branches = append(branches, source[start:start+end])
	}
	return branches
}

func readToolFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err, path)
	return string(content)
}

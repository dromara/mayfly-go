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

// TestAgentQueryToolIsReadOnlyAndGoverned 钉住 DbQueryData（只读、无审批的查询工具）的两条不变量：
//  1. 只读判定走 sqlparser.Classify 单一真源，不得把整个 OtherStmt 兜底大类当只读放行
//     （旧实现如此，使 CALL/DO/SET/VACUUM 等带副作用语句绕过 ExecSql 审批从查询通道执行写）；
//  2. 只读通道也过触发策略（与交互执行/ExecSql 同源），且拒绝结论不可重试。
func TestAgentQueryToolIsReadOnlyAndGoverned(t *testing.T) {
	querySource := readToolFile(t, filepath.Join("dbtool", "query_data.go"))

	require.Contains(t, querySource, "sqlparser.Classify", "查询工具必须复用 sqlparser.Classify 判定语句类型")
	require.NotContains(t, querySource, "*sqlstmt.SelectStmt, *sqlstmt.WithStmt, *sqlstmt.OtherStmt", "不得把 OtherStmt 整体当只读放行")
	require.Contains(t, querySource, "CheckSqlsWithoutTicket(ctx, conn, readOnlySqls)", "查询工具必须复用应用层的策略判定")
	require.Contains(t, querySource, "tools.RecoverNone", "策略拒绝必须按不可重试上抛")
}

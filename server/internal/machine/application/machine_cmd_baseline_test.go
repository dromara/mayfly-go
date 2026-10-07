package application

import (
	"os"
	"strings"
	"testing"
)

// 命令过滤规则与审批结果这两条链路都依赖「调用位置」而不是调用与否：
// 接线一旦被后续重构挪掉，编译与单测都不会红，只有真机才看得出来，故在此按源码守一次
func readSource(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s failed: %v", path, err)
	}
	return string(content)
}

// functionBody 取某个函数从签名到下一个顶层声明之间的正文
func functionBody(t *testing.T, source, signature string) string {
	t.Helper()
	start := strings.Index(source, signature)
	if start < 0 {
		t.Fatalf("signature %q not found", signature)
	}
	rest := source[start+len(signature):]
	end := strings.Index(rest, "\nfunc ")
	if end < 0 {
		return rest
	}
	return rest[:end]
}

// TestCmdRuleBaselineStaysWired 机器命令必须仍在策略之外兜住命令过滤规则。
//
// 「命令过滤规则」早于触发策略存在，命中即拒绝；接进策略后它变成一个可配置的检查项，
// 于是「未绑定流程定义」与「绑了但没勾这条」两种情况都可能让规则静默失效。
// 基线判定必须留在三个入口共用的 CheckMachineCmd 里，而不是某个入口自己实现一遍
func TestCmdRuleBaselineStaysWired(t *testing.T) {
	source := readSource(t, "machine_trigger.go")

	checkCmd := functionBody(t, source, "func CheckMachineCmd(")
	for _, call := range []string{"rejectUngovernedCmdRule(", "GetProcdefByCodePath(", "CheckProcdefTrigger("} {
		if !strings.Contains(checkCmd, call) {
			t.Fatalf("CheckMachineCmd must still call %s, got:\n%s", call, checkCmd)
		}
	}

	baseline := functionBody(t, source, "func rejectUngovernedCmdRule(")
	// 已被策略纳管则以配置为准（管理员可以降级），未纳管才兜底拦截
	for _, fact := range []string{"CheckConfigured(", "MatchCmdRule(", "imsg.TerminalCmdDisable"} {
		if !strings.Contains(baseline, fact) {
			t.Fatalf("the cmd-rule baseline must keep %s, got:\n%s", fact, baseline)
		}
	}

	// 三个执行入口共用同一前置校验，不允许再各自实现一遍
	for _, file := range []string{"machine_term_op.go", "machine.go"} {
		if strings.Contains(readSource(t, file), "GetCmdConfsByMachineTags") {
			t.Fatalf("%s must not consult cmd filter rules on its own any more", file)
		}
	}
}

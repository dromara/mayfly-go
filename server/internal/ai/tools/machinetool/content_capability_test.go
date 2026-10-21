package machinetool

import (
	"testing"

	"mayfly-go/internal/ai/tools"

	"github.com/stretchr/testify/require"
)

// TestMachineCommandExecContentCapability 验证 MachineCommandExec 的内容级能力判定：
// 只读白名单命令在只读诊断档算 Read（放行），变更/复合/命令替换一律算 Dangerous（拦）。
//
// 这是「精确命令解析」的回归防线——用户强调不能只看命令开头，必须覆盖管道、分号多命令、
// 命令替换等组合。判定复用 mcm.CmdAnalyzer（与工具体审批白名单同源），此处锁住接线正确性
func TestMachineCommandExecContentCapability(t *testing.T) {
	capOf := func(cmd string) tools.Capability {
		return tools.CapabilityForArgs(tools.ToolNameMachineCommandExec, `{"command":`+jsonStr(cmd)+`,"authCertName":"x","remark":"r"}`)
	}

	// —— 只读：单命令、管道、绝对路径都应放行（Read）——
	for _, cmd := range []string{
		"top -bn1",
		"ps aux",
		"df -h",
		"ps aux | grep nginx", // 管道，两段都在白名单
		"/usr/bin/free -m",    // 绝对路径去前缀后仍识别为 free
		"uptime && ls -la",
		"cat /var/log/messages", // 读文件内容属只读（诊断看日志刚需），保留免审批
		"tail -n 100 app.log",
		"netstat -tlnp", // && 连接两个只读命令
	} {
		require.Equal(t, tools.CapabilityRead, capOf(cmd), "只读命令应判 Read: %s", cmd)
	}

	// —— 变更/逃逸：必须判 Dangerous，绝不能被只读前缀骗过 ——
	for _, cmd := range []string{
		"rm -rf /",
		"top; rm -rf /",           // 分号：第二段危险 → 整体 Dangerous
		"ps aux | sh",             // 管道：sh 不在白名单 → Dangerous
		"cat $(echo /etc/shadow)", // 命令替换无法静态分析 → Dangerous
		"cat `id`",                // 反引号命令替换 → Dangerous
		"ls > /tmp/out.txt",       // 重定向写文件 → Dangerous
		"echo hi | tee /etc/x",    // tee 写文件且 tee 不在白名单 → Dangerous
		"uptime && reboot",        // && 第二段危险 → Dangerous
		"find / -name x -delete",  // find 有逃逸能力，不在白名单 → Dangerous
		"curl http://100.86.0.1",  // 外联：已从只读白名单移出
		"wget http://x/f.tar",     // 下载写盘：已移出
		"sort -o /tmp/x f",        // -o 写文件且不含重定向符：已移出
		"ip link set eth0 down",   // 可改网络配置：已移出
		"ping -c1 1.1.1.1",        // 对外网络交互：已移出
		"yum list",                // 联网查仓库：已移出
	} {
		require.Equal(t, tools.CapabilityDangerous, capOf(cmd), "危险/复合命令应判 Dangerous: %s", cmd)
	}
}

// TestMachineCommandExecContentCapabilityFailClosed 入参异常时不得放行（fail-closed）
func TestMachineCommandExecContentCapabilityFailClosed(t *testing.T) {
	// 非法 JSON、缺 command、空 command 全部按 Dangerous，绝不因解析不了就当作只读放行
	for _, args := range []string{"not-json", `{}`, `{"command":""}`, `{"authCertName":"x"}`} {
		require.Equal(t, tools.CapabilityDangerous,
			tools.CapabilityForArgs(tools.ToolNameMachineCommandExec, args), "异常入参须 fail-closed: %s", args)
	}
}

// jsonStr 把命令包成 JSON 字符串字面量（转义双引号/反斜杠，供拼进 args JSON）
func jsonStr(s string) string {
	out := []byte{'"'}
	for _, ch := range s {
		switch ch {
		case '"':
			out = append(out, '\\', '"')
		case '\\':
			out = append(out, '\\', '\\')
		default:
			out = append(out, string(ch)...)
		}
	}
	return string(append(out, '"'))
}

package tools

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// stubResolver 临时注册凭证解析器，测试结束恢复为「什么都解析不出来」的空实现
func stubResolver(t *testing.T, resolve func(input string) (string, bool)) {
	t.Helper()
	DeclareAuthCertResolver(func(_ context.Context, input string) (string, bool) { return resolve(input) })
	t.Cleanup(func() {
		DeclareAuthCertResolver(func(context.Context, string) (string, bool) { return "", false })
	})
}

// TestNormalizeActionParamsReplacesMachineNameWithSoleCert 唯一凭证时把机器名规范化为真实凭证名。
//
// 端到端实测到模型把 "aliyun"（机器名）写进 authCertName：键合法、执行时也会被兜底纠正，但
// 提案文本给审批人看、且放行要求批准参数与执行实收参数逐项相等，不规范化就让审计对不上
func TestNormalizeActionParamsReplacesMachineNameWithSoleCert(t *testing.T) {
	stubResolver(t, func(input string) (string, bool) {
		if input == "aliyun" {
			return "tVUcRFKwOC", true
		}
		return "", false
	})

	out := NormalizeActionParams(context.Background(), "MachineCommandExec",
		`{"authCertName":"aliyun","command":"true","remark":"探活"}`)

	require.Contains(t, out, "tVUcRFKwOC", "凭证名应被规范化")
	require.NotContains(t, out, "aliyun", "机器名不应残留")
	require.Contains(t, out, `"command":"true"`, "其余参数原样保留")
	require.Contains(t, out, `"remark":"探活"`, "其余参数原样保留")
}

// TestNormalizeActionParamsKeepsOriginalWhenUndecidable 任何不确定情形一律原样返回：
// 宁可不规范化，也不能凭空改坏审批人已据以判断的参数。
func TestNormalizeActionParamsKeepsOriginalWhenUndecidable(t *testing.T) {
	cases := []struct {
		name   string
		action string
		params string
	}{
		{"非机器动作不处理", "ExecSql", `{"authCertName":"aliyun","sql":"select 1","remark":"r"}`},
		{"没有该键不处理", "MachineCommandExec", `{"command":"true","remark":"r"}`},
		{"脏 JSON 不处理", "MachineCommandExec", `not-json`},
		{"空参数不处理", "MachineCommandExec", ``},
		{"非对象 JSON 不处理", "MachineCommandExec", `[1,2]`},
		{"键值不是字符串不处理", "MachineCommandExec", `{"authCertName":123,"command":"true","remark":"r"}`},
	}

	// 解析器对一切输入都解不出（多凭证不猜 / 未注册都走这条）
	stubResolver(t, func(string) (string, bool) { return "", false })
	for _, tc := range cases {
		require.Equal(t, tc.params, NormalizeActionParams(context.Background(), tc.action, tc.params), tc.name)
	}

	// 动作名合法但机器匹配不到：同样原样
	require.Equal(t, `{"authCertName":"不存在的机器","command":"true"}`,
		NormalizeActionParams(context.Background(), "MachineCommandExec", `{"authCertName":"不存在的机器","command":"true"}`))
}

// TestNormalizeActionParamsKeepsAlreadyValidCert 已是真实凭证名时不做无意义的重写。
//
// 这条同时挡住一类隐患：解析器被误注册成「恒返回同一个值」时，仍不会因为 cert==raw 而改写参数
func TestNormalizeActionParamsKeepsAlreadyValidCert(t *testing.T) {
	stubResolver(t, func(string) (string, bool) { return "tVUcRFKwOC", true })

	params := `{"authCertName":"tVUcRFKwOC","command":"true","remark":"r"}`
	require.Equal(t, params, NormalizeActionParams(context.Background(), "MachineCommandExec", params))
}

// TestResolveAuthCertNameDefaultsToNoGuess 未注册解析器时默认实现必须返回「解析不出」而不是 panic。
//
// 默认空实现的存在意义就是把「漏注册」变成「不猜」，而不是散落在各调用点的 nil 检查
func TestResolveAuthCertNameDefaultsToNoGuess(t *testing.T) {
	DeclareAuthCertResolver(nil) // nil 输入应被忽略，不覆盖已有注册

	cert, ok := ResolveAuthCertName(context.Background(), "aliyun")
	require.False(t, ok, "无可用解析结果时不得给出凭证")
	require.Empty(t, cert)
}

// TestProposeRemediationNormalizesBeforeRecord 提案登记前必须走参数规范化。
//
// 只测函数本身拦不住调用点退化（同类变异存活已实测发生过：改了 GetHistory 的调用方式，
// 新函数的断言照样全绿）。而装配整条工具执行需要 recorder 与流程桩，成本高于收益，
// 这里用静态断言钉住唯一入口
func TestProposeRemediationNormalizesBeforeRecord(t *testing.T) {
	src, err := os.ReadFile("propose_remediation.go")
	require.NoError(t, err, "读取 propose_remediation.go 失败")
	require.Contains(t, string(src), "Params: NormalizeActionParams(ctx, param.Action, param.Params),",
		"提案登记前必须规范化参数，不能把模型写的机器名原样带进审批单")
}

// TestDefaultAuthCertResolverIsOverridable 空实现可被工具侧注册覆盖（注册机制本身可用）。
//
// 注：「内置机器动作的入参形状含 authCertName」这条断言不放在本包——声明由 machinetool 的
// 包初始化完成，tools 包自己的测试不会触发那个 import，在这里断言只会得到空表。
// 它已落在 agentops application 的守卫用例里（那里真实依赖工具包）
func TestDefaultAuthCertResolverIsOverridable(t *testing.T) {
	DeclareAuthCertResolver(nil) // nil 输入应被忽略，不覆盖已有注册
	_, ok := ResolveAuthCertName(context.Background(), "aliyun")
	require.False(t, ok)

	stubResolver(t, func(string) (string, bool) { return "cert-by-name", true })
	cert, ok := ResolveAuthCertName(context.Background(), "aliyun")
	require.True(t, ok, "注册后应可解析")
	require.Equal(t, "cert-by-name", cert)
}

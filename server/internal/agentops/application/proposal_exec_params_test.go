package application

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"mayfly-go/internal/ai/tools"
	// 依赖两个内置工具包：它们的包初始化把真实的入参形状声明写入注册表，本文件不得另造副本
	"context"
	_ "mayfly-go/internal/ai/tools/dbtool"
	_ "mayfly-go/internal/ai/tools/machinetool"
)

// readProposalGatewaySource 读取提案网关源码（接线断言用）
func readProposalGatewaySource(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("proposal_gateway.go")
	require.NoError(t, err)
	return string(b)
}

// TestAllowedToolsRejectsMismatchedProposalParams 提案参数与工具入参不同形时，
// 必须拒绝下发受控执行许可，而不是批准后静默什么都执行不了。
//
// 真实数据形态：库里一条提案的 params 是 {"machineId":1,"cmd":"true"}，而
// MachineCommandExec 收的是 authCertName/command/remark。不拦的后果是：提案能登记、
// 能被批准、run 被记 succeeded、提案被记 executed，而实际一次写操作都没发生且无报错。
func TestAllowedToolsRejectsMismatchedProposalParams(t *testing.T) {
	// 不在此重新声明形状：machinetool 的包初始化已把真声明写入注册表（本文件已 blank-import 它）。
	// 手写一份副本会让「工具改了 json tag」与「校验用的属性集」脱钩——测试照样绿而线上失真
	allowed, err := allowedToolsForAction(context.Background(), "MachineCommandExec", `{"machineId":1,"cmd":"true"}`)
	require.Error(t, err, "参数不同形的提案不得下发受控执行许可")
	require.Nil(t, allowed)
	require.Contains(t, err.Error(), "unknown parameter")

	// 形状正确时正常下发「只允许这一个工具」的许可
	allowed, err = allowedToolsForAction(context.Background(), "MachineCommandExec", `{"authCertName":"c1","command":"true","remark":"探活"}`)
	require.NoError(t, err)
	require.Equal(t, []string{"MachineCommandExec"}, allowed)
}

// TestAllowedToolsRejectsReadOnlyActions 只读工具不能作为提案 action。
//
// 旧用例拿 DbQueryData（能力表里的只读工具）断言「可以下发许可」，等于把缺陷钉成期望值，已推翻。
// 不拦的后果：只读工具作为 action 时受控执行那轮不改动任何状态，提案却被记成 executed，
// 留下一条「已执行但什么都没发生」的审计事实（比记成失败更误导）；而 propose_remediation 在能力表里
// 同为只读，因此一并拦住——否则一条已批准的提案可以直接登记出另一条提案（审批链被「提案造提案」稀释）。
// 判据取能力表而不是工具名名单：新增只读工具自动被拦，本函数零改动。
func TestAllowedToolsRejectsReadOnlyActions(t *testing.T) {
	for _, readOnlyTool := range []string{"DbQueryData", "ListResources", "QueryMetrics", "propose_remediation"} {
		allowed, err := allowedToolsForAction(context.Background(), readOnlyTool, `{"anything":1}`)
		require.Errorf(t, err, "只读工具 %s 不能作为提案 action", readOnlyTool)
		require.Nil(t, allowed)
	}

	// 登记侧同一判据（两处强度不一致就会出现「存得进、批得了、跑不了」）
	require.Error(t, validateProposalAction(context.Background(), "DbQueryData", `{"anything":1}`),
		"登记侧必须同样拦住只读 action")
	// 可变工具不受该闸门影响（参数形状另断于 TestAllowedToolsRejectsMismatchedProposalParams）
	require.NoError(t, validateProposalAction(context.Background(), "MachineCommandExec",
		`{"authCertName":"c1","command":"true","remark":"探活"}`))
}

// TestProposalActionGateIsSingleSource action 准入判据必须只有一个定义点。
//
// Record 与 allowedToolsForAction 各自写一遍时，一侧收紧另一侧会静默落后；本断言把
// 「两侧都调同一函数」钉住（源码接线守卫，与 wiring_guard_test 同一形态）
func TestProposalActionGateIsSingleSource(t *testing.T) {
	source := readProposalGatewaySource(t)
	// 能力闸门与形状校验都只应出现在 validateProposalAction 里，不得在调用方重复实装。
	// 锚点取完整代码表达式而不是裸词 CapabilityRead：后者会连注释里的叙述一起数进去（实测如此）
	require.Equal(t, 1, strings.Count(source, "aitools.CapabilityOf(name) == aitools.CapabilityRead"),
		"只读 action 的判定只能有一处，否则登记侧与下发侧会强度漂移")
	require.Equal(t, 1, strings.Count(source, "aitools.ValidateToolParams("),
		"参数形状校验必须收在单一判据里，不得两处各调一遍")
	require.Equal(t, 2, strings.Count(source, "validateProposalAction(ctx,"),
		"Record 与 allowedToolsForAction 必须都经该判据（带逗号的调用形态，不把函数声明自身数进去）")
}

// TestBuiltinToolSchemasReadyAtPackageLoad 内置工具的入参形状必须在**包加载时**就绪。
//
// 曾经把声明写在 GetCommandExec() 里：下发受控执行许可（allowedToolsForAction 的参数校验）
// 早于工具首次构造，那时 schema 表为空 → 校验走「未声明则跳过」分支静默放行。
// 端到端实测到坏参数 {"machineId":1,"cmd":"true"} 一路漏到执行期，模型自己猜 authCertName
// 猜错（ac not found）后退到不传 → 挂在参数补全中断上，白烧一次运行
func TestBuiltinToolSchemasReadyAtPackageLoad(t *testing.T) {
	for _, name := range []string{"MachineCommandExec", "ExecSql"} {
		schema, ok := tools.ParamSchemaOf(name)
		require.True(t, ok, "%s 的入参形状应在包加载后就绪，不能等工具首次构造", name)
		require.NotEmpty(t, schema.Props)
	}

	// 提案参数规范化写回的键名必须落在工具声明的属性集内：两处各持一份 "authCertName" 字面量
	// （一份由反射 json tag 得出，一份是 tools.NormalizeActionParams 写回用的常量），
	// 工具侧改了 tag 而那边没跟上，规范化就会写出一个非法键并被登记校验拒掉——这条把两者锁在一起
	machineSchema, ok := tools.ParamSchemaOf("MachineCommandExec")
	require.True(t, ok)
	require.Contains(t, machineSchema.Props, "authCertName")
	require.Contains(t, machineSchema.Required, "remark", "remark 是必填：它是审批人理解执行目的的唯一来源")

	// 端到端实测到的真实坏参数：必须在下发许可前就被拒
	_, err := allowedToolsForAction(context.Background(), "MachineCommandExec", `{"machineId":1,"cmd":"true"}`)
	require.Error(t, err, "真实坏参数必须被拦在许可下发前")
	require.Contains(t, err.Error(), "unknown parameter")
	require.Contains(t, err.Error(), "authCertName", "报错要列出合法属性名，否则模型/运维无从自纠")
}

// TestAllowedToolsStillRejectsNonToolActions 原有的「action 必须是可调用工具名」守卫不受影响。
func TestAllowedToolsStillRejectsNonToolActions(t *testing.T) {
	_, err := allowedToolsForAction(context.Background(), "重启一下数据库", `{}`)
	require.Error(t, err, "语义动作不是工具名，拒绝下发许可")

	_, err = allowedToolsForAction(context.Background(), "  ", `{}`)
	require.Error(t, err, "空 action 拒绝")
}

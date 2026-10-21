package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// optionalParamShape 复刻 dbtool.SQLExecParam 的形状（四个字段都没有 omitempty，其中两个无
// required 标记）：工具执行时把参数指针 marshal 成 JSON，**必然输出全部四个键**（可选字段以零值出现）。
// 不复用 dbtool 类型是避免 tools ← dbtool 反向依赖成环。
//
// 关键约束：它只登记给**探针工具名**，绝不能登记到真实工具名上——本包测试与真实注册表共享进程，
// 用副本覆盖真声明会让「高危工具必须声明必填属性」的守卫看到空 Required 而误判，更会让真实工具的
// 校验口径在测试里被悄悄改掉（实测踩过两次）
type optionalParamShape struct {
	DbId   int64  `json:"dbId"`
	DbName string `json:"dbName"`
	SQL    string `json:"sql"`
	Remark string `json:"remark"`
}

// TestApprovedForAlignsWithValidateToolParamsOnOptionalParams 参数形状校验与放行判定必须同一口径。
//
// 审查实测到的矛盾：ValidateToolParams 允许省略可选属性（只拦未知键与缺必填），于是
// 「登记成功、被批准」的提案可以只有 3 个键；而执行那轮工具实收参数是 marshal 出来的 4 个键，
// ApprovedFor 原先按「键数必须相等」判定 → 两侧永不相等 → 不放行 → 挂在审批上 → 运行失败 →
// 提案被记 failed。表现是「批了却必然跑不了」，且日志里没有任何线索指向键数不匹配。
func TestApprovedForAlignsWithValidateToolParamsOnOptionalParams(t *testing.T) {
	// 探针工具名：不碰 ExecSql 的真实声明
	toolName := "ProbeOptionalParamTool"
	declareProbeSchema(t, toolName, optionalParamShape{})

	// 省略可选 dbName 的提案：这是登记校验允许通过的形状
	partialApproved := `{"dbId":5,"sql":"select 1","remark":"核对行数"}`
	// 登记与放行是两道独立关卡：登记现在要求参数完整，放行判据不得依赖该前提——
	// 存量提案或批准文本来自其它入口时，省略可选属性的批准仍须能与零值实收参数对齐
	require.Error(t, ValidateToolParams(toolName, partialApproved),
		"登记侧已收紧为要求完整参数")

	// 执行那轮工具实收参数（marshal 出全部四个键）
	var args any = &optionalParamShape{DbId: 5, SQL: "select 1", Remark: "核对行数"}

	grant := &RunGrant{ApprovedTools: map[string]string{toolName: partialApproved}}
	require.True(t, grant.ApprovedFor(toolName, args),
		"批准的就是这些参数、执行时也没被改动，必须放行（否则提案批准后永远执行不了）")

	// 安全底线不能被这个放宽破坏：执行时新增了批准里没有的属性（如夹带别的库），仍须拒绝
	grantWithExtra := &RunGrant{ApprovedTools: map[string]string{toolName: partialApproved}}
	argsSwitchedDb := &optionalParamShape{DbId: 5, DbName: "mysql", SQL: "select 1", Remark: "核对行数"}
	require.False(t, grantWithExtra.ApprovedFor(toolName, argsSwitchedDb),
		"批准时未指定 dbName、执行时却指定了：属于批准范围之外的改动，不能放行")

	// 同理：批准时指定了 dbName，执行时不能悄悄丢掉它
	fullApproved := `{"dbId":5,"dbName":"mayfly_go","sql":"select 1","remark":"r"}`
	grantFull := &RunGrant{ApprovedTools: map[string]string{toolName: fullApproved}}
	require.False(t, grantFull.ApprovedFor(toolName, &optionalParamShape{DbId: 5, SQL: "select 1", Remark: "r"}),
		"批准里明确带的 dbName 在执行时被丢掉，同样是改动批准内容，不能放行")
}

// TestValidateToolParamsRequiresEveryParam 收紧：提案参数必须给出该工具的**全部**入参属性。
//
// 为什么要求全而不是只要求必填：放行走的是「批准参数与执行实收参数逐项相等」，而执行实收参数
// 由工具参数 struct marshal 得出、必然含全部属性（可选项以零值出现）。登记侧若允许省略，两侧
// 形状就永久不同构，出现「批了但必然不放行」。要求全属性同时让审批人看到完整执行参数
func TestValidateToolParamsRequiresEveryParam(t *testing.T) {
	declareProbeSchema(t, "ProbeFullTool", optionalParamShape{})

	// 缺可选属性：以前放行，现在必须拒绝
	err := ValidateToolParams("ProbeFullTool", `{"dbId":5,"sql":"select 1","remark":"r"}`)
	require.Error(t, err, "缺任何入参属性都应拒绝")
	require.Contains(t, err.Error(), "dbName", "报错必须点名缺哪个属性，模型才能补齐")

	// 全属性齐备才通过（含显式空串的可选属性）
	require.NoError(t, ValidateToolParams("ProbeFullTool",
		`{"dbId":5,"dbName":"","sql":"select 1","remark":"r"}`))
	require.NoError(t, ValidateToolParams("ProbeFullTool",
		`{"dbId":5,"dbName":"mayfly_go","sql":"select 1","remark":"r"}`))

	// 未声明形状的工具仍然跳过（MCP/插件不猜它的参数）
	require.NoError(t, ValidateToolParams("SomeMcpTool", `{"anything":1}`))

	// 空 ctx 无关：只校验形状
	_ = context.Background()
}

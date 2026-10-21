package tools

import "strings"

// 跨层协议字符串的唯一真源。
//
// 这些字面量同时被「工具侧」与「agent / event_mapper 侧」读写，分头硬编码会在只改一边时
// 静默错配：不报错、不 panic，只表现为「用户拒了命令却被计入工具执行失败」「取消被当成系统
// 故障弹红色提示」。工具名还同时是能力表 key、许可清单元素与放行判据的 map key。
const (
	// ToolNameMachineCommandExec 机器命令执行工具名
	ToolNameMachineCommandExec = "MachineCommandExec"
	// ToolNameExecSql 数据库 SQL 执行工具名
	ToolNameExecSql = "ExecSql"

	// CancelledToolPrefix 用户取消参数补全时回给模型的错误前缀
	CancelledToolPrefix = "[PARAM_COMPLETION_CANCELLED]"
	// RejectedToolPrefix 用户拒绝审批时回给模型的错误前缀
	RejectedToolPrefix = "[OPERATION_REJECTED]"
)

// IsCancelledToolResult 该工具结果是否表示「用户取消了参数补全」
func IsCancelledToolResult(content string) bool {
	return strings.Contains(content, CancelledToolPrefix)
}

// IsRejectedToolResult 该工具结果是否表示「用户拒绝了审批」
func IsRejectedToolResult(content string) bool {
	return strings.Contains(content, RejectedToolPrefix)
}

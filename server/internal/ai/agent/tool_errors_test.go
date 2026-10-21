package agent

import (
	"testing"

	"mayfly-go/internal/ai/session"
	"mayfly-go/internal/ai/tools"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/require"
)

func toolMsg(content string, toolName string) *session.Message {
	m := &session.Message{Role: schema.Tool, Content: content, ToolName: toolName}
	SetToolStatus(m, tools.ToolStatusError)
	return m
}

// collectToolErrors 必须把「工具执行失败」从一轮消息里挑出来：
// run 终态是 succeeded 时，这是下游区分「干成了」与「没干成」的唯一事实来源
func TestCollectToolErrors(t *testing.T) {
	okMsg := &session.Message{Role: schema.Tool, Content: `{"rows":[{"c":3}]}`, ToolName: "DbQueryData"}
	failMsg := &session.Message{Role: schema.Tool, Content: tools.GetToolErrorMsg(errFake("access denied")), ToolName: "DbQueryData"}

	// 仅内容形态（SafeToolMiddleware 把 RecoverRetry 错误转成字符串结果，无 toolStatus 标记）
	errs := collectToolErrors([]*session.Message{okMsg, failMsg})
	require.Len(t, errs, 1)
	require.Contains(t, errs[0], "DbQueryData")
	require.Contains(t, errs[0], "access denied")

	// 结构化 toolStatus=error 形态同样计入
	errs = collectToolErrors([]*session.Message{toolMsg("boom", "ExecSql")})
	require.Len(t, errs, 1)
	require.Contains(t, errs[0], "ExecSql")

	// 用户拒绝与等待人工交互都不算执行失败。
	//
	// 内容必须用运行时真实形态：拒绝错误经 NewToolError(RecoverRetry) → SafeToolMiddleware 转字符串时
	// 带 "[tool error] " 前缀，即 "[tool error] [OPERATION_REJECTED] ..."。用裸 "[OPERATION_REJECTED]..."
	// 作夹具会让 HasPrefix 判据也恰好命中，掩盖「拒绝被误计为执行失败」这个真缺陷（假覆盖）
	rejected := &session.Message{
		Role:     schema.Tool,
		Content:  tools.GetToolErrorMsg(errFake(tools.RejectedToolPrefix + " The tool 'ExecSql' was explicitly rejected by the user.")),
		ToolName: "ExecSql",
	}
	SetToolStatus(rejected, tools.ToolStatusError)
	require.Empty(t, collectToolErrors([]*session.Message{rejected}), "带 [tool error] 前缀的拒绝内容不得计入执行失败")

	interrupted := &session.Message{Role: schema.Tool, Content: tools.GetToolErrorMsg(errFake("waiting")), ToolName: "ExecSql"}
	SetToolStatus(interrupted, tools.ToolStatusInterrupted)
	require.Empty(t, collectToolErrors([]*session.Message{interrupted}))

	// 非工具消息不参与判定
	require.Empty(t, collectToolErrors([]*session.Message{{Role: schema.Assistant, Content: "执行失败了"}}))
	require.Empty(t, collectToolErrors(nil))
}

// 单条错误过长要截断（工具错误可能带回显 SQL/堆栈，会污染运行行与提案 err 列）
func TestCollectToolErrorsTruncates(t *testing.T) {
	long := tools.GetToolErrorMsg(errFake(string(make([]byte, 900))))
	errs := collectToolErrors([]*session.Message{{Role: schema.Tool, Content: long, ToolName: "ExecSql"}})
	require.Len(t, errs, 1)
	require.Less(t, len([]rune(errs[0])), 600)
	require.Contains(t, errs[0], "截断")
}

// 「调用成功但载荷自报失败」必须计入：这是命令退出码非零的唯一痕迹。
//
// 真机事故形态：命令 false 退出非零，工具体为了让模型看到输出仍返回正常结果，只在结构里写
// success:false；不收这一条，下游就把变更判成已生效（提案 executed、工单处理成功）。
//
// 用探针工具名而不是真实工具名：跨包注册表无法清理，靠进程隔离兜住，避免与生产注册互相干扰
func TestCollectToolErrorsFromDeclaredJudge(t *testing.T) {
	const probeTool = "probe-self-reported-failure"
	tools.DeclareResultJudge(probeTool, func(output string) (bool, string) {
		if output == `{"success":false}` {
			return false, "command failed: exit status 1"
		}
		return true, ""
	})

	// 状态为 success、内容也不是 tool error —— 只会被 judge 捞到
	msg := &session.Message{Role: schema.Tool, Content: `{"success":false}`, ToolName: probeTool}
	errs := collectToolErrors([]*session.Message{msg})
	require.Len(t, errs, 1, "载荷自报失败必须计入，否则执行失败会被当成变更已生效")
	require.Contains(t, errs[0], probeTool)
	require.Contains(t, errs[0], "command failed")

	// 同工具成功载荷不计入
	require.Empty(t, collectToolErrors([]*session.Message{{Role: schema.Tool, Content: `{"success":true}`, ToolName: probeTool}}))

	// 未登记判定的工具：保持旧行为（不猜它的输出形状）
	require.Empty(t, collectToolErrors([]*session.Message{{Role: schema.Tool, Content: `{"success":false}`, ToolName: "probe-no-judge"}}))
}

type errFake string

func (e errFake) Error() string { return string(e) }

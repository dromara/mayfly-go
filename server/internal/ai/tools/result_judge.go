package tools

import "sync"

// ResultJudge 工具结果的**内容级**判定：调用本身成功（无 error）但载荷自报失败时的识别入口。
//
// 为什么需要这层：有些工具刻意在失败时仍返回正常结果 —— 远程命令退出码非零时，工具体为了让
// 模型看到输出而 `return output, nil`，失败事实只写在载荷的 `success:false` 里。这类结果进不了
// ToolErrors（它不是 tool error），下游「变更到底有没有发生」就会判成功：实测到命令 `false`
// 执行失败、提案落 executed、工单落「处理成功」，而执行报告里明写着「❌ 失败」。
//
// 形态与 DeclareCapability / DeclareParamSchema / DeclareContentCapability 一致：
// 工具在自己的包 init 期声明「我的输出长什么样、什么算失败」，消费方（agent.collectToolErrors）
// 保持通用——新增工具只在自己的包里加一行注册，不需要改消费方。
type ResultJudge func(output string) (ok bool, reason string)

var (
	resultJudgeMu sync.RWMutex
	resultJudges  = map[string]ResultJudge{}
)

// DeclareResultJudge 声明某工具的输出判定。
//
// 必须在包 init 期调用：判定发生在运行期收集工具消息时，但注册时机与「入参形状声明」同理，
// 越早越稳（首次工具构造之前就可能需要它）。
func DeclareResultJudge(toolName string, judge ResultJudge) {
	resultJudgeMu.Lock()
	defer resultJudgeMu.Unlock()
	resultJudges[toolName] = judge
}

// JudgeToolResult 判定一次工具输出是否真的成功。
//
// 未登记判定的工具一律返回 (true, "")：不猜它的输出形状（与「未声明参数形状则跳过校验」
// 同一保守口径）。返回 false 时 reason 是人类可读的失败原因，会被拼进 ToolErrors 供提案侧判失败。
func JudgeToolResult(toolName, output string) (ok bool, reason string) {
	resultJudgeMu.RLock()
	judge := resultJudges[toolName]
	resultJudgeMu.RUnlock()
	if judge == nil {
		return true, ""
	}
	return judge(output)
}

// HasResultJudge 是否已为该工具登记输出判定（守卫用：防止 init 漏注册让判定静默失效）
func HasResultJudge(toolName string) bool {
	resultJudgeMu.RLock()
	defer resultJudgeMu.RUnlock()
	_, ok := resultJudges[toolName]
	return ok
}

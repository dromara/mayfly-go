package tools

import "context"

// 无人值守运行标记。
//
// 存在动机：Agent 有两类「等人类回答」的接缝——工具审批中断与参数补全中断。它们在有前端连接的
// 对话里是对的（卡片弹出来、用户填完续跑），但在无头运行里**没有任何应答者**：RunHeadless 的
// publisher 传 nil，只收集落库而不推送，卡片发不出去也没人看。
//
// 参数补全那一类此前没有这个判定，真机后果是：一次事件触发的诊断运行里模型提交了
// `MachineCommandExec{authCertName:""}`（只读命令，护栏按入参放行），工具体随即发起补全中断，
// 于是整轮停在 interrupted、运行被记成 failed，摘要就是那句「缺少机器信息，请完善参数」——
// 7139 tokens 花完，一条诊断结论都没产出。而 ListResources 那一步其实已经把凭证名返回给模型了，
// 只差一次「你漏了参数，可选值是这些」的回传。
//
// 判定放在运行上下文而不是按 profile/工具名推断：唯一可靠的依据是「这次运行有没有连接着的前端」，
// 由 RunHeadless 单点写入（见 ai/application/agent_runner.go），与 RunGrant 同一注入形态。
//
// 注意：审批中断**不**因该标记而放开——无头运行不得做变更是安全底线（批准过的变更另有
// RunGrant.ApprovedTools 通道），这里只改变「等人填参数」这一件事的应答方式。
type unattendedKey struct{}

// WithUnattended 标记本次运行无人值守（无人能回答中断卡片）
func WithUnattended(ctx context.Context) context.Context {
	return context.WithValue(ctx, unattendedKey{}, true)
}

// IsUnattended 本次运行是否无人值守。缺省 false：默认按「有人应答」走中断卡片，
// 宁可让对话路径维持原语义，也不把等待人类的路径静默改掉
func IsUnattended(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	unattended, _ := ctx.Value(unattendedKey{}).(bool)
	return unattended
}

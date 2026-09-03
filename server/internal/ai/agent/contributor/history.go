package contributor

import (
	"context"

	"github.com/cloudwego/eino/adk"
)

// HistoryBuildContext 历史贡献上下文（对齐 tokhub HistoryBuildContext）
//
// 仅携带 HistoryContributor 实际消费的信息；新增字段须以真实消费方为前提。
type HistoryBuildContext struct {
	// SessionKey 会话标识（历史消息的加载入口）
	SessionKey string
}

// HistoryContributor 对话历史消息段贡献者
//
// Extension 通过实现此接口向对话历史**贡献消息段**，与 ContextContributor
// （prompt 片段注入）职责分离：
//   - 片段进入 preamble（system 消息），无协议不变式约束
//   - 消息段进入 messages 数组，受 tool_call/tool_result 配对等协议约束
//
// ## 合并语义（Registry dispatch 负责）
//
// 多个贡献者按注册顺序各自返回消息段，Registry 拼接后**统一做一次
// NormalizeHistory**（配对修复必须对最终全量列表执行，对齐 tokhub
// build_histories dispatch）。典型贡献内容：
//   - 会话历史扩展：压缩摘要（[之前的对话摘要] 前缀）+ 未摘要的近期消息
//   - 未来扩展：跨会话召回段、钉选消息段等
//
// ## 无状态约束
//
// 实现者不应依赖进程内状态：当轮所需的历史状态由实现者按 SessionKey
// 自行加载（域内查询），页面刷新、服务重启、跨实例处理均不受影响。
type HistoryContributor interface {
	Contributor
	// ContributeMessages 贡献对话历史消息段（非全量构建）
	//
	// 返回空切片表示本扩展本轮不贡献任何消息。
	// 实现者不需要自行调用 NormalizeHistory——配对修复由 Registry
	// 对合并后的全量列表统一执行。
	ContributeMessages(ctx context.Context, bc *HistoryBuildContext) ([]adk.Message, error)
}

// MidTurnCompactionParams mid-turn 紧急压缩参数（token 口径由宿主估算后传入）
//
// 宿主用与预算追踪同一口径的计数器估算 preamble / history 占用，
// 贡献者据此判断是否触发紧急压缩。
type MidTurnCompactionParams struct {
	// SessionKey 会话标识（软阈值触发后台增量摘要的入口）
	SessionKey string
	// ContextWindow 上下文窗口大小（token 数）
	ContextWindow int
	// PreambleTokens preamble 占用 token 数（宿主估算）
	PreambleTokens int
	// HistoryTokens 当前合并后 history 占用 token 数（宿主估算）
	HistoryTokens int
}

// MidTurnCompactionInfo mid-turn 压缩结果（宿主据此记录压缩事件）
type MidTurnCompactionInfo struct {
	// OriginalTokens 压缩前估算 token 数
	OriginalTokens int
	// CompressedTokens 压缩后估算 token 数
	CompressedTokens int
}

// MidTurnCompactor mid-turn 紧急压缩可选能力（HistoryContributor 的扩展接口）
//
// Go 无 trait 默认实现，Registry dispatch 对 HistoryContributor 做类型
// 断言：实现本接口的贡献者才参与 mid-turn 压缩竞争。
//
// 宿主在 preamble 与合并后 history 均就绪后调用（短路通道，按注册逆序
// 遍历：首个返回压缩结果的贡献者视为实际执行了压缩，后注册的插件扩展
// 因此可覆盖内置压缩策略）。
type MidTurnCompactor interface {
	// TryMidTurnCompaction 尝试就地压缩历史
	//
	// 未触发阈值时返回原切片与 nil；触发时返回压缩后的消息列表与压缩信息。
	TryMidTurnCompaction(ctx context.Context, history []adk.Message, params *MidTurnCompactionParams) ([]adk.Message, *MidTurnCompactionInfo)
}

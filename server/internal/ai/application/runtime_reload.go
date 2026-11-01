package application

import (
	"mayfly-go/internal/ai/agent"
	"mayfly-go/pkg/logx"
)

// ReloadModelConfig 模型配置（系统配置 AiModelConfig）热生效入口
//
// 由宿主在配置变更钩子中调用（internal/ai/init）。模型地址 / Key / 模型名 / 超时 /
// 温度 / 最大 token / 思考开关 / 上下文窗口 / 重试与故障转移都在装配时读取，
// 这里把两层缓存一起失效：
//
//   - 模型运行时 + 默认 Agent 单例（agent.ResetModelRuntime 内部一并重置）
//   - 档位（profile）Agent 缓存 —— 提案执行 / 评测等 headless 链路用的是这份缓存，
//     只重置默认 Agent 会让「聊天已用新模型、提案执行还在连旧地址」
//
// 失效后按需懒重建：下一轮对话、下一次会话摘要 / 记忆提取立即使用新配置；
// 正在执行的轮次持有旧实例，不受影响。
func ReloadModelConfig() {
	agent.ResetModelRuntime()
	ResetAllProfiles()
	logx.Info("[ai] model config changed, runtime & agents invalidated for next turn")
}

// ReloadAgentConfig Agent 配置（系统配置 AiAgentConfig）热生效入口
//
// 工具搜索阈值在装配 Agent 时读取，失效 Agent 缓存即下一轮生效。
// 注意：禁用扩展清单 / 会话与记忆存储目录属进程级装配项（扩展激活与中断注册表
// 在首次装配时冻结、存储实例已创建），保存后仍需重启服务才生效（见 config.AgentConfig）。
func ReloadAgentConfig() {
	agent.ResetDefaultAgent()
	ResetAllProfiles()
	logx.Info("[ai] agent config changed, agents invalidated for next turn")
}

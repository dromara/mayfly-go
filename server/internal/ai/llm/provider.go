// Package llm 定义「当前生效模型」的访问契约。
//
// 长生命周期组件（会话摘要器、长期记忆提取器、上下文/会话管理器）不得在装配期
// 固化 ChatModel 实例引用：模型地址 / Key / 模型名 / 上下文窗口等配置随时可能被
// 管理员改写，固化引用会让会话摘要、记忆提取、窗口压缩继续走旧地址（且编译与
// 日志全绿，只有真机改配置才能发现）。
//
// 这些组件改为在**每次使用时**经 Provider 读取当前生效的模型能力，模型配置变更后
// 宿主只需替换 Provider 背后的实现，组件本身无需重建（依赖倒置）。
//
// 本包是叶子包：只依赖 context 与 eino 模型接口，供 session / memory / agent
// 各层共用，不引入配置与装配层的传递依赖。
package llm

import (
	"context"

	"github.com/cloudwego/eino/components/model"
)

// Provider 当前生效模型能力的访问契约
type Provider interface {
	// ChatModel 返回当前生效的 ChatModel（未配置或构造失败时返回 error，
	// 由调用方按各自降级策略处理：摘要降级为规则摘要、记忆提取跳过等）
	ChatModel(ctx context.Context) (model.AgenticModel, error)
	// ContextWindow 返回当前模型的上下文窗口大小（token），0 表示未配置
	// （未配置时窗口检查与压缩决策跳过，与既有语义一致）
	ContextWindow() int
}

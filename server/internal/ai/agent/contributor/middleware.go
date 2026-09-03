package contributor

import (
	"context"

	"github.com/cloudwego/eino/adk"
)

// ToolMiddlewareContributor 工具执行中间件贡献者（对齐 tokhub ToolMiddlewareContributor）
//
// 返回的中间件由宿主聚合后注入 adk Handlers，参与工具调用链包装
// （错误转换、中断传播、审计等）。业务插件注册本通道贡献者即可追加
// 工具拦截逻辑，无需修改宿主代码。
// fail-open：单个贡献者失败记日志跳过，不阻断主流程。
type ToolMiddlewareContributor interface {
	Contributor
	// Middlewares 返回贡献的中间件列表
	Middlewares(ctx context.Context) ([]adk.ChatModelAgentMiddleware, error)
}

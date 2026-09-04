package middleware

import (
	"context"

	"mayfly-go/internal/ai/agent/contributor"
)

// SafeToolContributor 安全工具中间件贡献者（ToolMiddlewareContributor 通道）
//
// 将 SafeToolMiddleware 纳入贡献者通道：工具错误统一转换为 ToolError
// （中断错误原样传播、可重试错误转换为错误字符串）。业务插件可注册
// 中间件贡献者追加拦截逻辑，或经配置按 Id（safety_middleware）裁剪。
type SafeToolContributor struct{}

// NewSafeToolContributor 创建安全工具中间件贡献者
func NewSafeToolContributor() *SafeToolContributor {
	return &SafeToolContributor{}
}

var _ contributor.ToolMiddlewareContributor = (*SafeToolContributor)(nil)

func (c *SafeToolContributor) Id() string { return "safety_middleware" }

func (c *SafeToolContributor) Middlewares(ctx context.Context) ([]contributor.AgentMiddleware, error) {
	return []contributor.AgentMiddleware{&SafeToolMiddleware{}}, nil
}

// Install 注册中间件贡献者到 Builder
func Install(b *contributor.Builder) {
	b.RegisterMiddleware(NewSafeToolContributor())
}

package contributor

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
)

// ToolContributionContext 工具贡献上下文（Agent 构造期 per-agent）
//
// 仅携带真实填充且可能消费的信息；新增字段须以真实消费方为前提。
type ToolContributionContext struct {
	// AgentId 当前 Agent 标识（条件工具组按 Agent 过滤的注入点）
	AgentId string
}

// ToolContributor 工具贡献者
//
// 在构建工具列表时被调用，返回贡献的工具执行器。
// 所有工具（无条件内置工具、条件工具组）统一走此通道聚合，
//
// ## 同名工具的覆盖协议
//
// Registry 聚合时按注册顺序遍历，后注册者覆盖先注册者（保留先注册位置）。
// 内置工具贡献者注册在前、业务插件注册在后，因此插件可直接替换内置工具
// 实现，无需修改宿主代码（开闭原则）。
type ToolContributor interface {
	Contributor
	// Tools 返回贡献的工具列表；tc 可能为 nil（调用方无 Agent 上下文时）
	Tools(ctx context.Context, tc *ToolContributionContext) ([]tool.BaseTool, error)
}

// DeferredToolContributor 可延迟加载的工具贡献者（可选接口）
//
// 实现此接口的 ToolContributor 可声明其贡献的工具中哪些可延迟加载
// （初始对模型不可见，经 tool_search 元工具按需发现加载）。
// 用于工具总量超过阈值时避免挤占上下文与 KV-cache。
//
// 典型场景：MCP 工具数量庞大时，仅保留核心工具直注，其余经 tool_search
// 按需加载。内置核心工具（db/machine/resource/memory 等）不应实现此接口。
type DeferredToolContributor interface {
	ToolContributor
	// DeferredToolNames 返回可延迟加载的工具名称集合
	DeferredToolNames() map[string]struct{}
}

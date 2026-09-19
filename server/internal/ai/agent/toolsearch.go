package agent

import (
	"context"
	"mayfly-go/internal/ai/agent/contributor"
	"mayfly-go/pkg/logx"

	"github.com/cloudwego/eino/adk/middlewares/dynamictool/toolsearch"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// buildToolSearchMiddleware 构建工具搜索中间件（eino v0.9 tool search 能力）
//
// 架构说明：与 safety_middleware（静态无状态，走 ToolMiddlewareContributor
// 通道可被 DisabledExtensions 裁剪）不同，本中间件依赖 per-agent 的最终工具
// 清单（BuildTools 产出后才可确定 deferred 集合），且注册中心为多 Agent
// 共享实例、不允许携带 per-agent 状态，故在 NewAgent 内配置门控直接追加
// （属既有「选项指定者追加在后」路径），非装配旁路。
//
// deferredNames 由 Registry.BuildTools 经 DeferredToolContributor 可选接口
// 收集（贡献者自行声明哪些工具可延迟），不再硬编码特定来源前缀 ——
// 任何工具贡献者实现 DeferredToolContributor 即可接入 tool search 机制。
//
// 工具总量超过阈值（AgentConfig.ToolSearchThreshold）时，deferred 工具转为
// 延迟加载：初始对模型不可见，模型经 tool_search 元工具按关键词检索后按需
// 加载（Eino 自定义 tool_search 模式，不依赖模型侧协议）。内置核心工具
// （db/machine/resource/memory 等）不实现 DeferredToolContributor，始终直注，
// 保障基础运维能力不受影响。
//
// 返回第二个值为静态工具清单（排除 deferred 工具）：eino 官方接线契约是
// ToolsConfig.Tools 仅含静态工具，DynamicTools 由中间件在 BeforeAgent 阶段
// 追加为可执行工具（runCtx.Tools 为每轮权威工具集）——若 ToolsConfig 仍传
// 全量工具会导致重名工具在 runCtx.Tools 中重复注册。
//
// 未配置阈值（0）、未超阈值、或无 deferred 工具时返回 (nil, 原工具清单)
// （全量直注，行为不变）。构建失败 fail-open（记日志降级为全量直注，不阻断
// 装配）。
func buildToolSearchMiddleware(ctx context.Context, tools []tool.BaseTool, deferredNames map[string]struct{}, threshold int) (contributor.AgentMiddleware, []tool.BaseTool) {
	if threshold <= 0 || len(tools) <= threshold || len(deferredNames) == 0 {
		return nil, tools
	}
	var deferred, static []tool.BaseTool
	for _, t := range tools {
		info, err := t.Info(ctx)
		if err != nil {
			// 元信息异常的工具不延迟（保持直注，可用性优先）
			static = append(static, t)
			continue
		}
		if _, ok := deferredNames[info.Name]; ok {
			deferred = append(deferred, t)
		} else {
			static = append(static, t)
		}
	}
	if len(deferred) == 0 {
		return nil, tools
	}
	mw, err := toolsearch.NewTyped[*schema.AgenticMessage](ctx, &toolsearch.Config{
		DynamicTools: deferred,
	})
	if err != nil {
		logx.WarnfContext(ctx, "[agent] build tool search middleware failed, fallback to full tool list: %v", err)
		return nil, tools
	}
	logx.InfofContext(ctx, "[agent] tool search enabled: %d/%d tools deferred via tool_search (threshold=%d)",
		len(deferred), len(tools), threshold)
	return mw, static
}

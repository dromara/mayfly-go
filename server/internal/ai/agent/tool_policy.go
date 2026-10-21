package agent

import (
	"context"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/pkg/logx"

	"github.com/cloudwego/eino/components/tool"
)

// filterToolsByPolicy 按工具能力策略裁剪工具清单（通用 capability 过滤）
//
// 开闭原则收敛点：过滤完全基于工具在注册处声明的 capability（tools.CapabilityOf），
// 不针对任何具体工具名硬编码 —— 新增工具只需声明能力即可被正确纳入/排除，本函数零改动。
//
// 同时裁剪静态工具清单与 deferred 名称集合（deferred 工具经 tool_search 延迟加载，
// 若其能力不被策略放行则不应出现在可发现集合中）。
//
// 元信息读取失败的工具按保守策略排除（无法确认能力即不放行），并记日志。
// 返回裁剪后的工具清单与 deferred 名称集合。
func filterToolsByPolicy(ctx context.Context, src []tool.BaseTool, deferredNames map[string]struct{}, policy *tools.ToolPolicy) ([]tool.BaseTool, map[string]struct{}) {
	if !policy.IsRestricted() {
		return src, deferredNames
	}

	filtered := make([]tool.BaseTool, 0, len(src))
	var dropped int
	for _, t := range src {
		info, err := t.Info(ctx)
		if err != nil {
			// 无法确认能力的工具保守排除
			logx.WarnfContext(ctx, "[agent] tool policy: read tool info failed, drop tool: %v", err)
			dropped++
			continue
		}
		if policy.Allows(info.Name) {
			filtered = append(filtered, t)
			continue
		}
		dropped++
		// 被裁剪的工具同步从 deferred 集合移除（避免 tool_search 仍可发现）
		if deferredNames != nil {
			delete(deferredNames, info.Name)
		}
	}

	if dropped > 0 {
		logx.InfofContext(ctx, "[agent] tool policy applied: %d/%d tools allowed, %d dropped by capability filter",
			len(filtered), len(src), dropped)
	}
	return filtered, deferredNames
}

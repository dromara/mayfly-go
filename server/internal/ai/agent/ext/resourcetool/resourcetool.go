// Package resourcetoolext 资源清单工具扩展
//
// 实现 ToolContributor 通道：贡献资源清单查询工具（ListResources），
// 数据源为 ai/application/resource 统一资源查询服务。业务插件注册同名
// 工具即可覆盖（后注册胜出），或经配置按 Id（resource_tools）裁剪。
package resourcetoolext

import (
	"context"

	"mayfly-go/internal/ai/agent/contributor"
	resourcetool "mayfly-go/internal/ai/tools/resourcetool"

	"github.com/cloudwego/eino/components/tool"
)

// ResourceToolsExtension 资源清单工具扩展
type ResourceToolsExtension struct{}

// NewExtension 创建资源清单工具扩展
func NewExtension() *ResourceToolsExtension {
	return &ResourceToolsExtension{}
}

var _ contributor.ToolContributor = (*ResourceToolsExtension)(nil)

func (e *ResourceToolsExtension) Id() string { return "resource_tools" }

func (e *ResourceToolsExtension) Tools(ctx context.Context, tc *contributor.ToolContributionContext) ([]tool.BaseTool, error) {
	t, err := resourcetool.GetResourceList()
	if err != nil {
		return nil, err
	}
	return []tool.BaseTool{t}, nil
}

// Install 注册扩展到 Builder
func Install(b *contributor.Builder) {
	b.RegisterTool(NewExtension())
}

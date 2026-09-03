package skillext

import (
	"context"

	"mayfly-go/internal/ai/agent/contributor"
	"mayfly-go/internal/ai/tools"

	"github.com/cloudwego/eino/components/tool"
)

// SkillToolExtension 技能读取工具扩展（ToolContributor 通道）
//
// 贡献 skill_read 工具（渐进式披露 L3 内容层）。业务插件注册同名工具
// 即可覆盖（后注册胜出），或经配置按 Id（skill_tools）裁剪。
type SkillToolExtension struct{}

// NewToolExtension 创建技能读取工具扩展
func NewToolExtension() *SkillToolExtension {
	return &SkillToolExtension{}
}

var _ contributor.ToolContributor = (*SkillToolExtension)(nil)

func (e *SkillToolExtension) Id() string { return "skill_tools" }

func (e *SkillToolExtension) Tools(ctx context.Context, tc *contributor.ToolContributionContext) ([]tool.BaseTool, error) {
	t, err := tools.GetSkillRead()
	if err != nil {
		return nil, err
	}
	return []tool.BaseTool{t}, nil
}

// InstallTools 注册技能工具扩展到 Builder
func InstallTools(b *contributor.Builder) {
	b.RegisterTool(NewToolExtension())
}

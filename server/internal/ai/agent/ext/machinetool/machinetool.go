// Package machinetoolext 机器工具扩展
//
// 实现 ToolContributor 通道：贡献机器命令执行工具（MachineCommandExec）。
// 业务插件注册同名工具即可覆盖（后注册胜出），或经配置按 Id（machine_tools）裁剪。
package machinetoolext

import (
	"context"

	"mayfly-go/internal/ai/agent/contributor"
	machinetool "mayfly-go/internal/ai/tools/machinetool"

	"github.com/cloudwego/eino/components/tool"
)

// MachineToolsExtension 机器工具扩展
type MachineToolsExtension struct{}

// NewExtension 创建机器工具扩展
func NewExtension() *MachineToolsExtension {
	return &MachineToolsExtension{}
}

var _ contributor.ToolContributor = (*MachineToolsExtension)(nil)

func (e *MachineToolsExtension) Id() string { return "machine_tools" }

func (e *MachineToolsExtension) Tools(ctx context.Context, tc *contributor.ToolContributionContext) ([]tool.BaseTool, error) {
	return machinetool.Tools()
}

// Install 注册扩展到 Builder
func Install(b *contributor.Builder) {
	b.RegisterTool(NewExtension())
}

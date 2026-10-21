// Package proposaltoolext 变更提案工具扩展
//
// 实现 ToolContributor 通道：贡献 propose_remediation 工具（capability=read，只读档也可用）。
// 工具本身仅面向 ai/tools.ProposalRecorder 端口编程，提案的持久化与审批发起由 agentops
// 在 init 期注入实现（依赖倒置，破除 ai → agentops 反向依赖）。业务插件注册同名工具即可
// 覆盖（后注册胜出），或经配置按 Id（proposal_tools）裁剪。
package proposaltoolext

import (
	"context"

	"mayfly-go/internal/ai/agent/contributor"
	"mayfly-go/internal/ai/tools"

	"github.com/cloudwego/eino/components/tool"
)

// ProposalToolExtension 变更提案工具扩展
type ProposalToolExtension struct{}

// NewExtension 创建变更提案工具扩展
func NewExtension() *ProposalToolExtension {
	return &ProposalToolExtension{}
}

var _ contributor.ToolContributor = (*ProposalToolExtension)(nil)

func (e *ProposalToolExtension) Id() string { return "proposal_tools" }

func (e *ProposalToolExtension) Tools(_ context.Context, _ *contributor.ToolContributionContext) ([]tool.BaseTool, error) {
	return tools.ProposeRemediationTools()
}

// Install 注册扩展到 Builder
func Install(b *contributor.Builder) {
	b.RegisterTool(NewExtension())
}

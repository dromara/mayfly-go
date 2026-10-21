// Package monitortool 可观测数据工具扩展
//
// 实现 ToolContributor 通道：贡献只读可观测查询工具（QueryMetrics、QueryAlertHistory），
// 补齐 Agent 对指标/告警历史的原生感知（对齐 Datadog Bits AI / PagerDuty AIOps 的
// 自动调查能力）。依赖业务应用层（alert），经 ai/init 的宿主级安装器注册（与 dbtool
// 同范式），避免 agent → 业务层循环依赖。
package monitortool

import (
	"context"

	"mayfly-go/internal/ai/agent/contributor"
	monitor "mayfly-go/internal/ai/tools/monitor"

	"github.com/cloudwego/eino/components/tool"
)

// MonitorToolsExtension 可观测数据工具扩展
type MonitorToolsExtension struct{}

// NewExtension 创建可观测数据工具扩展
func NewExtension() *MonitorToolsExtension {
	return &MonitorToolsExtension{}
}

var _ contributor.ToolContributor = (*MonitorToolsExtension)(nil)

func (e *MonitorToolsExtension) Id() string { return "monitor_tools" }

func (e *MonitorToolsExtension) Tools(ctx context.Context, tc *contributor.ToolContributionContext) ([]tool.BaseTool, error) {
	return monitor.Tools()
}

// Install 注册扩展到 Builder
func Install(b *contributor.Builder) {
	b.RegisterTool(NewExtension())
}

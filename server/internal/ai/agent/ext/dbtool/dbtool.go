// Package dbtoolext 数据库工具扩展
//
// 实现 ToolContributor 通道：贡献数据库查询/执行工具（QueryTableDDL、
// QueryTables、QueryData、ExecSql）。业务插件注册同名工具即可覆盖
// （后注册胜出），或经配置按 Id（db_tools）裁剪。
package dbtoolext

import (
	"context"

	"mayfly-go/internal/ai/agent/contributor"
	dbtool "mayfly-go/internal/ai/tools/dbtool"

	"github.com/cloudwego/eino/components/tool"
)

// DBToolsExtension 数据库工具扩展
type DBToolsExtension struct{}

// NewExtension 创建数据库工具扩展
func NewExtension() *DBToolsExtension {
	return &DBToolsExtension{}
}

var _ contributor.ToolContributor = (*DBToolsExtension)(nil)

func (e *DBToolsExtension) Id() string { return "db_tools" }

func (e *DBToolsExtension) Tools(ctx context.Context, tc *contributor.ToolContributionContext) ([]tool.BaseTool, error) {
	return dbtool.Tools()
}

// Install 注册扩展到 Builder
func Install(b *contributor.Builder) {
	b.RegisterTool(NewExtension())
}

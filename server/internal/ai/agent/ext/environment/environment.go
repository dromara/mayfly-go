// Package environmentext 轮次环境信息扩展
//
// 实现 ContextContributor 通道：每轮向 preamble 注入当前用户/服务器时间等环境信息。
package environmentext

import (
	"context"
	"fmt"
	"time"

	"mayfly-go/internal/ai/agent/contributor"
	"mayfly-go/pkg/contextx"
)

// EnvironmentExtension 轮次环境信息扩展
type EnvironmentExtension struct{}

// NewExtension 创建环境信息扩展
func NewExtension() *EnvironmentExtension {
	return &EnvironmentExtension{}
}

var _ contributor.ContextContributor = (*EnvironmentExtension)(nil)

func (e *EnvironmentExtension) Id() string     { return "environment" }
func (e *EnvironmentExtension) Retained() bool { return false }
func (e *EnvironmentExtension) Priority() int  { return 40 }

func (e *EnvironmentExtension) ContributeTurnContext(ctx context.Context, in *contributor.TurnInput) ([]contributor.PromptFragment, error) {
	userName := ""
	if la := contextx.GetLoginAccount(ctx); la != nil {
		userName = la.Username
	}
	content := fmt.Sprintf("[环境信息]\n- 当前用户: %s\n- 服务器时间: %s",
		userName, time.Now().Format("2006-01-02 15:04:05 MST"))
	return []contributor.PromptFragment{
		{Source: "environment", Content: content},
	}, nil
}

// Install 注册扩展到 Builder（含预算提醒扩展）
func Install(b *contributor.Builder) {
	b.RegisterContext(NewExtension())
	b.RegisterPreambleFooter(NewBudgetFooterExtension())
}

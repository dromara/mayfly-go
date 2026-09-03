// Package memoryext 长期记忆扩展
//
// 实现 ContextContributor 通道：每轮向 preamble 注入用户长期记忆片段。
package memoryext

import (
	"context"

	"mayfly-go/internal/ai/agent/contributor"
	"mayfly-go/internal/ai/memory"
)

// 片段优先级约定（数值越大越优先保留）：memory=60

// MemoryExtension 长期记忆扩展
type MemoryExtension struct {
	memoryManager *memory.Manager
}

// NewExtension 创建记忆扩展（memoryManager 允许为 nil，未启用记忆时自动跳过）
func NewExtension(memoryManager *memory.Manager) *MemoryExtension {
	return &MemoryExtension{memoryManager: memoryManager}
}

var _ contributor.ContextContributor = (*MemoryExtension)(nil)

func (e *MemoryExtension) Id() string     { return "memory" }
func (e *MemoryExtension) Retained() bool { return false }
func (e *MemoryExtension) Priority() int  { return 60 }

func (e *MemoryExtension) ContributeTurnContext(ctx context.Context, in *contributor.TurnInput) ([]contributor.PromptFragment, error) {
	if e.memoryManager == nil || in.UserId == "" {
		return nil, nil
	}
	memoryMsg := e.memoryManager.BuildMemoryMessage(ctx, in.UserId)
	if memoryMsg == nil {
		return nil, nil
	}
	return []contributor.PromptFragment{
		{Source: "memory", Content: memoryMsg.Content},
	}, nil
}

// Install 注册扩展到 Builder
func Install(b *contributor.Builder, memoryManager *memory.Manager) {
	b.RegisterContext(NewExtension(memoryManager))
}

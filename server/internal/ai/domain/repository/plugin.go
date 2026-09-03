package repository

import (
	"context"

	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/pkg/base"
)

// Skill 技能仓库
type Skill interface {
	base.Repo[*entity.Skill]

	// SelectByCode 按 code 查询技能（未删除）
	SelectByCode(ctx context.Context, code string) (*entity.Skill, error)
	// SelectByStatus 按状态查询技能（code 排序，保证目录渲染顺序稳定）
	SelectByStatus(ctx context.Context, status string) ([]*entity.Skill, error)
}

// SkillResource 技能资源仓库
type SkillResource interface {
	base.Repo[*entity.SkillResource]

	// SelectBySkillId 查询技能全部资源（path 排序）
	SelectBySkillId(ctx context.Context, skillId uint64) ([]*entity.SkillResource, error)
	// SelectBySkillIdAndPath 按 (skillId, path) 精确查询资源
	SelectBySkillIdAndPath(ctx context.Context, skillId uint64, path string) (*entity.SkillResource, error)
	// DeleteBySkillId 删除技能全部资源
	DeleteBySkillId(ctx context.Context, skillId uint64) error
}

// McpServer MCP 服务器仓库
type McpServer interface {
	base.Repo[*entity.McpServer]

	// SelectByCode 按 code 查询 MCP 服务器（未删除）
	SelectByCode(ctx context.Context, code string) (*entity.McpServer, error)
	// SelectEnabled 查询启用中的 MCP 服务器（id 排序）
	SelectEnabled(ctx context.Context) ([]*entity.McpServer, error)
}

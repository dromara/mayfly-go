package repository

import (
	"context"
	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/pkg/base"
)

type Memory interface {
	base.Repo[*entity.Memory]

	// SelectByUser 查询用户记忆（按 update_time 倒序，近期记忆优先）
	//   - keywords 非空时按 content/tags 模糊匹配（OR 语义，任一命中即返回）
	//   - tags 非空时按标签过滤（含任一标签即返回）
	//   - limit <= 0 时默认 100
	SelectByUser(ctx context.Context, userId string, keywords []string, tags []string, limit int) ([]*entity.Memory, error)
}

package repository

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
)

type TagTreeTeam interface {
	base.Repo[*entity.TagTreeTeam]

	// 获取团队标签信息列表
	// ListTag(condition *entity.TagTreeTeam, toEntity any, orderBy ...string)

	SelectTagPathsByAccountId(accountId uint64) []string
}

package repository

import "mayfly-go/internal/tag/domain/entity"

type TagTreeTeam interface {

	// 获取团队标签信息列表
	ListTag(condition *entity.TagTreeTeam, toEntity interface{}, orderBy ...string)

	Save(mp *entity.TagTreeTeam)

	DeleteBy(condition *entity.TagTreeTeam)

	SelectTagPathsByAccountId(accountId uint64) []string
}

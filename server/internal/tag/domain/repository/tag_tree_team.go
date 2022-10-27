package repository

import "mayfly-go/internal/tag/domain/entity"

type TagTreeTeam interface {

	// 获取团队项目信息列表
	ListProject(condition *entity.TagTreeTeam, toEntity interface{}, orderBy ...string)

	Save(mp *entity.TagTreeTeam)

	DeleteBy(condition *entity.TagTreeTeam)

	SelectTagPathsByAccountId(accountId uint64) []string
}

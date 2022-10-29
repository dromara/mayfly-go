package repository

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/model"
)

type TeamMember interface {

	// 获取项目成员列表
	ListMemeber(condition *entity.TeamMember, toEntity interface{}, orderBy ...string)

	Save(mp *entity.TeamMember)

	GetPageList(condition *entity.TeamMember, pageParam *model.PageParam, toEntity interface{}) *model.PageResult

	DeleteBy(condition *entity.TeamMember)

	// 是否存在指定的团队成员关联信息
	IsExist(teamId, accountId uint64) bool
}

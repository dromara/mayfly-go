package repository

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type TeamMember interface {
	base.Repo[*entity.TeamMember]

	// 获取项目成员列表
	ListMemeber(condition *entity.TeamMember, toEntity any, orderBy ...string)

	GetPageList(condition *entity.TeamMember, pageParam *model.PageParam, toEntity any) (*model.PageResult[any], error)

	// 是否存在指定的团队成员关联信息
	IsExist(teamId, accountId uint64) bool
}

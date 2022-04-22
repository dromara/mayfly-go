package repository

import (
	"mayfly-go/base/model"
	"mayfly-go/server/devops/domain/entity"
)

type ProjectMemeber interface {

	// 获取项目成员列表
	ListMemeber(condition *entity.ProjectMember, toEntity interface{}, orderBy ...string)

	Save(mp *entity.ProjectMember)

	GetPageList(condition *entity.ProjectMember, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	// 根据成员id和项目id删除关联关系
	DeleteByPidMid(projectId, accountId uint64)

	DeleteMems(projectId uint64)

	// 是否存在指定的项目成员关联信息
	IsExist(projectId, accountId uint64) bool
}

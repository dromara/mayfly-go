package repository

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type Role interface {
	base.Repo[*entity.Role]

	GetPageList(condition *entity.RoleQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	ListByQuery(condition *entity.RoleQuery) ([]*entity.Role, error)
}

type AccountRole interface {
	base.Repo[*entity.AccountRole]

	GetPageList(condition *entity.RoleAccountQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}

type RoleResource interface {
	base.Repo[*entity.RoleResource]

	// 获取角色拥有的资源id数组，从role_resource表获取
	GetRoleResourceIds(roleId uint64) []uint64

	GetRoleResources(roleId uint64, toEntity any)
}

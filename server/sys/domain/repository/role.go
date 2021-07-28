package repository

import (
	"mayfly-go/base/model"
	"mayfly-go/server/sys/domain/entity"
)

type Role interface {
	GetPageList(condition *entity.Role, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	Delete(id uint64)

	// 获取角色拥有的资源id数组，从role_resource表获取
	GetRoleResourceIds(roleId uint64) []uint64

	GetRoleResources(roleId uint64, toEntity interface{})

	SaveRoleResource(rr *entity.RoleResource)

	DeleteRoleResource(roleId uint64, resourceId uint64)

	// 获取账号拥有的角色id数组，从account_role表获取
	GetAccountRoleIds(accountId uint64) []uint64

	SaveAccountRole(ar *entity.AccountRole)

	DeleteAccountRole(accountId, roleId uint64)

	// 获取账号角色信息列表
	GetAccountRoles(accountId uint64, toEntity interface{})
}

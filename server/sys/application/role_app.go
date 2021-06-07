package application

import (
	"mayfly-go/base/model"
	"mayfly-go/server/sys/domain/entity"
	"mayfly-go/server/sys/domain/repository"
	"mayfly-go/server/sys/infrastructure/persistence"
)

type IRole interface {
	GetPageList(condition *entity.Role, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) model.PageResult

	SaveRole(role *entity.Role)

	DeleteRole(id uint64)

	GetRoleResourceIds(roleId uint64) []uint64

	GetRoleResources(roleId uint64, toEntity interface{})

	SaveRoleResource(rr *entity.RoleResource)

	DeleteRoleResource(roleId uint64, resourceId uint64)

	GetAccountRoleIds(accountId uint64) []uint64

	SaveAccountRole(rr *entity.AccountRole)

	DeleteAccountRole(accountId, roleId uint64)

	GetAccountRoles(accountId uint64, toEntity interface{})
}

type roleApp struct {
	roleRepo repository.Role
}

// 实现类单例
var Role IRole = &roleApp{
	roleRepo: persistence.RoleDao,
}

func (m *roleApp) GetPageList(condition *entity.Role, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) model.PageResult {
	return m.roleRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

func (m *roleApp) SaveRole(role *entity.Role) {
	if role.Id != 0 {
		model.UpdateById(role)
	} else {
		model.Insert(role)
	}
}

func (m *roleApp) DeleteRole(id uint64) {
	m.roleRepo.Delete(id)
	// 删除角色与资源的关联关系
	model.DeleteByCondition(&entity.RoleResource{RoleId: id})
}

func (m *roleApp) GetRoleResourceIds(roleId uint64) []uint64 {
	return m.roleRepo.GetRoleResourceIds(roleId)
}

func (m *roleApp) GetRoleResources(roleId uint64, toEntity interface{}) {
	m.roleRepo.GetRoleResources(roleId, toEntity)
}

func (m *roleApp) SaveRoleResource(rr *entity.RoleResource) {
	m.roleRepo.SaveRoleResource(rr)
}

func (m *roleApp) DeleteRoleResource(roleId uint64, resourceId uint64) {
	m.roleRepo.DeleteRoleResource(roleId, resourceId)
}

func (m *roleApp) GetAccountRoleIds(accountId uint64) []uint64 {
	return m.roleRepo.GetAccountRoleIds(accountId)
}

func (m *roleApp) SaveAccountRole(rr *entity.AccountRole) {
	m.roleRepo.SaveAccountRole(rr)
}

func (m *roleApp) DeleteAccountRole(accountId, roleId uint64) {
	m.roleRepo.DeleteAccountRole(accountId, roleId)
}

func (m *roleApp) GetAccountRoles(accountId uint64, toEntity interface{}) {
	m.roleRepo.GetAccountRoles(accountId, toEntity)
}

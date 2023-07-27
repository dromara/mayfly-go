package application

import (
	"context"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"strings"
	"time"
)

type Role interface {
	GetPageList(condition *entity.Role, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any]

	SaveRole(role *entity.Role)

	DeleteRole(id uint64)

	GetRoleResourceIds(roleId uint64) []uint64

	GetRoleResources(roleId uint64, toEntity any)

	// 保存角色资源关联记录
	SaveRoleResource(ctx context.Context, roleId uint64, resourceIds []uint64)

	// 删除角色资源关联记录
	DeleteRoleResource(roleId uint64, resourceId uint64)

	// 获取账号角色id列表
	GetAccountRoleIds(accountId uint64) []uint64

	// 保存账号角色关联信息
	SaveAccountRole(ctx context.Context, accountId uint64, roleIds []uint64)

	DeleteAccountRole(accountId, roleId uint64)

	GetAccountRoles(accountId uint64, toEntity any)
}

func newRoleApp(roleRepo repository.Role) Role {
	return &roleAppImpl{
		roleRepo: roleRepo,
	}
}

type roleAppImpl struct {
	roleRepo repository.Role
}

func (m *roleAppImpl) GetPageList(condition *entity.Role, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any] {
	return m.roleRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

func (m *roleAppImpl) SaveRole(role *entity.Role) {
	role.Code = strings.ToUpper(role.Code)
	if role.Id != 0 {
		// code不可更改，防止误传
		role.Code = ""
		gormx.UpdateById(role)
	} else {
		role.Status = 1
		gormx.Insert(role)
	}
}

func (m *roleAppImpl) DeleteRole(id uint64) {
	m.roleRepo.Delete(id)
	// 删除角色与资源的关联关系
	gormx.DeleteByCondition(&entity.RoleResource{RoleId: id})
}

func (m *roleAppImpl) GetRoleResourceIds(roleId uint64) []uint64 {
	return m.roleRepo.GetRoleResourceIds(roleId)
}

func (m *roleAppImpl) GetRoleResources(roleId uint64, toEntity any) {
	m.roleRepo.GetRoleResources(roleId, toEntity)
}

func (m *roleAppImpl) SaveRoleResource(ctx context.Context, roleId uint64, resourceIds []uint64) {
	oIds := m.GetRoleResourceIds(roleId)

	addIds, delIds, _ := collx.ArrayCompare(resourceIds, oIds, func(i1, i2 uint64) bool {
		return i1 == i2
	})

	la := contextx.GetLoginAccount(ctx)
	createTime := time.Now()
	creator := la.Username
	creatorId := la.Id
	undeleted := model.ModelUndeleted

	addVals := make([]*entity.RoleResource, 0)
	for _, v := range addIds {
		rr := &entity.RoleResource{RoleId: roleId, ResourceId: v, CreateTime: &createTime, CreatorId: creatorId, Creator: creator}
		rr.IsDeleted = undeleted
		addVals = append(addVals, rr)
	}
	m.roleRepo.SaveRoleResource(addVals)

	for _, v := range delIds {
		m.DeleteRoleResource(roleId, v)
	}
}

func (m *roleAppImpl) DeleteRoleResource(roleId uint64, resourceId uint64) {
	m.roleRepo.DeleteRoleResource(roleId, resourceId)
}

func (m *roleAppImpl) GetAccountRoleIds(accountId uint64) []uint64 {
	return m.roleRepo.GetAccountRoleIds(accountId)
}

// 保存账号角色关联信息
func (m *roleAppImpl) SaveAccountRole(ctx context.Context, accountId uint64, roleIds []uint64) {
	oIds := m.GetAccountRoleIds(accountId)

	addIds, delIds, _ := collx.ArrayCompare(roleIds, oIds, func(i1, i2 uint64) bool {
		return i1 == i2
	})

	la := contextx.GetLoginAccount(ctx)

	createTime := time.Now()
	creator := la.Username
	creatorId := la.Id
	for _, v := range addIds {
		rr := &entity.AccountRole{AccountId: accountId, RoleId: v, CreateTime: &createTime, CreatorId: creatorId, Creator: creator}
		m.roleRepo.SaveAccountRole(rr)
	}
	for _, v := range delIds {
		m.DeleteAccountRole(accountId, v)
	}
}

func (m *roleAppImpl) DeleteAccountRole(accountId, roleId uint64) {
	m.roleRepo.DeleteAccountRole(accountId, roleId)
}

func (m *roleAppImpl) GetAccountRoles(accountId uint64, toEntity any) {
	m.roleRepo.GetAccountRoles(accountId, toEntity)
}

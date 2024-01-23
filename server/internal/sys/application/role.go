package application

import (
	"context"
	"mayfly-go/internal/sys/consts"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"time"

	"gorm.io/gorm"
)

type Role interface {
	GetPageList(condition *entity.RoleQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	ListByQuery(condition *entity.RoleQuery) ([]*entity.Role, error)

	SaveRole(ctx context.Context, role *entity.Role) error

	DeleteRole(ctx context.Context, id uint64) error

	GetRoleResourceIds(roleId uint64) []uint64

	GetRoleResources(roleId uint64, toEntity any)

	// 保存角色资源关联记录
	SaveRoleResource(ctx context.Context, roleId uint64, resourceIds []uint64)

	// 关联账号角色
	RelateAccountRole(ctx context.Context, accountId, roleId uint64, relateType consts.AccountRoleRelateType) error

	// 获取账号关联角色
	GetAccountRoles(accountId uint64) ([]*entity.AccountRole, error)

	// 获取角色关联的用户信息
	GetRoleAccountPage(condition *entity.RoleAccountQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}

type roleAppImpl struct {
	roleRepo        repository.Role        `inject:"RoleRepo"`
	accountRoleRepo repository.AccountRole `inject:"AccountRoleRepo"`
}

func (m *roleAppImpl) GetPageList(condition *entity.RoleQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return m.roleRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

func (m *roleAppImpl) ListByQuery(condition *entity.RoleQuery) ([]*entity.Role, error) {
	return m.roleRepo.ListByQuery(condition)
}

func (m *roleAppImpl) SaveRole(ctx context.Context, role *entity.Role) error {
	if role.Id != 0 {
		// code不可更改，防止误传
		role.Code = ""
		return gormx.UpdateById(role)
	}

	role.Status = 1
	return m.roleRepo.Insert(ctx, role)
}

func (m *roleAppImpl) DeleteRole(ctx context.Context, id uint64) error {
	// 删除角色与资源账号的关联关系
	return gormx.Tx(
		func(db *gorm.DB) error {
			return m.roleRepo.DeleteByIdWithDb(ctx, db, id)
		},
		func(db *gorm.DB) error {
			return gormx.DeleteByWithDb(db, &entity.RoleResource{RoleId: id})
		},
		func(db *gorm.DB) error {
			return gormx.DeleteByWithDb(db, &entity.AccountRole{RoleId: id})
		},
	)
}

func (m *roleAppImpl) GetRoleResourceIds(roleId uint64) []uint64 {
	return m.roleRepo.GetRoleResourceIds(roleId)
}

func (m *roleAppImpl) GetRoleResources(roleId uint64, toEntity any) {
	m.roleRepo.GetRoleResources(roleId, toEntity)
}

func (m *roleAppImpl) SaveRoleResource(ctx context.Context, roleId uint64, resourceIds []uint64) {
	oIds := m.GetRoleResourceIds(roleId)

	addIds, delIds, _ := collx.ArrayCompare(resourceIds, oIds)

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
		m.roleRepo.DeleteRoleResource(roleId, v)
	}
}

func (m *roleAppImpl) RelateAccountRole(ctx context.Context, accountId, roleId uint64, relateType consts.AccountRoleRelateType) error {
	accountRole := &entity.AccountRole{AccountId: accountId, RoleId: roleId}
	if relateType == consts.AccountRoleUnbind {
		return m.accountRoleRepo.DeleteByCond(ctx, accountRole)
	}

	err := m.accountRoleRepo.GetBy(accountRole)
	if err == nil {
		return errorx.NewBiz("该用户已拥有该权限")
	}

	la := contextx.GetLoginAccount(ctx)
	createTime := time.Now()
	accountRole.Creator = la.Username
	accountRole.CreatorId = la.Id
	accountRole.CreateTime = &createTime
	return m.accountRoleRepo.Insert(ctx, accountRole)
}

func (m *roleAppImpl) GetAccountRoles(accountId uint64) ([]*entity.AccountRole, error) {
	var res []*entity.AccountRole
	err := m.accountRoleRepo.ListByCond(&entity.AccountRole{AccountId: accountId}, &res)
	return res, err
}

func (m *roleAppImpl) GetRoleAccountPage(condition *entity.RoleAccountQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return m.accountRoleRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

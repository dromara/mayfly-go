package application

import (
	"context"
	"mayfly-go/internal/sys/consts"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"time"
)

type Role interface {
	base.App[*entity.Role]

	GetPageList(condition *entity.RoleQuery, orderBy ...string) (*model.PageResult[*entity.Role], error)

	ListByQuery(condition *entity.RoleQuery) ([]*entity.Role, error)

	SaveRole(ctx context.Context, role *entity.Role) error

	DeleteRole(ctx context.Context, id uint64) error

	GetRoleResourceIds(roleId uint64) []uint64

	GetRoleResources(roleId uint64, toEntity any)

	GetResourceRoles(resourceId uint64) ([]*entity.RoleResource, error)

	// 保存角色资源关联记录
	SaveRoleResource(ctx context.Context, roleId uint64, resourceIds []uint64) error

	// 关联账号角色
	RelateAccountRole(ctx context.Context, accountId, roleId uint64, relateType consts.AccountRoleRelateType) error

	// 获取账号关联角色
	GetAccountRoles(accountId uint64) ([]*entity.AccountRole, error)

	// 获取角色关联的用户信息
	GetRoleAccountPage(condition *entity.RoleAccountQuery, orderBy ...string) (*model.PageResult[*entity.AccountRolePO], error)
}

type roleAppImpl struct {
	base.AppImpl[*entity.Role, repository.Role]

	accountRoleRepo  repository.AccountRole  `inject:"T"`
	roleResourceRepo repository.RoleResource `inject:"T"`
}

var _ (Role) = (*roleAppImpl)(nil)

func (m *roleAppImpl) GetPageList(condition *entity.RoleQuery, orderBy ...string) (*model.PageResult[*entity.Role], error) {
	return m.GetRepo().GetPageList(condition, orderBy...)
}

func (m *roleAppImpl) ListByQuery(condition *entity.RoleQuery) ([]*entity.Role, error) {
	return m.GetRepo().ListByQuery(condition)
}

func (m *roleAppImpl) SaveRole(ctx context.Context, role *entity.Role) error {
	if role.Id != 0 {
		// code不可更改，防止误传
		role.Code = ""
		return m.UpdateById(ctx, role)
	}

	role.Status = 1
	return m.Insert(ctx, role)
}

func (m *roleAppImpl) DeleteRole(ctx context.Context, id uint64) error {
	// 删除角色与资源账号的关联关系
	return m.Tx(ctx, func(ctx context.Context) error {
		return m.DeleteById(ctx, id)
	}, func(ctx context.Context) error {
		return m.roleResourceRepo.DeleteByCond(ctx, &entity.RoleResource{RoleId: id})
	}, func(ctx context.Context) error {
		return m.accountRoleRepo.DeleteByCond(ctx, &entity.AccountRole{RoleId: id})
	})
}

func (m *roleAppImpl) GetRoleResourceIds(roleId uint64) []uint64 {
	return m.roleResourceRepo.GetRoleResourceIds(roleId)
}

func (m *roleAppImpl) GetRoleResources(roleId uint64, toEntity any) {
	m.roleResourceRepo.GetRoleResources(roleId, toEntity)
}

func (m *roleAppImpl) GetResourceRoles(resourceId uint64) ([]*entity.RoleResource, error) {
	return m.roleResourceRepo.SelectByCond(&entity.RoleResource{ResourceId: resourceId})
}

func (m *roleAppImpl) SaveRoleResource(ctx context.Context, roleId uint64, resourceIds []uint64) error {
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

	return m.Tx(ctx, func(ctx context.Context) error {
		if len(addVals) > 0 {
			return m.roleResourceRepo.BatchInsert(ctx, addVals)
		}
		return nil
	}, func(ctx context.Context) error {
		if len(delIds) > 0 {
			return m.roleResourceRepo.DeleteByCond(ctx, model.NewCond().In("resource_id", delIds))
		}
		return nil
	})
}

func (m *roleAppImpl) RelateAccountRole(ctx context.Context, accountId, roleId uint64, relateType consts.AccountRoleRelateType) error {
	accountRole := &entity.AccountRole{AccountId: accountId, RoleId: roleId}
	if relateType == consts.AccountRoleUnbind {
		return m.accountRoleRepo.DeleteByCond(ctx, accountRole)
	}

	err := m.accountRoleRepo.GetByCond(accountRole)
	if err == nil {
		return errorx.NewBiz("The user already owns the role")
	}

	la := contextx.GetLoginAccount(ctx)
	createTime := time.Now()
	accountRole.Creator = la.Username
	accountRole.CreatorId = la.Id
	accountRole.CreateTime = &createTime
	return m.accountRoleRepo.Insert(ctx, accountRole)
}

func (m *roleAppImpl) GetAccountRoles(accountId uint64) ([]*entity.AccountRole, error) {
	return m.accountRoleRepo.SelectByCond(&entity.AccountRole{AccountId: accountId})
}

func (m *roleAppImpl) GetRoleAccountPage(condition *entity.RoleAccountQuery, orderBy ...string) (*model.PageResult[*entity.AccountRolePO], error) {
	return m.accountRoleRepo.GetPageList(condition, orderBy...)
}

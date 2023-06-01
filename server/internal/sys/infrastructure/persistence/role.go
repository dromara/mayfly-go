package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type roleRepoImpl struct{}

func newRoleRepo() repository.Role {
	return new(roleRepoImpl)
}

func (m *roleRepoImpl) GetPageList(condition *entity.Role, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult {
	return model.GetPage(pageParam, condition, condition, toEntity, orderBy...)
}

func (m *roleRepoImpl) Delete(id uint64) {
	biz.ErrIsNil(model.DeleteById(new(entity.Role), id), "删除角色失败")
}

// 获取角色拥有的资源id数组，从role_resource表获取
func (m *roleRepoImpl) GetRoleResourceIds(roleId uint64) []uint64 {
	var rrs []entity.RoleResource

	condtion := &entity.RoleResource{RoleId: roleId}
	model.ListBy(condtion, &rrs, "ResourceId")

	var rids []uint64
	for _, v := range rrs {
		rids = append(rids, v.ResourceId)
	}
	return rids
}

func (m *roleRepoImpl) GetRoleResources(roleId uint64, toEntity any) {
	sql := "select rr.creator AS creator, rr.create_time AS CreateTime, rr.resource_id AS id, r.pid AS pid, " +
		"r.name AS name, r.type AS type, r.status AS status " +
		"FROM t_sys_role_resource rr JOIN t_sys_resource r ON rr.resource_id = r.id " +
		"WHERE rr.role_id = ? " +
		"ORDER BY r.pid ASC, r.weight ASC"
	model.GetListBySql2Model(sql, toEntity, roleId)
}

func (m *roleRepoImpl) SaveRoleResource(rr *entity.RoleResource) {
	model.Insert(rr)
}

func (m *roleRepoImpl) DeleteRoleResource(roleId uint64, resourceId uint64) {
	model.DeleteByCondition(&entity.RoleResource{RoleId: roleId, ResourceId: resourceId})
}

func (m *roleRepoImpl) GetAccountRoleIds(accountId uint64) []uint64 {
	var rrs []entity.AccountRole

	condtion := &entity.AccountRole{AccountId: accountId}
	model.ListBy(condtion, &rrs, "RoleId")

	var rids []uint64
	for _, v := range rrs {
		rids = append(rids, v.RoleId)
	}
	return rids
}

func (m *roleRepoImpl) SaveAccountRole(ar *entity.AccountRole) {
	model.Insert(ar)
}

func (m *roleRepoImpl) DeleteAccountRole(accountId, roleId uint64) {
	model.DeleteByCondition(&entity.AccountRole{RoleId: roleId, AccountId: accountId})
}

// 获取账号角色信息列表
func (m *roleRepoImpl) GetAccountRoles(accountId uint64, toEntity any) {
	sql := "SELECT r.status, r.name, ar.create_time AS CreateTime, ar.creator AS creator " +
		"FROM t_sys_role r JOIN t_sys_account_role ar ON r.id = ar.role_id AND ar.account_id = ? " +
		"ORDER BY ar.create_time DESC"
	model.GetListBySql2Model(sql, toEntity, accountId)
}

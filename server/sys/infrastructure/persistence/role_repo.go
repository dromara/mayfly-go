package persistence

import (
	"mayfly-go/base/biz"
	"mayfly-go/base/model"
	"mayfly-go/server/sys/domain/entity"
	"mayfly-go/server/sys/domain/repository"
)

type roleRepo struct{}

var RoleDao repository.Role = &roleRepo{}

func (m *roleRepo) GetPageList(condition *entity.Role, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) model.PageResult {
	return model.GetPage(pageParam, condition, toEntity, orderBy...)
}

func (m *roleRepo) Delete(id uint64) {
	biz.ErrIsNil(model.DeleteById(new(entity.Role), id), "删除角色失败")
}

// 获取角色拥有的资源id数组，从role_resource表获取
func (m *roleRepo) GetRoleResourceIds(roleId uint64) []uint64 {
	var rrs []entity.RoleResource

	condtion := &entity.RoleResource{RoleId: roleId}
	model.ListBy(condtion, &rrs, "ResourceId")

	var rids []uint64
	for _, v := range rrs {
		rids = append(rids, v.ResourceId)
	}
	return rids
}

func (m *roleRepo) GetRoleResources(roleId uint64, toEntity interface{}) {
	sql := "select rr.creator AS creator, rr.create_time AS CreateTime, rr.resource_id AS id, r.pid AS pid, " +
		"r.name AS name, r.type AS type, r.status AS status " +
		"FROM t_role_resource rr JOIN t_resource r ON rr.resource_id = r.id " +
		"WHERE rr.role_id = ? " +
		"ORDER BY r.pid ASC, r.weight ASC"
	model.GetListBySql2Model(sql, toEntity, roleId)
}

func (m *roleRepo) SaveRoleResource(rr *entity.RoleResource) {
	model.Insert(rr)
}

func (m *roleRepo) DeleteRoleResource(roleId uint64, resourceId uint64) {
	model.DeleteByCondition(&entity.RoleResource{RoleId: roleId, ResourceId: resourceId})
}

func (m *roleRepo) GetAccountRoleIds(accountId uint64) []uint64 {
	var rrs []entity.AccountRole

	condtion := &entity.AccountRole{AccountId: accountId}
	model.ListBy(condtion, &rrs, "RoleId")

	var rids []uint64
	for _, v := range rrs {
		rids = append(rids, v.RoleId)
	}
	return rids
}

func (m *roleRepo) SaveAccountRole(ar *entity.AccountRole) {
	model.Insert(ar)
}

func (m *roleRepo) DeleteAccountRole(accountId, roleId uint64) {
	model.DeleteByCondition(&entity.AccountRole{RoleId: roleId, AccountId: accountId})
}

// 获取账号角色信息列表
func (m *roleRepo) GetAccountRoles(accountId uint64, toEntity interface{}) {
	sql := "SELECT r.status, r.name, ar.create_time AS CreateTime, ar.creator AS creator " +
		"FROM t_role r JOIN t_account_role ar ON r.id = ar.role_id AND ar.account_id = ? " +
		"ORDER BY ar.create_time DESC"
	model.GetListBySql2Model(sql, toEntity, accountId)
}

package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/base"
)

type roleResourceRepoImpl struct {
	base.RepoImpl[*entity.RoleResource]
}

func newRoleResourceRepo() repository.RoleResource {
	return &roleResourceRepoImpl{}
}

// 获取角色拥有的资源id数组
func (m *roleResourceRepoImpl) GetRoleResourceIds(roleId uint64) []uint64 {
	rrs, _ := m.SelectByCond(&entity.RoleResource{RoleId: roleId}, "ResourceId")

	var rids []uint64
	for _, v := range rrs {
		rids = append(rids, v.ResourceId)
	}
	return rids
}

func (m *roleResourceRepoImpl) GetRoleResources(roleId uint64, toEntity any) {
	sql := "SELECT rr.creator AS creator, rr.create_time AS CreateTime, rr.resource_id AS id, r.pid AS pid, " +
		"r.name AS name, r.type AS type, r.status AS status, r.meta " +
		"FROM t_sys_role_resource rr JOIN t_sys_resource r ON rr.resource_id = r.id " +
		"WHERE rr.role_id = ? AND rr.is_deleted = 0 AND r.is_deleted = 0 " +
		"ORDER BY r.pid ASC, r.weight ASC"
	m.SelectBySql(sql, toEntity, roleId)
}

package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type roleRepoImpl struct {
	base.RepoImpl[*entity.Role]
}

func newRoleRepo() repository.Role {
	return &roleRepoImpl{base.RepoImpl[*entity.Role]{M: new(entity.Role)}}
}

func (m *roleRepoImpl) GetPageList(condition *entity.RoleQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := gormx.NewQuery(new(entity.Role)).
		Like("name", condition.Name).
		Like("code", condition.Code).
		In("id", condition.Ids).
		NotIn("id", condition.NotIds).
		WithOrderBy(orderBy...)
	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (m *roleRepoImpl) ListByQuery(condition *entity.RoleQuery) ([]*entity.Role, error) {
	var res []*entity.Role
	qd := gormx.NewQuery(new(entity.Role)).
		Like("name", condition.Name).
		Like("code", condition.Code).
		In("id", condition.Ids).
		NotIn("id", condition.NotIds).
		OrderByDesc("id")
	err := gormx.ListByQueryCond(qd, &res)
	return res, err
}

// 获取角色拥有的资源id数组，从role_resource表获取
func (m *roleRepoImpl) GetRoleResourceIds(roleId uint64) []uint64 {
	var rrs []entity.RoleResource

	condtion := &entity.RoleResource{RoleId: roleId}
	gormx.ListBy(condtion, &rrs, "ResourceId")

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
		"WHERE rr.role_id = ? AND rr.is_deleted = 0 AND r.is_deleted = 0 " +
		"ORDER BY r.pid ASC, r.weight ASC"
	gormx.GetListBySql2Model(sql, toEntity, roleId)
}

func (m *roleRepoImpl) SaveRoleResource(rr []*entity.RoleResource) {
	gormx.BatchInsert[*entity.RoleResource](rr)
}

func (m *roleRepoImpl) DeleteRoleResource(roleId uint64, resourceId uint64) {
	gormx.DeleteBy(&entity.RoleResource{RoleId: roleId, ResourceId: resourceId})
}

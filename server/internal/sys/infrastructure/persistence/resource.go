package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
)

type resourceRepoImpl struct {
	base.RepoImpl[*entity.Resource]
}

func newResourceRepo() repository.Resource {
	return &resourceRepoImpl{
		base.RepoImpl[*entity.Resource]{M: new(entity.Resource)},
	}
}

func (r *resourceRepoImpl) GetChildren(uiPath string) []entity.Resource {
	sql := "SELECT id, ui_path FROM t_sys_resource WHERE ui_path LIKE ? AND is_deleted = 0"
	var rs []entity.Resource
	gormx.GetListBySql2Model(sql, &rs, uiPath+"%")
	return rs
}

func (r *resourceRepoImpl) UpdateByUiPathLike(resource *entity.Resource) error {
	sql := "UPDATE t_sys_resource SET status=? WHERE (ui_path LIKE ?)"
	return gormx.ExecSql(sql, resource.Status, resource.UiPath+"%")
}

func (r *resourceRepoImpl) GetAccountResources(accountId uint64, toEntity any) error {
	sql := `SELECT
	           m.id,
	           m.pid,
	           m.weight,
	           m.name,
	           m.code,
	           m.meta,
	           m.type,
	           m.status
            FROM
	           t_sys_resource m 
            WHERE
         	   m.status = 1 AND m.is_deleted = 0
	        AND m.id IN (
	            SELECT DISTINCT
		            ( rmb.resource_id ) 
	            FROM
		            t_sys_account_role p
		        JOIN t_sys_role r ON p.role_Id = r.id 
		        AND p.account_id = ? AND p.is_deleted = 0
		        AND r.STATUS = 1 AND r.is_deleted = 0
		        JOIN t_sys_role_resource rmb ON rmb.role_id = r.id AND rmb.is_deleted = 0 UNION
	            SELECT
		            r.id 
	            FROM
	             	t_sys_resource r
		        JOIN t_sys_role_resource rr ON r.id = rr.resource_id
		        JOIN t_sys_role ro ON rr.role_id = ro.id 
		        AND ro.status = 1 AND ro.code LIKE 'COMMON%' AND ro.is_deleted = 0 AND rr.is_deleted = 0
	        ) 
            ORDER BY
	        m.pid ASC,
	        m.weight ASC`
	return gormx.GetListBySql2Model(sql, toEntity, accountId)
}

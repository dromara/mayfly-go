package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/gormx"
)

type resourceRepoImpl struct{}

func newResourceRepo() repository.Resource {
	return new(resourceRepoImpl)
}

func (r *resourceRepoImpl) GetResourceList(condition *entity.Resource, toEntity any, orderBy ...string) {
	gormx.ListByOrder(condition, toEntity, orderBy...)
}

func (r *resourceRepoImpl) GetById(id uint64, cols ...string) *entity.Resource {
	res := new(entity.Resource)
	if err := gormx.GetById(res, id, cols...); err != nil {
		return nil

	}
	return res
}

func (r *resourceRepoImpl) Delete(id uint64) {
	biz.ErrIsNil(gormx.DeleteById(new(entity.Resource), id), "删除失败")
}

func (r *resourceRepoImpl) GetByCondition(condition *entity.Resource, cols ...string) error {
	return gormx.GetBy(condition, cols...)
}

func (r *resourceRepoImpl) GetChildren(uiPath string) []entity.Resource {
	sql := "SELECT id, ui_path FROM t_sys_resource WHERE ui_path LIKE ?"
	var rs []entity.Resource
	gormx.GetListBySql2Model(sql, &rs, uiPath+"%")
	return rs
}

func (r *resourceRepoImpl) UpdateByUiPathLike(resource *entity.Resource) {
	sql := "UPDATE t_sys_resource SET status=? WHERE (ui_path LIKE ?)"
	gormx.ExecSql(sql, resource.Status, resource.UiPath+"%")
}

func (r *resourceRepoImpl) GetAccountResources(accountId uint64, toEntity any) {
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
         	   m.status = 1 
	        AND m.id IN (
	            SELECT DISTINCT
		            ( rmb.resource_id ) 
	            FROM
		            t_sys_account_role p
		        JOIN t_sys_role r ON p.role_Id = r.id 
		        AND p.account_id = ?
		        AND r.STATUS = 1
		        JOIN t_sys_role_resource rmb ON rmb.role_id = r.id UNION
	            SELECT
		            r.id 
	            FROM
	             	t_sys_resource r
		        JOIN t_sys_role_resource rr ON r.id = rr.resource_id
		        JOIN t_sys_role ro ON rr.role_id = ro.id 
		        AND ro.status = 1 AND ro.code LIKE 'COMMON%' 
	        ) 
            ORDER BY
	        m.pid ASC,
	        m.weight ASC`
	biz.ErrIsNilAppendErr(gormx.GetListBySql2Model(sql, toEntity, accountId), "查询账号资源失败: %s")
}

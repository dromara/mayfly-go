package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/base"
)

type resourceRepoImpl struct {
	base.RepoImpl[*entity.Resource]
}

func newResourceRepo() repository.Resource {
	return &resourceRepoImpl{}
}

func (r *resourceRepoImpl) GetChildren(uiPath string) []entity.Resource {
	sql := "SELECT id, ui_path FROM t_sys_resource WHERE ui_path LIKE ? AND is_deleted = 0"
	var rs []entity.Resource
	r.SelectBySql(sql, &rs, uiPath+"%")
	return rs
}

func (r *resourceRepoImpl) UpdateByUiPathLike(resource *entity.Resource) error {
	sql := "UPDATE t_sys_resource SET status=? WHERE (ui_path LIKE ?)"
	return r.ExecBySql(sql, resource.Status, resource.UiPath+"%")
}

func (r *resourceRepoImpl) GetAccountResources(accountId uint64, toEntity any) error {
	sql := `
SELECT
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
JOIN (
  SELECT DISTINCT rmb.resource_id
  FROM
    t_sys_account_role p
    JOIN t_sys_role r ON p.role_Id = r.id
    JOIN t_sys_role_resource rmb ON rmb.role_id = r.id
  WHERE
    p.account_id = ?
    AND p.is_deleted = 0
    AND r.status = 1
    AND r.is_deleted = 0
    AND rmb.is_deleted = 0
  UNION
  SELECT
    r.id
  FROM
    t_sys_resource r
    JOIN t_sys_role_resource rr ON r.id = rr.resource_id
    JOIN t_sys_role ro ON rr.role_id = ro.id
  WHERE
    ro.status = 1
    AND ro.code LIKE 'COMMON%'
    AND ro.is_deleted = 0
    AND rr.is_deleted = 0
) AS subquery ON m.id = subquery.resource_id
WHERE
  m.status = 1
  AND m.is_deleted = 0
ORDER BY
  m.pid ASC,
  m.weight ASC;`
	return r.SelectBySql(sql, toEntity, accountId)
}

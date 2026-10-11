package persistence

import (
	"context"
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type resourceRepoImpl struct {
	base.RepoImpl[*entity.Resource]
}

var _ repository.Resource = (*resourceRepoImpl)(nil)

func newResourceRepo() repository.Resource {
	return &resourceRepoImpl{}
}

func (r *resourceRepoImpl) GetChildren(ctx context.Context, uiPath string) ([]entity.Resource, error) {
	// 表名取自实体：物理表名由 NamingStrategy 统一派生，写死后改前缀或换方言就会静默查错表
	sql := "SELECT id, ui_path FROM " + (&entity.Resource{}).TableName() + " WHERE ui_path LIKE ? AND is_deleted = 0"
	var rs []entity.Resource
	if err := r.SelectBySQL(ctx, sql, &rs, uiPath+"%"); err != nil {
		return nil, err
	}
	return rs, nil
}

// UpdateByUiPathLike 按唯一标识路径前缀批量变更状态（停用/启用父节点时级联其整棵子树）
func (r *resourceRepoImpl) UpdateByUiPathLike(ctx context.Context, resource *entity.Resource) error {
	_, err := r.UpdateColsByCond(ctx, base.NewPatch().Set("status", resource.Status), model.NewCond().RLike("ui_path", resource.UiPath))
	return err
}

func (r *resourceRepoImpl) GetAccountResources(ctx context.Context, accountId uint64, toEntity any) error {
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
	return r.SelectBySQL(ctx, sql, toEntity, accountId)
}

package persistence

import (
	"context"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
	"slices"
)

var _ repository.DbRestore = (*dbRestoreRepoImpl)(nil)

type dbRestoreRepoImpl struct {
	dbJobBaseImpl[*entity.DbRestore]
}

func NewDbRestoreRepo() repository.DbRestore {
	return &dbRestoreRepoImpl{}
}

func (d *dbRestoreRepoImpl) GetDbNamesWithoutRestore(instanceId uint64, dbNames []string) ([]string, error) {
	var dbNamesWithRestore []string
	err := global.Db.Model(d.GetModel()).
		Where("db_instance_id = ?", instanceId).
		Where("repeated = ?", true).
		Scopes(gormx.UndeleteScope).
		Pluck("db_name", &dbNamesWithRestore).
		Error
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(dbNames))
	for _, name := range dbNames {
		if !slices.Contains(dbNamesWithRestore, name) {
			result = append(result, name)
		}
	}
	return result, nil
}

func (d *dbRestoreRepoImpl) ListToDo(jobs any) error {
	db := global.Db.Model(d.GetModel())
	err := db.Where("enabled = ?", true).
		Where(db.Where("repeated = ?", true).Or("last_status <> ?", entity.DbJobSuccess)).
		Scopes(gormx.UndeleteScope).
		Find(jobs).Error
	if err != nil {
		return err
	}
	return nil
}

// GetPageList 分页获取数据库备份任务列表
func (d *dbRestoreRepoImpl) GetPageList(condition *entity.DbRestoreQuery, pageParam *model.PageParam, toEntity any, _ ...string) (*model.PageResult[any], error) {
	qd := gormx.NewQuery(d.GetModel()).
		Eq("id", condition.Id).
		Eq0("db_instance_id", condition.DbInstanceId).
		Eq0("repeated", condition.Repeated).
		In0("db_name", condition.InDbNames).
		Like("db_name", condition.DbName)
	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (d *dbRestoreRepoImpl) GetEnabledRestores(toEntity any, backupHistoryId ...uint64) error {
	return global.Db.Model(d.GetModel()).
		Select("id", "db_backup_history_id", "last_status", "last_result", "last_time").
		Where("db_backup_history_id in ?", backupHistoryId).
		Where("enabled = true").
		Scopes(gormx.UndeleteScope).
		Order("id DESC").
		Find(toEntity).
		Error
}

// AddJob 添加数据库任务
func (d *dbRestoreRepoImpl) AddJob(ctx context.Context, jobs any) error {
	return addJob[*entity.DbRestore](ctx, d.dbJobBaseImpl, jobs)
}

func (d *dbRestoreRepoImpl) UpdateEnabled(_ context.Context, jobId uint64, enabled bool) error {
	cond := map[string]any{
		"id": jobId,
	}
	desc := "已禁用"
	if enabled {
		desc = "已启用"
	}
	return d.Updates(cond, map[string]any{
		"enabled":      enabled,
		"enabled_desc": desc,
	})
}

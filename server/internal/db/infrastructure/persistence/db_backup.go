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

var _ repository.DbBackup = (*dbBackupRepoImpl)(nil)

type dbBackupRepoImpl struct {
	dbJobBaseImpl[*entity.DbBackup]
}

func NewDbBackupRepo() repository.DbBackup {
	return &dbBackupRepoImpl{}
}

func (d *dbBackupRepoImpl) GetDbNamesWithoutBackup(instanceId uint64, dbNames []string) ([]string, error) {
	var dbNamesWithBackup []string
	err := global.Db.Model(d.NewModel()).
		Where("db_instance_id = ?", instanceId).
		Where("repeated = ?", true).
		Scopes(gormx.UndeleteScope).
		Pluck("db_name", &dbNamesWithBackup).
		Error
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(dbNames))
	for _, name := range dbNames {
		if !slices.Contains(dbNamesWithBackup, name) {
			result = append(result, name)
		}
	}
	return result, nil
}

func (d *dbBackupRepoImpl) ListDbInstances(enabled bool, repeated bool, instanceIds *[]uint64) error {
	return global.Db.Model(d.NewModel()).
		Where("enabled = ?", enabled).
		Where("repeated = ?", repeated).
		Scopes(gormx.UndeleteScope).
		Distinct().
		Pluck("db_instance_id", &instanceIds).
		Error
}

func (d *dbBackupRepoImpl) ListToDo(jobs any) error {
	db := global.Db.Model(d.NewModel())
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
func (d *dbBackupRepoImpl) GetPageList(condition *entity.DbBackupQuery, pageParam *model.PageParam, toEntity any, _ ...string) (*model.PageResult[any], error) {
	qd := model.NewCond().
		Eq("id", condition.Id).
		Eq0("db_instance_id", condition.DbInstanceId).
		Eq0("repeated", condition.Repeated).
		In0("db_name", condition.InDbNames).
		Like("db_name", condition.DbName)
	return d.PageByCondToAny(qd, pageParam, toEntity)
}

// AddJob 添加数据库任务
func (d *dbBackupRepoImpl) AddJob(ctx context.Context, jobs any) error {
	return addJob[*entity.DbBackup](ctx, d.dbJobBaseImpl, jobs)
}

func (d *dbBackupRepoImpl) UpdateEnabled(ctx context.Context, jobId uint64, enabled bool) error {
	cond := map[string]any{
		"id": jobId,
	}
	desc := "已禁用"
	if enabled {
		desc = "已启用"
	}
	return d.UpdateByCond(ctx, map[string]any{
		"enabled":      enabled,
		"enabled_desc": desc,
	}, cond)
}

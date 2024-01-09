package persistence

import (
	"context"
	"errors"
	"fmt"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
	"slices"

	"gorm.io/gorm"
)

var _ repository.DbRestore = (*dbRestoreRepoImpl)(nil)

type dbRestoreRepoImpl struct {
	dbTaskBase[*entity.DbRestore]
}

func NewDbRestoreRepo() repository.DbRestore {
	return &dbRestoreRepoImpl{}
}

// GetDbRestoreList 分页获取数据库备份任务列表
func (d *dbRestoreRepoImpl) GetDbRestoreList(condition *entity.DbRestoreQuery, pageParam *model.PageParam, toEntity any, _ ...string) (*model.PageResult[any], error) {
	qd := gormx.NewQuery(d.GetModel()).
		Eq("id", condition.Id).
		Eq0("db_instance_id", condition.DbInstanceId).
		Eq0("repeated", condition.Repeated).
		In0("db_name", condition.InDbNames).
		Like("db_name", condition.DbName)
	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (d *dbRestoreRepoImpl) AddTask(ctx context.Context, tasks ...*entity.DbRestore) error {
	return gormx.Tx(func(db *gorm.DB) error {
		var instanceId uint64
		dbNames := make([]string, 0, len(tasks))
		for _, task := range tasks {
			if instanceId == 0 {
				instanceId = task.DbInstanceId
			}
			if task.DbInstanceId != instanceId {
				return errors.New("不支持同时为多个数据库实例添加备份任务")
			}
			if task.Interval == 0 {
				// 单次执行的恢复任务可重复创建
				continue
			}
			dbNames = append(dbNames, task.DbName)
		}
		var res []string
		err := db.Model(new(entity.DbRestore)).Select("db_name").
			Where("db_instance_id = ?", instanceId).
			Where("db_name in ?", dbNames).
			Where("repeated = true").
			Scopes(gormx.UndeleteScope).Find(&res).Error
		if err != nil {
			return err
		}
		if len(res) > 0 {
			return fmt.Errorf("数据库备份任务已存在: %v", res)
		}

		return d.BatchInsertWithDb(ctx, db, tasks)
	})
}

func (d *dbRestoreRepoImpl) GetDbNamesWithoutRestore(instanceId uint64, dbNames []string) ([]string, error) {
	var dbNamesWithRestore []string
	query := gormx.NewQuery(d.GetModel()).
		Eq("db_instance_id", instanceId).
		Eq("repeated", true)
	if err := query.GenGdb().Pluck("db_name", &dbNamesWithRestore).Error; err != nil {
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

package persistence

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
	"reflect"
)

type dbJobBase[T entity.DbJob] struct {
	base.RepoImpl[T]
}

func (d *dbJobBase[T]) GetById(e entity.DbJob, id uint64, cols ...string) error {
	return d.RepoImpl.GetById(e.(T), id, cols...)
}

func (d *dbJobBase[T]) UpdateById(ctx context.Context, e entity.DbJob, columns ...string) error {
	return d.RepoImpl.UpdateById(ctx, e.(T), columns...)
}

func (d *dbJobBase[T]) UpdateEnabled(_ context.Context, jobId uint64, enabled bool) error {
	cond := map[string]any{
		"id": jobId,
	}
	return d.Updates(cond, map[string]any{
		"enabled": enabled,
	})
}

func (d *dbJobBase[T]) UpdateLastStatus(ctx context.Context, job entity.DbJob) error {
	return d.UpdateById(ctx, job.(T), "last_status", "last_result", "last_time")
}

func (d *dbJobBase[T]) ListToDo(jobs any) error {
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

func (d *dbJobBase[T]) ListRepeating(jobs any) error {
	cond := map[string]any{
		"enabled":  true,
		"repeated": true,
	}
	if err := d.ListByCond(cond, jobs); err != nil {
		return err
	}
	return nil
}

// GetPageList 分页获取数据库备份任务列表
func (d *dbJobBase[T]) GetPageList(condition *entity.DbJobQuery, pageParam *model.PageParam, toEntity any, _ ...string) (*model.PageResult[any], error) {
	d.GetModel()
	qd := gormx.NewQuery(d.GetModel()).
		Eq("id", condition.Id).
		Eq0("db_instance_id", condition.DbInstanceId).
		Eq0("repeated", condition.Repeated).
		In0("db_name", condition.InDbNames).
		Like("db_name", condition.DbName)
	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (d *dbJobBase[T]) AddJob(ctx context.Context, jobs any) error {
	return gormx.Tx(func(db *gorm.DB) error {
		var instanceId uint64
		var dbNames []string
		reflectValue := reflect.ValueOf(jobs)
		var plural bool
		switch reflectValue.Kind() {
		case reflect.Slice, reflect.Array:
			plural = true
			reflectLen := reflectValue.Len()
			dbNames = make([]string, 0, reflectLen)
			for i := 0; i < reflectLen; i++ {
				job := reflectValue.Index(i).Interface().(entity.DbJob)
				jobBase := job.GetJobBase()
				if instanceId == 0 {
					instanceId = jobBase.DbInstanceId
				}
				if jobBase.DbInstanceId != instanceId {
					return errors.New("不支持同时为多个数据库实例添加数据库任务")
				}
				if jobBase.Interval == 0 {
					// 单次执行的数据库任务可重复创建
					continue
				}
				dbNames = append(dbNames, jobBase.DbName)
			}
		default:
			jobBase := jobs.(entity.DbJob).GetJobBase()
			instanceId = jobBase.DbInstanceId
			if jobBase.Interval > 0 {
				dbNames = append(dbNames, jobBase.DbName)
			}
		}

		var res []string
		err := db.Model(d.GetModel()).Select("db_name").
			Where("db_instance_id = ?", instanceId).
			Where("db_name in ?", dbNames).
			Where("repeated = true").
			Scopes(gormx.UndeleteScope).Find(&res).Error
		if err != nil {
			return err
		}
		if len(res) > 0 {
			return errors.New(fmt.Sprintf("数据库任务已存在: %v", res))
		}
		if plural {
			return d.BatchInsertWithDb(ctx, db, jobs)
		}
		return d.InsertWithDb(ctx, db, jobs.(T))
	})
}

package persistence

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"reflect"
)

var _ repository.DbJobBase[entity.DbJob] = (*dbJobBaseImpl[entity.DbJob])(nil)

type dbJobBaseImpl[T entity.DbJob] struct {
	base.RepoImpl[T]
}

func (d *dbJobBaseImpl[T]) UpdateLastStatus(ctx context.Context, job entity.DbJob) error {
	return d.UpdateById(ctx, job.(T), "last_status", "last_result", "last_time")
}

func addJob[T entity.DbJob](ctx context.Context, repo dbJobBaseImpl[T], jobs any) error {
	// refactor jobs from any to []T
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
				if instanceId == 0 {
					instanceId = job.GetInstanceId()
				}
				if job.GetInstanceId() != instanceId {
					return errors.New("不支持同时为多个数据库实例添加数据库任务")
				}
				if job.GetInterval() == 0 {
					// 单次执行的数据库任务可重复创建
					continue
				}
				dbNames = append(dbNames, job.GetDbName())
			}
		default:
			job := jobs.(entity.DbJob)
			instanceId = job.GetInstanceId()
			if job.GetInterval() > 0 {
				dbNames = append(dbNames, job.GetDbName())
			}
		}

		var res []string
		err := db.Model(repo.GetModel()).Select("db_name").
			Where("db_instance_id = ?", instanceId).
			Where("db_name in ?", dbNames).
			Where("repeated = true").
			Scopes(gormx.UndeleteScope).
			Find(&res).Error
		if err != nil {
			return err
		}
		if len(res) > 0 {
			return errors.New(fmt.Sprintf("数据库任务已存在: %v", res))
		}
		if plural {
			return repo.BatchInsertWithDb(ctx, db, jobs.([]T))
		}
		return repo.InsertWithDb(ctx, db, jobs.(T))
	})
}

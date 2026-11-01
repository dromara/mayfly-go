package persistence

import (
	"context"
	"time"

	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type dbMetricSampleRepoImpl struct {
	base.RepoImpl[*entity.DbMetricSample]
}

var _ repository.DbMetricSample = (*dbMetricSampleRepoImpl)(nil)

func newDbMetricSampleRepo() repository.DbMetricSample {
	return &dbMetricSampleRepoImpl{}
}

// Series 取时间窗内的采样点
func (r *dbMetricSampleRepoImpl) Series(instanceId uint64, metric string, since time.Time) ([]*entity.DbMetricSample, error) {
	return r.SelectByCond(model.NewCond().
		Eq("instance_id", instanceId).
		Eq("metric", metric).
		Ge("sample_time", since).
		OrderByAsc("sample_time"))
}

// DeleteBefore 物理删除早期采样点
func (r *dbMetricSampleRepoImpl) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	result := gormx.DB(ctx).Unscoped().Where("sample_time < ?", before).Delete(&entity.DbMetricSample{})
	return result.RowsAffected, result.Error
}

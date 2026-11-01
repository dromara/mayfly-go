package repository

import (
	"context"
	"time"

	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
)

type DbMetricSample interface {
	base.Repo[*entity.DbMetricSample]

	// Series 取某实例某指标自 since 起的采样点（按时间升序）
	Series(instanceId uint64, metric string, since time.Time) ([]*entity.DbMetricSample, error)

	// DeleteBefore 物理删除早期采样：指标点没有审计价值，留着只会让表无限增长
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)
}

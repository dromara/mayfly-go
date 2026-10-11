package persistence

import (
	"context"

	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type schemaBaselineRepoImpl struct {
	base.RepoImpl[*entity.DbSchemaBaseline]
}

var _ repository.SchemaBaseline = (*schemaBaselineRepoImpl)(nil)

func newSchemaBaselineRepo() repository.SchemaBaseline {
	return &schemaBaselineRepoImpl{}
}

func (s *schemaBaselineRepoImpl) ListByDb(dbId uint64, dbName string) ([]*entity.DbSchemaBaseline, error) {
	return s.SelectByCond(&entity.DbSchemaBaseline{DbId: dbId, Db: dbName})
}

// ListDbKeys 只取定位所需的三列（基线行可能上千条，把指纹与明细一起拉回内存没有意义）；
// 去重由调用方做，不在 SQL 里拼 DISTINCT，避开把原始片段塞进 Select 子句
func (s *schemaBaselineRepoImpl) ListDbKeys() ([]*entity.DbSchemaBaseline, error) {
	return s.SelectByCond(model.NewCond().Columns("db_id", "db", "instance_id"))
}

func (s *schemaBaselineRepoImpl) DriftStat(instanceId uint64) (int64, int64, error) {
	baselineRows, err := s.CountByCond(model.NewCond().Eq("instance_id", instanceId))
	if err != nil {
		return 0, 0, err
	}
	driftedRows, err := s.CountByCond(model.NewCond().Eq("instance_id", instanceId).Eq("drifted", true))
	if err != nil {
		return 0, 0, err
	}
	return baselineRows, driftedRows, nil
}

func (s *schemaBaselineRepoImpl) DeleteByDb(ctx context.Context, dbId uint64, dbName string) error {
	return s.DeleteByCond(ctx, model.NewCond().Eq("db_id", dbId).Eq("db", dbName))
}

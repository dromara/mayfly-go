package persistence

import (
	"context"

	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/base"
)

type machineMetricRepoImpl struct {
	base.RepoImpl[*entity.MachineMetric]
}

var _ repository.MachineMetric = (*machineMetricRepoImpl)(nil)

func newMachineMetricRepo() repository.MachineMetric {
	return &machineMetricRepoImpl{}
}

// LatestPerMachine 取每台机器最新采集点：inner join 每机 max(collect_time)
func (m *machineMetricRepoImpl) LatestPerMachine(ctx context.Context) ([]*entity.MachineMetric, error) {
	var list []*entity.MachineMetric
	// 表名取自实体：物理表名由 NamingStrategy 统一派生，写死后改前缀或换方言就会静默查错表
	tableName := (&entity.MachineMetric{}).TableName()
	sql := `SELECT mm.* FROM ` + tableName + ` mm
		INNER JOIN (SELECT machine_id, MAX(collect_time) mt FROM ` + tableName + ` GROUP BY machine_id) t
		ON mm.machine_id = t.machine_id AND mm.collect_time = t.mt`
	if err := m.SelectBySQL(ctx, sql, &list); err != nil {
		return nil, err
	}
	return list, nil
}

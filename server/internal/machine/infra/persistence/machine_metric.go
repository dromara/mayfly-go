package persistence

import (
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
func (m *machineMetricRepoImpl) LatestPerMachine() ([]*entity.MachineMetric, error) {
	var list []*entity.MachineMetric
	sql := `SELECT mm.* FROM t_machine_metric mm
		INNER JOIN (SELECT machine_id, MAX(collect_time) mt FROM t_machine_metric GROUP BY machine_id) t
		ON mm.machine_id = t.machine_id AND mm.collect_time = t.mt`
	if err := m.SelectBySQL(sql, &list); err != nil {
		return nil, err
	}
	return list, nil
}

package persistence

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
)

type machineCronJobRelateRepoImpl struct {
	base.RepoImpl[*entity.MachineCronJobRelate]
}

func newMachineCronJobRelateRepo() repository.MachineCronJobRelate {
	return &machineCronJobRelateRepoImpl{base.RepoImpl[*entity.MachineCronJobRelate]{M: new(entity.MachineCronJobRelate)}}
}

func (m *machineCronJobRelateRepoImpl) GetList(condition *entity.MachineCronJobRelate) []entity.MachineCronJobRelate {
	list := new([]entity.MachineCronJobRelate)
	m.ListByCond(condition, list)
	return *list
}

func (m *machineCronJobRelateRepoImpl) GetMachineIds(cronJobId uint64) []uint64 {
	var machineIds []uint64
	m.ListByCond(&entity.MachineCronJobRelate{CronJobId: cronJobId}, &machineIds, "machine_id")
	return machineIds
}

func (m *machineCronJobRelateRepoImpl) GetCronJobIds(machineId uint64) []uint64 {
	var cronJobIds []uint64
	gormx.ListBy(&entity.MachineCronJobRelate{MachineId: machineId}, &cronJobIds, "cron_job_id")
	return cronJobIds
}

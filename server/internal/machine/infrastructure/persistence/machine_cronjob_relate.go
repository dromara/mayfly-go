package persistence

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type machineCronJobRelateRepoImpl struct {
	base.RepoImpl[*entity.MachineCronJobRelate]
}

func newMachineCronJobRelateRepo() repository.MachineCronJobRelate {
	return &machineCronJobRelateRepoImpl{base.RepoImpl[*entity.MachineCronJobRelate]{M: new(entity.MachineCronJobRelate)}}
}

func (m *machineCronJobRelateRepoImpl) GetList(condition *entity.MachineCronJobRelate) []entity.MachineCronJobRelate {
	list := new([]entity.MachineCronJobRelate)
	m.SelectByCond(condition, list)
	return *list
}

func (m *machineCronJobRelateRepoImpl) GetMachineIds(cronJobId uint64) []uint64 {
	var machineIds []uint64
	m.SelectByCond(model.NewModelCond(&entity.MachineCronJobRelate{CronJobId: cronJobId}).Columns("machine_id"), &machineIds)
	return machineIds
}

func (m *machineCronJobRelateRepoImpl) GetCronJobIds(machineId uint64) []uint64 {
	var cronJobIds []uint64
	m.SelectByCond(model.NewModelCond(&entity.MachineCronJobRelate{MachineId: machineId}).Columns("cron_job_id"), &cronJobIds)
	return cronJobIds
}

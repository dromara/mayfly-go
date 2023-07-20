package persistence

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/gormx"
)

type machineCropJobRelateRepoImpl struct{}

func newMachineCropJobRelateRepo() repository.MachineCronJobRelate {
	return new(machineCropJobRelateRepoImpl)
}

func (m *machineCropJobRelateRepoImpl) GetList(condition *entity.MachineCronJobRelate) []entity.MachineCronJobRelate {
	list := new([]entity.MachineCronJobRelate)
	gormx.ListByOrder(condition, list)
	return *list
}

func (m *machineCropJobRelateRepoImpl) GetMachineIds(cronJobId uint64) []uint64 {
	var machineIds []uint64
	gormx.ListBy(&entity.MachineCronJobRelate{CronJobId: cronJobId}, &machineIds, "machine_id")
	return machineIds
}

func (m *machineCropJobRelateRepoImpl) GetCronJobIds(machineId uint64) []uint64 {
	var cronJobIds []uint64
	gormx.ListBy(&entity.MachineCronJobRelate{MachineId: machineId}, &cronJobIds, "cron_job_id")
	return cronJobIds
}

func (m *machineCropJobRelateRepoImpl) Delete(condition *entity.MachineCronJobRelate) {
	gormx.DeleteByCondition(condition)
}

func (m *machineCropJobRelateRepoImpl) BatchInsert(mcjrs []*entity.MachineCronJobRelate) {
	gormx.BatchInsert(mcjrs)
}

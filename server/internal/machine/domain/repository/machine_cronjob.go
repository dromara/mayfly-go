package repository

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/pkg/model"
)

type MachineCronJob interface {
	GetPageList(condition *entity.MachineCronJob, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any]

	// 根据条件获取
	GetBy(condition *entity.MachineCronJob, cols ...string) error

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.MachineCronJob

	Delete(id uint64)

	Insert(entity *entity.MachineCronJob)

	UpdateById(entity *entity.MachineCronJob)
}

type MachineCronJobRelate interface {
	GetList(condition *entity.MachineCronJobRelate) []entity.MachineCronJobRelate

	GetMachineIds(cronJobId uint64) []uint64

	GetCronJobIds(machineId uint64) []uint64

	Delete(condition *entity.MachineCronJobRelate)

	BatchInsert(mcjrs []*entity.MachineCronJobRelate)
}

type MachineCronJobExec interface {
	GetPageList(condition *entity.MachineCronJobExec, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any]

	Insert(mcje *entity.MachineCronJobExec)

	Delete(m *entity.MachineCronJobExec)
}

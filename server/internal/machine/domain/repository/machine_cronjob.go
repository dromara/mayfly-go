package repository

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type MachineCronJob interface {
	base.Repo[*entity.MachineCronJob]

	GetPageList(condition *entity.MachineCronJob, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}

type MachineCronJobRelate interface {
	base.Repo[*entity.MachineCronJobRelate]

	GetList(condition *entity.MachineCronJobRelate) []entity.MachineCronJobRelate

	GetMachineIds(cronJobId uint64) []uint64

	GetCronJobIds(machineId uint64) []uint64
}

type MachineCronJobExec interface {
	base.Repo[*entity.MachineCronJobExec]

	GetPageList(condition *entity.MachineCronJobExec, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)
}

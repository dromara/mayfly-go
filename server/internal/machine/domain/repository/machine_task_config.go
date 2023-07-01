package repository

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/pkg/model"
)

type MachineTaskConfig interface {
	GetPageList(condition *entity.MachineTaskConfig, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any]

	// 根据条件获取
	GetBy(condition *entity.MachineTaskConfig, cols ...string) error

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.MachineTaskConfig

	Delete(id uint64)

	Create(entity *entity.MachineTaskConfig)

	UpdateById(entity *entity.MachineTaskConfig)
}

package repository

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/pkg/model"
)

type Machine interface {
	// 分页获取机器信息列表
	GetMachineList(condition *entity.Machine, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	Count(condition *entity.Machine) int64

	// 根据条件获取账号信息
	GetMachine(condition *entity.Machine, cols ...string) error

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.Machine

	Create(entity *entity.Machine)

	UpdateById(entity *entity.Machine)
}

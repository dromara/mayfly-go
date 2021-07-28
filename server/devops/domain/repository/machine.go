package repository

import (
	"mayfly-go/base/model"
	"mayfly-go/server/devops/domain/entity"
)

type Machine interface {
	// 分页获取机器信息列表
	GetMachineList(condition *entity.Machine, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	// 根据条件获取账号信息
	GetMachine(condition *entity.Machine, cols ...string) error

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.Machine

	Create(entity *entity.Machine)

	UpdateById(entity *entity.Machine)
}

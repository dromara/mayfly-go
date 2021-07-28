package repository

import (
	"mayfly-go/base/model"
	"mayfly-go/server/devops/domain/entity"
)

type MachineScript interface {
	// 分页获取机器脚本信息列表
	GetPageList(condition *entity.MachineScript, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	// 根据条件获取
	GetMachineScript(condition *entity.MachineScript, cols ...string) error

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.MachineScript

	Delete(id uint64)

	Create(entity *entity.MachineScript)

	UpdateById(entity *entity.MachineScript)
}

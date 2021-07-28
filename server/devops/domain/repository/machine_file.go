package repository

import (
	"mayfly-go/base/model"
	"mayfly-go/server/devops/domain/entity"
)

type MachineFile interface {
	// 分页获取机器脚本信息列表
	GetPageList(condition *entity.MachineFile, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult

	// 根据条件获取
	GetMachineFile(condition *entity.MachineFile, cols ...string) error

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.MachineFile

	Delete(id uint64)

	Create(entity *entity.MachineFile)

	UpdateById(entity *entity.MachineFile)
}

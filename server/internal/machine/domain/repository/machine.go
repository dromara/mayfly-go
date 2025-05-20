package repository

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type Machine interface {
	base.Repo[*entity.Machine]

	// 分页获取机器信息列表
	GetMachineList(condition *entity.MachineQuery, orderBy ...string) (*model.PageResult[*entity.Machine], error)
}

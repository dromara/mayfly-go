package repository

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type MachineScript interface {
	base.Repo[*entity.MachineScript]

	// 分页获取机器脚本信息列表
	GetPageList(condition *entity.MachineScript, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MachineScript], error)
}
